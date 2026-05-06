package dialect

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob"
)

type hints struct {
	hints []string
}

func (h *hints) AppendHint(hint string) {
	h.hints = append(h.hints, hint)
}

func (h hints) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, h.hints, "/*+ ", " ", " */ ")
}

type hintable interface{ AppendHint(string) }

// Scan hints

func SeqScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("SeqScan(%s)", table)
	return bob.ModFunc[Q](func (q Q) {
		q.AppendHint(hint)
	})
}

func NoSeqScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoSeqScan(%s)", table)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func IndexScan[Q hintable](table string, indexes ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("IndexScan(%s)", joinWithTable(table, indexes))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoIndexScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoIndexScan(%s)", table)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func IndexOnlyScan[Q hintable](table string, indexes ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("IndexOnlyScan(%s)", joinWithTable(table, indexes))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoIndexOnlyScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoIndexOnlyScan(%s)", table)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func BitmapScan[Q hintable](table string, indexes ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("BitmapScan(%s)", joinWithTable(table, indexes))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoBitmapScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoBitmapScan(%s)", table)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func TidScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("TidScan(%s)", table)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoTidScan[Q hintable](table string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoTidScan(%s)", table)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

// Join hints

func NestLoop[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NestLoop(%s)", strings.Join(tables, " "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoNestLoop[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoNestLoop(%s)", strings.Join(tables, " "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func HashJoin[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("HashJoin(%s)", strings.Join(tables, " "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoHashJoin[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoHashJoin(%s)", strings.Join(tables, " "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func MergeJoin[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("MergeJoin(%s)", strings.Join(tables, " "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func NoMergeJoin[Q hintable](tables ...string) bob.Mod[Q] {
	hint := fmt.Sprintf("NoMergeJoin(%s)", strings.Join(tables, " "))
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

// Join order hint

func Leading[Q hintable](spec string) bob.Mod[Q] {
	hint := fmt.Sprintf("Leading(%s)", spec)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

// Row estimation hint

func Rows[Q hintable](spec string) bob.Mod[Q] {
	hint := fmt.Sprintf("Rows(%s)", spec)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

// Parallel hint

func Parallel[Q hintable](table string, nworkers int, strength string) bob.Mod[Q] {
	hint := fmt.Sprintf("Parallel(%s %d %s)", table, nworkers, strength)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

// GUC hint

func Set[Q hintable](variable string, value string) bob.Mod[Q] {
	hint := fmt.Sprintf("Set(%s %s)", variable, value)
	return bob.ModFunc[Q](func(q Q) {
		q.AppendHint(hint)
	})
}

func joinWithTable(table string, extras []string) string {
	if len(extras) == 0 {
		return table
	}
	return table + " " + strings.Join(extras, " ")
}
