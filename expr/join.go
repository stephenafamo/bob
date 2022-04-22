package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

const (
	InnerJoin    = "INNER JOIN"
	LeftJoin     = "LEFT JOIN"
	RightJoin    = "RIGHT JOIN"
	FullJoin     = "FULL JOIN"
	CrossJoin    = "CROSS JOIN"
	StraightJoin = "STRAIGHT_JOIN"
)

type Join struct {
	Type  string
	To    any // the expression for the table
	Alias string

	// Join methods
	Natural bool
	On      []any
	Using   []any
}

func (j Join) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if j.Natural {
		w.Write([]byte("NATURAL "))
	}

	w.Write([]byte(j.Type))

	args, err := query.Express(w, d, start, j.To)
	if err != nil {
		return nil, err
	}

	onArgs, err := query.ExpressSlice(w, d, start+len(args), j.On, " ON ", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, onArgs...)

	usingArgs, err := query.ExpressSlice(w, d, start+len(args), j.Using, " USING(", ", ", ")")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	if j.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, j.Alias)
	}

	return args, nil
}
