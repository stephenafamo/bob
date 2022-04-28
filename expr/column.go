package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

func C(e any, as string) Column {
	return Column{
		Expression: e,
		Alias:      as,
	}
}

type Column struct {
	Expression any
	Alias      string
}

func (c Column) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	args, err := query.Express(w, d, start, c.Expression)
	if err != nil {
		return nil, err
	}

	if c.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, c.Alias)
	}

	return args, nil
}
