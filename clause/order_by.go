package clause

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
)

type OrderBy struct {
	Expressions []OrderDef
}

func (o *OrderBy) SetOrderBy(orders ...OrderDef) {
	o.Expressions = orders
}

func (o *OrderBy) AppendOrder(order OrderDef) {
	o.Expressions = append(o.Expressions, order)
}

func (o OrderBy) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, o.Expressions, "ORDER BY ", ", ", "")
}

type OrderDef struct {
	Expression any
	Direction  string // ASC | DESC | USING operator
	Nulls      string // FIRST | LAST
	Collation  bob.Expression
}

func (o OrderDef) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.Express(ctx, w, d, start, o.Expression)
	if err != nil {
		return nil, err
	}

	if o.Collation != nil {
		_, err = o.Collation.WriteSQL(ctx, w, d, start)
		if err != nil {
			return nil, err
		}
	}

	if o.Direction != "" {
		w.Write([]byte(" "))
		w.Write([]byte(o.Direction))
	}

	if o.Nulls != "" {
		fmt.Fprintf(w, " NULLS %s", o.Nulls)
	}

	return args, nil
}
