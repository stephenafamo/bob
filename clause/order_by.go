package clause

import (
	"context"
	"fmt"
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
	args, err := bob.Express(ctx, w, d, start, o.Expression)
	if err != nil {
		return nil, err
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
		w.WriteString(fmt.Sprintf(" NULLS %s", o.Nulls))
	}

	return args, nil
}
