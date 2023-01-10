package dialect

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

type modifiers[T any] struct {
	modifiers []T
}

func (h *modifiers[T]) AppendModifier(modifier T) {
	h.modifiers = append(h.modifiers, modifier)
}

func (h modifiers[T]) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, h.modifiers, "", " ", "")
}

type Set struct {
	Col string
	Val any
}

func (s Set) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.Express(w, d, start, expr.OP("=", expr.Quote(s.Col), s.Val))
}

type partitions struct {
	partitions []string
}

func (h *partitions) AppendPartition(partitions ...string) {
	h.partitions = append(h.partitions, partitions...)
}

func (h partitions) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(w, d, start, h.partitions, "PARTITION (", ", ", ")")
}
