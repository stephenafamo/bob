package expr

import (
	"io"
	"strconv"

	"github.com/stephenafamo/typesql/query"
)

type Limit struct {
	Count *int64
}

func (l *Limit) SetLimit(limit Limit) {
	l = &limit
}

func (l Limit) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if l.Count == nil {
		return nil, nil
	}

	w.Write([]byte("LIMIT "))
	w.Write([]byte(strconv.FormatInt(*l.Count, 10)))
	return nil, nil
}
