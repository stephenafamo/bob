package clause

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type OrderBy struct {
	Expressions []OrderDef
}

func (o *OrderBy) AppendOrder(order OrderDef) {
	o.Expressions = append(o.Expressions, order)
}

func (o OrderBy) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, o.Expressions, "ORDER BY ", ", ", "")
}

type OrderDef struct {
	Expression    any
	Direction     string // ASC | DESC | USING operator
	Nulls         string // FIRST | LAST
	CollationName string
}

func (o OrderDef) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if o.CollationName != "" {
		w.Write([]byte("COLLATE "))
		w.Write([]byte(o.CollationName))
	}

	args, err := query.Express(w, d, start, o.Expression)
	if err != nil {
		return nil, err
	}

	if o.Direction != "" {
		w.Write([]byte(" "))
		w.Write([]byte(o.Direction))
	}

	if o.Nulls != "" {
		w.Write([]byte(" NULLS"))
		w.Write([]byte(o.Nulls))
	}

	return args, nil
}
