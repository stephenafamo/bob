package builder

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type as struct {
	Expression any
	Alias      string
}

func (a as) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.Express(w, d, start, a.Expression)
	if err != nil {
		return nil, err
	}

	if a.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, a.Alias)
	}

	return args, nil
}
