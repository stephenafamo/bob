package mysql

import (
	"io"

	"github.com/stephenafamo/bob"
)

type hints struct {
	hints []string
}

func (h *hints) AppendHint(hint string) {
	h.hints = append(h.hints, hint)
}

func (h hints) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, h.hints, "/*+ ", "\n    ", " */")
}
