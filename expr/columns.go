package expr

import (
	"context"
	"io"
	"reflect"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/internal/mappings"
)

// NewColumnsExpr returns a [ColumnsExpr] object with the given column names
func NewColumnsExpr(names ...string) ColumnsExpr {
	return ColumnsExpr{names: internal.FilterNonZero(names)}
}

// ColumnsExpr is a set of columns that can be used in a query
// It is used to properly quote and format the columns in the query
// such as "users"."id" AS "id", "users"."name" AS "name"
type ColumnsExpr struct {
	parent        []string
	names         []string
	aggFunc       [2]string
	aliasPrefix   string
	aliasDisabled bool
}

// Expressions is a list of expressions that can be rendered as a comma-separated SQL list.
type Expressions []bob.Expression

func (e Expressions) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, e, "", commaSpace, "")
}

func (e Expressions) Any() []any {
	return internal.ToAnySlice(e)
}

func (Expressions) ShouldOmitParens() bool {
	return true
}

var (
	_ bob.Expression    = Expressions(nil)
	_ bob.ParensOmitter = Expressions(nil)
)

// Names returns the names of the columns
func (c ColumnsExpr) Names() []string {
	return slices.Clone(c.names)
}

// Expressions returns each column as an expression using current options.
func (c ColumnsExpr) Expressions() Expressions {
	exprs := make(Expressions, 0, len(c.names))

	for _, col := range c.names {
		colExpr := c
		colExpr.names = []string{col}
		exprs = append(exprs, colExpr)
	}

	return exprs
}

func (c ColumnsExpr) WithAggFunc(a, b string) ColumnsExpr {
	c.aggFunc = [2]string{a, b}
	return c
}

// WithParent sets the parent of the columns.
func (c ColumnsExpr) WithParent(p ...string) ColumnsExpr {
	c.parent = p
	return c
}

// WithPrefix sets the prefix of the aliases of the column set.
func (c ColumnsExpr) WithPrefix(prefix string) ColumnsExpr {
	c.aliasPrefix = prefix
	return c
}

// EnableAlias enables adding 'AS "prefix_column_name"' when writing SQL.
func (c ColumnsExpr) EnableAlias() ColumnsExpr {
	c.aliasDisabled = false
	return c
}

// DisableAlias disables adding 'AS "prefix_column_name"' when writing SQL.
func (c ColumnsExpr) DisableAlias() ColumnsExpr {
	c.aliasDisabled = true
	return c
}

// Only drops other column names from the column set
func (c ColumnsExpr) Only(cols ...string) ColumnsExpr {
	c.names = internal.Only(c.names, cols...)
	return c
}

// Except drops the given column names from the column set
func (c ColumnsExpr) Except(cols ...string) ColumnsExpr {
	c.names = internal.Except(c.names, cols...)
	return c
}

func (c ColumnsExpr) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if len(c.names) == 0 {
		return nil, nil
	}

	hasParent := false
	for _, part := range c.parent {
		if part != "" {
			hasParent = true
			break
		}
	}

	shouldAlias := !c.aliasDisabled && (hasParent || c.aliasPrefix != "" || c.aggFunc != [2]string{})

	// wrap in parenthesis and join with comma
	for k, col := range c.names {
		if k != 0 {
			w.WriteString(", ")
		}

		w.WriteString(c.aggFunc[0])
		for _, part := range c.parent {
			if part == "" {
				continue
			}
			d.WriteQuoted(w, part)
			w.WriteString(".")
		}

		d.WriteQuoted(w, col)
		w.WriteString(c.aggFunc[1])

		if shouldAlias {
			w.WriteString(" AS ")
			d.WriteQuoted(w, c.aliasPrefix+col)
		}
	}

	return nil, nil
}

func ColsForStruct[T any](name string) ColumnsExpr {
	var model T
	return NewColumnsExpr(mappings.GetMappings(reflect.TypeOf(model)).All...).WithParent(name)
}
