package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// CountPreloadColumn is the SELECT-list column a count Preloader appends
// to a query via AppendPreloadSelect.
type CountPreloadColumn struct {
	// Name is the relationship name.
	Name string
	// Alias is always "__count_" + Name.
	Alias string
	// Expr is the correlated count subquery.
	Expr bob.Expression
}

// WriteSQL renders `<Expr> AS "<Alias>"`.
func (c CountPreloadColumn) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	args, err := c.Expr.WriteSQL(ctx, w, d, start)
	if err != nil {
		return nil, err
	}
	w.WriteString(" AS ")
	d.WriteQuoted(w, c.Alias)
	return args, nil
}
