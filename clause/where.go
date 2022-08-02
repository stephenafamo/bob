package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type Where struct {
	Conditions []any
}

func (wh *Where) AppendWhere(e ...any) {
	wh.Conditions = append(wh.Conditions, e...)
}

func (wh Where) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.ExpressSlice(w, d, start, wh.Conditions, "WHERE ", " AND ", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
