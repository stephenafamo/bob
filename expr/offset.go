package expr

import (
	"io"
	"strconv"

	"github.com/stephenafamo/typesql/query"
)

type Offset struct {
	Count *int64
}

func (o *Offset) SetOffset(offset Offset) {
	*o = offset
}

func (o Offset) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if o.Count == nil {
		return nil, nil
	}
	w.Write([]byte("OFFSET "))
	w.Write([]byte(strconv.FormatInt(*o.Count, 10)))
	return nil, nil
}
