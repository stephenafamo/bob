package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
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
	Type string
	To   From // the expression for the table

	// Join methods
	Natural bool
	On      []bob.Expression
	Using   []string
}

func (j Join) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if j.Natural {
		w.Write([]byte("NATURAL "))
	}

	w.Write([]byte(j.Type))
	w.Write([]byte(" "))

	args, err := bob.Express(ctx, w, d, start, j.To)
	if err != nil {
		return nil, err
	}

	onArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), j.On, " ON ", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, onArgs...)

	for k, col := range j.Using {
		if k == 0 {
			w.Write([]byte(" USING("))
		} else {
			w.Write([]byte(", "))
		}

		_, err = expr.Quote(col).WriteSQL(ctx, w, d, 1) // start does not matter
		if err != nil {
			return nil, err
		}

		if k == len(j.Using)-1 {
			w.Write([]byte(") "))
		}

	}

	return args, nil
}
