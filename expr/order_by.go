package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
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

func Order(e any) OrderDef {
	return OrderDef{Expression: e}
}

type OrderDef struct {
	Expression    any
	Direction     string // ASC | DESC | USING operator
	Nulls         string // FIRST | LAST
	CollationName string
}

func (o OrderDef) Asc() OrderDef {
	o.Direction = "ASC"
	return o
}

func (o OrderDef) Desc() OrderDef {
	o.Direction = "DESC"
	return o
}

func (o OrderDef) Using(operator string) OrderDef {
	o.Direction = "USING " + operator
	return o
}

func (o OrderDef) NullsFirst() OrderDef {
	o.Nulls = "FIRST"
	return o
}

func (o OrderDef) NullsLast() OrderDef {
	o.Nulls = "LAST"
	return o
}

func (o OrderDef) Collate(collation string) OrderDef {
	o.CollationName = collation
	return o
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
