package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type Using struct {
	Tables []any
}

func (u *Using) AppendUsing(tables ...any) {
	u.Tables = append(u.Tables, tables...)
}

func (u Using) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if len(u.Tables) == 0 {
		return nil, nil
	}

	w.Write([]byte("USING "))

	args, err := query.ExpressSlice(w, d, start, u.Tables, "", ", ", " ")
	if err != nil {
		return nil, err
	}

	return args, nil
}
