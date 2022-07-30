package mysql

import (
	"fmt"
	"io"

	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

type hints struct {
	hints []string
}

func (h *hints) AppendHint(hint string) {
	h.hints = append(h.hints, hint)
}

func (h hints) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressSlice(w, d, start, h.hints, "/*+ ", "\n    ", " */")
}

type hintMod[Q interface{ AppendHint(string) }] struct{}

func (hintMod[Q]) SetVar(statement string) query.Mod[Q] {
	hint := fmt.Sprintf("SET_VAR(%s)", statement)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func (hintMod[Q]) MaxExecutionTime(n int) query.Mod[Q] {
	hint := fmt.Sprintf("MAX_EXECUTION_TIME(%d)", n)
	return mods.QueryModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

//TODO: Add all other hints: https://dev.mysql.com/doc/refman/8.0/en/optimizer-hints.html
