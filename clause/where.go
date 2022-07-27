package clause

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Where struct {
	Conditions []any
}

func (wh *Where) AppendWhere(e ...any) {
	wh.Conditions = append(wh.Conditions, e...)
}

func (wh Where) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.ExpressSlice(w, d, start, wh.Conditions, "WHERE ", " AND ", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
