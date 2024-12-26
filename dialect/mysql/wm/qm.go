package wm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
)

func BasedOn(name string) bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetBasedOn(name)
	})
}

func PartitionBy(condition any) bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.AddPartitionBy(condition)
	})
}

func OrderBy(e any) dialect.OrderBy[*clause.Window] {
	return dialect.OrderBy[*clause.Window](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func Range() bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetMode("RANGE")
	})
}

func Rows() bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetMode("ROWS")
	})
}

func FromUnboundedPreceding() bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetStart("UNBOUNDED PRECEDING")
	})
}

func FromPreceding(exp any) bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetStart(bob.ExpressionFunc(
			func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
				return bob.ExpressIf(ctx, w, d, start, exp, true, "", " PRECEDING")
			}),
		)
	})
}

func FromCurrentRow() bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetStart("CURRENT ROW")
	})
}

func FromFollowing(exp any) bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetStart(bob.ExpressionFunc(
			func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
				return bob.ExpressIf(ctx, w, d, start, exp, true, "", " FOLLOWING")
			}),
		)
	})
}

func ToPreceding(exp any) bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetEnd(bob.ExpressionFunc(
			func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
				return bob.ExpressIf(ctx, w, d, start, exp, true, "", " PRECEDING")
			}),
		)
	})
}

func ToCurrentRow() bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetEnd("CURRENT ROW")
	})
}

func ToFollowing(exp any) bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetEnd(bob.ExpressionFunc(
			func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
				return bob.ExpressIf(ctx, w, d, start, exp, true, "", " FOLLOWING")
			}),
		)
	})
}

func ToUnboundedFollowing() bob.Mod[*clause.Window] {
	return bob.ModFunc[*clause.Window](func(w *clause.Window) {
		w.SetEnd("UNBOUNDED FOLLOWING")
	})
}
