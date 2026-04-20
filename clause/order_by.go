package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type OrderBy struct {
	Expressions []bob.Expression
}

func (o *OrderBy) ClearOrderBy() {
	o.Expressions = nil
}

func (o *OrderBy) AppendOrder(order bob.Expression) {
	o.Expressions = append(o.Expressions, order)
}

func (o OrderBy) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, o.Expressions, "ORDER BY ", ", ", "")
}

type OrderDef struct {
	Expression any
	Direction  string // ASC | DESC | USING operator
	Nulls      string // FIRST | LAST
	Collation  string
}

func (o OrderDef) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	err := o.WriteSQLTo(ctx, w, d, start, &args)
	return args, err
}

func (o OrderDef) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	if err := bob.ExpressTo(ctx, w, d, start, o.Expression, args); err != nil {
		return err
	}

	if o.Collation != "" {
		w.WriteString(" COLLATE ")
		d.WriteQuoted(w, o.Collation)
	}

	if o.Direction != "" {
		w.WriteString(" ")
		w.WriteString(o.Direction)
	}

	if o.Nulls != "" {
		w.WriteString(" NULLS ")
		w.WriteString(o.Nulls)
	}

	return nil
}
