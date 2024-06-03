package clause

import (
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

func (o OrderBy) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, o.Expressions, "ORDER BY ", ", ", "")
}

type OrderDef struct {
	Expression    any
	Direction     string // ASC | DESC | USING operator
	Nulls         string // FIRST | LAST
	CollationName string
}

func (o OrderDef) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if o.CollationName != "" {
		w.Write([]byte("COLLATE "))
		w.Write([]byte(o.CollationName))
	}

	args, err := bob.Express(w, d, start, o.Expression)
	if err != nil {
		return nil, err
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
