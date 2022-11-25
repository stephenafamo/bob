package clause

import (
	"io"

	"github.com/stephenafamo/bob"
)

type Table struct {
	Expression any
	Alias      string
	Columns    []string

	Partitions []string // MySQL
}

func (t Table) As(alias string, columns ...string) Table {
	t.Alias = alias
	t.Columns = append(t.Columns, columns...)

	return t
}

func (t Table) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.Express(w, d, start, t.Expression)
	if err != nil {
		return nil, err
	}

	if t.Alias != "" {
		w.Write([]byte(" AS "))
		d.WriteQuoted(w, t.Alias)
	}

	if len(t.Columns) > 0 {
		w.Write([]byte(" ("))
		for k, cAlias := range t.Columns {
			if k != 0 {
				w.Write([]byte(", "))
			}

			d.WriteQuoted(w, cAlias)
		}
		w.Write([]byte(")"))
	}

	_, err = bob.ExpressSlice(w, d, start, t.Partitions, " PARTITION (", ", ", ")")
	if err != nil {
		return nil, err
	}

	return args, nil
}
