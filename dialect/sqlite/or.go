package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
)

type or struct {
	action string
}

func (o *or) SetOr(to string) {
	o.action = to
}

func (o or) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressIf(w, d, start, o.action, o.action != "", " OR ", "")
}
