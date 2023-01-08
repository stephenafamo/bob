package orm

import (
	"context"

	"github.com/stephenafamo/bob"
)

type RelWhere struct {
	Column string
	Value  string
}

type RelSide struct {
	From        string
	FromColumns []string
	To          string
	ToColumns   []string

	FromWhere, ToWhere []RelWhere

	// If the destination columns contain the key
	// if false, it means the source columns are the foreign key
	ToKey bool
	// if the destination is unique
	ToUnique bool
	// If the key is nullable. We need this to know if we can remove the
	// relationship without deleting it
	KeyNullable bool

	// Kinda hacky, used for preloading
	ToExpr func(context.Context) bob.Expression `json:"-"`
}

type Relationship struct {
	Name        string
	Alias       string // if present is used instead of computing from the columns
	ByJoinTable bool
	Sides       []RelSide

	Ignored bool // Can be set through user configuration
}

func (r Relationship) Local() string {
	return r.Sides[0].From
}

func (r Relationship) Foreign() string {
	return r.Sides[len(r.Sides)-1].To
}

func (r Relationship) IsToMany() bool {
	for _, side := range r.Sides {
		if !side.ToUnique {
			return true
		}
	}

	return false
}

func (r Relationship) IsRemovable() bool {
	return false
}

func (r Relationship) InsertEarly() bool {
	foreign := r.Foreign()
	mappings := r.ValuedSides()
	for _, mapping := range mappings {
		if mapping.TableName == foreign {
			return false
		}
	}

	return true
}

func (r Relationship) NeededColumns() []string {
	ma := []string{}

	local := r.Local()
	foreign := r.Foreign()
	mappings := r.ValuedSides()
	for _, mapping := range mappings {
		for _, ext := range mapping.Mapped {
			if ext.ExternalTable == "" {
				continue
			}
			if ext.ExternalTable == local {
				continue
			}
			if ext.ExternalTable == foreign {
				continue
			}

			ma = append(ma, ext.ExternalTable)
		}
	}

	return ma
}

type RelSetDetails struct {
	TableName string
	Mapped    []RelSetMapping
}

type RelSetMapping struct {
	Column         string
	Value          string
	ExternalTable  string
	ExternalColumn string
}

func (r Relationship) StaticSides() []struct {
	Table   string
	Columns [][2]string
} {
	x := make(map[string][][2]string, len(r.Sides))
	for _, side := range r.Sides {
		if len(side.FromWhere) > 0 {
			columns := make([][2]string, 0, len(side.FromWhere))
			for _, f := range side.FromWhere {
				columns = append(columns, [2]string{f.Column, f.Value})
			}
			x[side.From] = append(x[side.From], columns...)
		}

		if len(side.ToWhere) > 0 {
			columns := make([][2]string, 0, len(side.ToWhere))
			for _, f := range side.ToWhere {
				columns = append(columns, [2]string{f.Column, f.Value})
			}
			x[side.To] = append(x[side.To], columns...)
		}
	}

	x2 := make([]struct {
		Table   string
		Columns [][2]string
	}, 0, len(x))

	for table, columns := range x {
		x2 = append(x2, struct {
			Table   string
			Columns [][2]string
		}{
			Table:   table,
			Columns: columns,
		})
	}
	return x2
}

func (r Relationship) ValuedSides() []RelSetDetails {
	x := make([]RelSetDetails, 0, len(r.Sides))

	for i, side := range r.Sides {
		fromDeets := RelSetDetails{
			TableName: side.From,
			Mapped:    make([]RelSetMapping, 0, len(side.FromColumns)+len(side.FromWhere)),
		}

		toDeets := RelSetDetails{
			TableName: side.To,
			Mapped:    make([]RelSetMapping, 0, len(side.ToColumns)+len(side.ToWhere)),
		}

		if len(side.FromWhere) > 0 {
			for _, f := range side.FromWhere {
				fromDeets.Mapped = append(fromDeets.Mapped, RelSetMapping{
					Column: f.Column,
					Value:  f.Value,
				})
			}
		}

		if len(side.ToWhere) > 0 {
			for _, f := range side.ToWhere {
				toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
					Column: f.Column,
					Value:  f.Value,
				})
			}
		}

		//nolint:nestif
		if !side.ToKey {
			if i == 0 || !r.Sides[i-1].ToKey {
				for i, f := range side.FromColumns {
					fromDeets.Mapped = append(fromDeets.Mapped, RelSetMapping{
						Column:         f,
						ExternalTable:  side.To,
						ExternalColumn: side.ToColumns[i],
					})
				}
			}
		} else {
			for i, f := range side.FromColumns {
				toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
					Column:         side.ToColumns[i],
					ExternalTable:  side.From,
					ExternalColumn: f,
				})
			}

			if len(r.Sides) > i+1 {
				nextSide := r.Sides[i+1]
				if !nextSide.ToKey {
					for i, f := range nextSide.FromColumns {
						toDeets.Mapped = append(toDeets.Mapped, RelSetMapping{
							Column:         f,
							ExternalTable:  nextSide.To,
							ExternalColumn: nextSide.ToColumns[i],
						})
					}
				}
			}
		}

		if len(fromDeets.Mapped) > 0 {
			x = append(x, fromDeets)
		}
		if len(toDeets.Mapped) > 0 {
			x = append(x, toDeets)
		}
	}

	return x
}
