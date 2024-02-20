package orm

import (
	"context"
	"fmt"
	"sort"

	"github.com/stephenafamo/bob"
)

type RelWhere struct {
	Column   string `yaml:"column"`
	SQLValue string `yaml:"sql_value"`
	GoValue  string `yaml:"go_value"`
}

type RelSide struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`

	// To make sure the column lengths match and are in the right order,
	// a slice of tupules is expected.
	// bobgen-helpers.GenConfig has a function to spread that into From/ToColumns
	Columns     [][2]string `yaml:"columns"`
	FromColumns []string    `yaml:"-"`
	ToColumns   []string    `yaml:"-"`

	FromWhere []RelWhere `yaml:"from_where"`
	ToWhere   []RelWhere `yaml:"to_where"`

	// These are columns that exist in the database, but should not be
	// considered by the relationship when determining if it is a join table
	// the columns are never set or read, so make sure they have a default value
	// or operations will fail
	// the first slice is for the from table, the second is for the to table
	IgnoredColumns [2][]string `yaml:"-"`

	//--------------------------------------------
	// The Uniques are set in Relationships.init()
	//--------------------------------------------

	// if the origin is unique
	FromUnique bool `yaml:"-"`
	// if the destination is unique
	ToUnique bool `yaml:"-"`

	// Which side to modify, "from" or "to"
	// If not set, it will try to "guess" which side to modify
	// - if only one of the sides contains a primary key,
	//   it will choose to modify the other side
	// - If (both or none) of them contains a primary key,
	//   it will try with "Unique" columns
	// - If it still cannot choose, it defaults to "to"
	Modify string `yaml:"modify"`

	// If the key is nullable. We need this to know if we can remove the
	// relationship without deleting it
	// this is set in Relationships.init()
	KeyNullable bool `yaml:"-"`

	// Kinda hacky, used for preloading
	ToExpr func(context.Context) bob.Expression `json:"-" yaml:"-"`
}

type Relationship struct {
	Name  string    `yaml:"name"`
	Sides []RelSide `yaml:"sides"`

	// These can be set through user configuration
	Ignored bool
	// Do not create the inverse of a user configured relationship
	NoReverse bool `yaml:"no_reverse"`
	// Makes sure the factories does not require the relationship to be set.
	// Useful if you're not using foreign keys
	NeverRequired bool `yaml:"never_required"`
}

func (r Relationship) Validate() error {
	for index, side := range r.Sides {
		for _, where := range append(side.FromWhere, side.ToWhere...) {
			if where.Column == "" {
				return fmt.Errorf("rel %s has a where clause with an empty column", r.Name)
			}

			if where.SQLValue == "" {
				return fmt.Errorf("rel %s has a where clause with an empty SQL value", r.Name)
			}

			if where.GoValue == "" {
				return fmt.Errorf("rel %s has a where clause with an empty Go value", r.Name)
			}
		}

		// Only compare from/to tables if it is not the first side
		if index == 0 {
			continue
		}

		if r.Sides[index-1].To != side.From {
			return fmt.Errorf("rel %s has a gap between %s and %s", r.Name, r.Sides[index-1].To, r.Sides[index].From)
		}
	}

	return nil
}

func (r Relationship) Local() string {
	return r.Sides[0].From
}

func (r Relationship) LocalPosition() int {
	return 0
}

func (r Relationship) Foreign() string {
	return r.Sides[len(r.Sides)-1].To
}

func (r Relationship) ForeignPosition() int {
	return len(r.Sides)
}

func (r Relationship) IsToMany() bool {
	// If the modifiable part of a side is not unique, it is to-many
	for i := 0; i < len(r.Sides); i++ {
		if r.Sides[i].Modify == "to" && !r.Sides[i].ToUnique {
			return true
		}
	}

	return false
}

func (r Relationship) IsRemovable() bool {
	return false
}

func (r Relationship) InsertEarly() bool {
	for _, mapping := range r.ValuedSides() {
		if mapping.End {
			return false
		}
	}

	return true
}

type RelSetDetails struct {
	TableName string
	Mapped    []RelSetMapping
	Position  int
	Start     bool
	End       bool
}

type RelSetMapping struct {
	Column         string
	Value          [2]string // [0] is the SQL value, [1] is the Go value
	ExternalTable  string
	ExternalColumn string
	ExtPosition    int
	ExternalStart  bool
	ExternalEnd    bool
}

func (r RelSetMapping) HasValue() bool {
	return r.Value[0] != ""
}

func (r RelSetDetails) Columns() []string {
	cols := make([]string, len(r.Mapped))
	for i, m := range r.Mapped {
		cols[i] = m.Column
	}

	return cols
}

// NeedsMany returns true if the table on this side needs to be many
func (r Relationship) NeedsMany(position int) bool {
	if position == 0 {
		return false
	}

	// If it is the last side, then it needs to be many
	// if any items are many
	if position == len(r.Sides) {
		return r.IsToMany()
	}

	// If there is another "Many" side down the line
	// this this should not be many
	for i := position; i < len(r.Sides); i++ {
		if r.Sides[i].Modify == "to" && !r.Sides[i].ToUnique {
			return false
		}
	}

	// If this or any others up the line are many, it should be many
	for i := position - 1; i >= 0; i-- {
		if r.Sides[i].Modify == "to" && !r.Sides[i].ToUnique {
			return true
		}
	}

	return false
}

func (r RelSetDetails) UniqueExternals() []RelSetMapping {
	found := make(map[int]struct{})
	ma := []RelSetMapping{}

	for _, ext := range r.Mapped {
		if ext.ExternalTable == "" {
			continue
		}
		if _, ok := found[ext.ExtPosition]; ok {
			continue
		}

		ma = append(ma, ext)
		found[ext.ExtPosition] = struct{}{}
	}

	return ma
}

func (r Relationship) ValuedSides() []RelSetDetails {
	valuedSides := make([]RelSetDetails, 0, len(r.Sides))

	for sideIndex, side := range r.Sides {
		fromDeets := RelSetDetails{
			TableName: side.From,
			Mapped:    make([]RelSetMapping, 0, len(side.FromColumns)+len(side.FromWhere)),
			Start:     sideIndex == 0,
			Position:  sideIndex,
		}

		toDeets := RelSetDetails{
			TableName: side.To,
			Mapped:    make([]RelSetMapping, 0, len(side.ToColumns)+len(side.ToWhere)),
			End:       sideIndex == (len(r.Sides) - 1),
			Position:  sideIndex + 1,
		}

		if len(side.FromWhere) > 0 {
			for _, f := range side.FromWhere {
				fromDeets.Mapped = append(fromDeets.Mapped, RelSetMapping{
					Column: f.Column,
					Value:  [2]string{f.SQLValue, f.GoValue},
				})
			}
		}

		if len(side.ToWhere) > 0 {
			for _, f := range side.ToWhere {
				toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
					Column: f.Column,
					Value:  [2]string{f.SQLValue, f.GoValue},
				})
			}
		}

		//nolint:nestif
		if side.Modify == "from" {
			if sideIndex == 0 || r.Sides[sideIndex-1].Modify == "from" {
				for i, f := range side.FromColumns {
					fromDeets.Mapped = append(fromDeets.Mapped, RelSetMapping{
						Column:         f,
						ExternalTable:  side.To,
						ExternalColumn: side.ToColumns[i],
						ExternalEnd:    sideIndex == (len(r.Sides) - 1),
						ExtPosition:    sideIndex + 1,
					})
				}
			}
		} else {
			for i, f := range side.FromColumns {
				toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
					Column:         side.ToColumns[i],
					ExternalTable:  side.From,
					ExternalColumn: f,
					ExternalStart:  sideIndex == 0,
					ExtPosition:    sideIndex,
				})
			}

			if len(r.Sides) > sideIndex+1 {
				nextSide := r.Sides[sideIndex+1]
				if nextSide.Modify == "from" {
					for i, f := range nextSide.FromColumns {
						toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
							Column:         f,
							ExternalTable:  nextSide.To,
							ExternalColumn: nextSide.ToColumns[i],
							ExternalEnd:    (sideIndex + 1) == (len(r.Sides) - 1),
							ExtPosition:    sideIndex + 2,
						})
					}
				}
			}
		}

		if len(fromDeets.Mapped) > 0 {
			valuedSides = append(valuedSides, fromDeets)
		}
		if len(toDeets.Mapped) > 0 {
			valuedSides = append(valuedSides, toDeets)
		}
	}

	sort.Slice(valuedSides, func(i, j int) bool {
		for _, iMapped := range valuedSides[i].Mapped {
			if iMapped.ExtPosition == valuedSides[j].Position {
				return true
			}
		}
		return false
	})

	return valuedSides
}
