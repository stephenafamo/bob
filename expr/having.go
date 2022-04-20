package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type Having struct {
	Conditions []any
}

func (h *Having) AppendHaving(e ...any) {
	h.Conditions = append(h.Conditions, e...)
}

func (h Having) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.ExpressSlice(w, d, start, h.Conditions, "HAVING ", " AND ", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
