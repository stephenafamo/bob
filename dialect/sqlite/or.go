package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

type or struct {
	action string
}

func (o *or) SetOr(to string) {
	o.action = to
}

func (o or) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressIf(w, d, start, o.action, o.action != "", " OR ", "")
}

type orMod[Q interface{ SetOr(string) }] struct{}

func (o orMod[Q]) OrAbort() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("ABORT")
	})
}

func (o orMod[Q]) OrFail() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("FAIL")
	})
}

func (o orMod[Q]) OrIgnore() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("IGNORE")
	})
}

func (o orMod[Q]) OrReplace() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("REPLACE")
	})
}

func (o orMod[Q]) OrRollback() query.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("ROLLBACK")
	})
}
