package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

type Values struct {
	// Query takes the highest priority
	// If present, will attempt to insert from this query
	Query query.Query

	// for multiple inserts
	// each sub-slice is one set of values
	Vals []group
}

func (v *Values) AppendValues(vals ...any) {
	if len(vals) == 0 {
		return
	}

	v.Vals = append(v.Vals, vals)
}

func (v Values) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	// If a query is present, use it
	if v.Query != nil {
		return v.Query.WriteQuery(w, start)
	}

	// If values are present, use them
	if len(v.Vals) > 0 {
		return query.ExpressSlice(w, d, start, v.Vals, "VALUES ", ", ", "")
	}

	// If no value was present, use default value
	w.Write([]byte("DEFAULT VALUES"))
	return nil, nil
}
