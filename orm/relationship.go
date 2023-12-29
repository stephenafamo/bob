package orm

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
)

type RelWhere struct {
	Column  string `yaml:"column"`
	Value   string `yaml:"value"`
	GoValue string `yaml:"go_value"`
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

	// if the origin is unique
	FromUnique bool `yaml:"from_unique"`
	// if the destination is unique
	ToUnique bool `yaml:"to_unique"`
	// If the destination columns contain the key
	// if false, it means the source columns are the foreign key
	ToKey bool `yaml:"to_key"`
	// If the key is nullable. We need this to know if we can remove the
	// relationship without deleting it
	KeyNullable bool `yaml:"key_nullable"`

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

	// if present is used instead of computing from the columns
	// only expected to be set by drivers not by configuration
	// configuration should set aliases though the alias configuration
	Alias string `yaml:"-"`
}

func (r Relationship) Validate() error {
	for index := range r.Sides {
		if index == 0 {
			continue
		}

		if r.Sides[index-1].To != r.Sides[index].From {
			return fmt.Errorf("relationship %s has a gap between %s and %s", r.Name, r.Sides[index-1].To, r.Sides[index].From)
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
	for _, side := range r.ValuedSides() {
		if side.Position == 0 {
			return false
		}

		if r.Sides[side.Position-1].ToUnique {
			return false
		}
	}

	return true
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
	Value          string
	ExternalTable  string
	ExternalColumn string
	ExtPosition    int
	ExternalStart  bool
	ExternalEnd    bool
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

	for i := position; i < len(r.Sides); i++ {
		// If there is another "Many" side down the line
		// this this should not be many
		if !r.Sides[i].ToUnique {
			return false
		}
	}

	// If the key on the side is not unique, then it needs to be many
	return !r.Sides[position-1].ToUnique
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
					Value:  f.GoValue,
				})
			}
		}

		if len(side.ToWhere) > 0 {
			for _, f := range side.ToWhere {
				toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
					Column: f.Column,
					Value:  f.GoValue,
				})
			}
		}

		//nolint:nestif
		if !side.ToKey {
			if sideIndex == 0 || !r.Sides[sideIndex-1].ToKey {
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
				if !nextSide.ToKey {
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

	return valuedSides
}
