package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/mods"
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

type orMod[Q interface{ SetOr(string) }] struct{}

func (o orMod[Q]) OrAbort() bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("ABORT")
	})
}

func (o orMod[Q]) OrFail() bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("FAIL")
	})
}

func (o orMod[Q]) OrIgnore() bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("IGNORE")
	})
}

func (o orMod[Q]) OrReplace() bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("REPLACE")
	})
}

func (o orMod[Q]) OrRollback() bob.Mod[Q] {
	return mods.QueryModFunc[Q](func(i Q) {
		i.SetOr("ROLLBACK")
	})
}
