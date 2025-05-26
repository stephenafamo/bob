package orm

import (
	"context"
	"io"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal"
)

// NewColumns returns a [Columns] object with the given column names
func NewColumns(names ...string) Columns {
	return Columns{names: internal.FilterNonZero(names)}
}

// Columns is a set of columns that can be used in a query
// It is used to properly quote and format the columns in the query
// such as "users"."id" AS "id", "users"."name" AS "name"
type Columns struct {
	parent        []string
	names         []string
	aggFunc       [2]string
	aliasPrefix   string
	aliasDisabled bool
}

// Names returns the names of the columns
func (c Columns) Names() []string {
	return slices.Clone(c.names)
}

func (c Columns) WithAggFunc(a, b string) Columns {
	c.aggFunc = [2]string{a, b}
	return c
}

// WithPrefix sets the parent of the columns
func (c Columns) WithParent(p ...string) Columns {
	c.parent = p
	return c
}

// WithPrefix sets the prefix of the aliases of the column set
func (c Columns) WithPrefix(prefix string) Columns {
	c.aliasPrefix = prefix
	return c
}

// Enables adding 'AS "prefix_column_name"' when writing SQL
func (c Columns) EnableAlias() Columns {
	c.aliasDisabled = false
	return c
}

// Disables add 'AS "prefix_column_name"' when writing SQL
func (c Columns) DisableAlias() Columns {
	c.aliasDisabled = true
	return c
}

// Only drops other column names from the column set
func (c Columns) Only(cols ...string) Columns {
	c.names = Only(c.names, cols...)
	return c
}

// Except drops the given column names from the column set
func (c Columns) Except(cols ...string) Columns {
	c.names = Except(c.names, cols...)
	return c
}

func (c Columns) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if len(c.names) == 0 {
		return nil, nil
	}

	// wrap in parenthesis and join with comma
	for k, col := range c.names {
		if k != 0 {
			w.Write([]byte(", "))
		}

		w.Write([]byte(c.aggFunc[0]))
		for _, part := range c.parent {
			if part == "" {
				continue
			}
			d.WriteQuoted(w, part)
			w.Write([]byte("."))
		}

		d.WriteQuoted(w, col)
		w.Write([]byte(c.aggFunc[1]))

		if !c.aliasDisabled {
			w.Write([]byte(" AS "))
			d.WriteQuoted(w, c.aliasPrefix+col)
		}
	}

	return nil, nil
}

// Only drops other column names from the column set
func Only(cols []string, includes ...string) []string {
	filtered := make([]string, 0, len(includes)) // max capacity is the only list

Outer:
	for _, basecol := range cols {
		if len(basecol) == 0 {
			continue
		}

		for _, include := range includes {
			if include == basecol {
				filtered = append(filtered, basecol)
				continue Outer
			}
		}
	}

	return filtered
}

// Except drops the given column names from the column set
func Except(cols []string, excludes ...string) []string {
	filtered := make([]string, 0, len(cols)) // max capacity is current capacity

Outer:
	for _, basecol := range cols {
		if len(basecol) == 0 {
			continue
		}

		for _, exclude := range excludes {
			if exclude == basecol {
				continue Outer
			}
		}

		filtered = append(filtered, basecol)
	}

	return filtered
}
