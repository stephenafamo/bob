package expr

import (
	"io"

	"github.com/stephenafamo/bob"
)

type as struct {
	Expression any
	Alias      string
}

func (a as) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.Express(w, d, start, a.Expression)
	if err != nil {
		return nil, err
	}

	if a.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, a.Alias)
	}

	return args, nil
}
