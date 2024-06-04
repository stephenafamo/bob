package mods

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

type QueryMods[T any] []bob.Mod[T]

func (q QueryMods[T]) Apply(query T) {
	for _, v := range q {
		v.Apply(query)
	}
}

type QueryModFunc[T any] func(T)

func (q QueryModFunc[T]) Apply(query T) {
	q(query)
}

// This is a generic type for expressions can take extra mods as a function
// allows for some fluent API, for example with functions
type Moddable[T bob.Expression] func(...bob.Mod[T]) T

func (m Moddable[T]) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return m().WriteSQL(w, d, start)
}

type With[Q interface{ AppendWith(clause.CTE) }] clause.CTE

func (f With[Q]) Apply(q Q) {
	q.AppendWith(clause.CTE(f))
}

type Recursive[Q interface{ SetRecursive(bool) }] bool

func (r Recursive[Q]) Apply(q Q) {
	q.SetRecursive(bool(r))
}

type Select[Q interface{ AppendSelect(columns ...any) }] []any

func (s Select[Q]) Apply(q Q) {
	q.AppendSelect(s...)
}

type Preload[Q interface{ AppendPreloadSelect(columns ...any) }] []any

func (s Preload[Q]) Apply(q Q) {
	q.AppendPreloadSelect(s...)
}

type Join[Q interface{ AppendJoin(clause.Join) }] clause.Join

func (j Join[Q]) Apply(q Q) {
	q.AppendJoin(clause.Join(j))
}

type Where[Q interface{ AppendWhere(e ...any) }] struct {
	E bob.Expression
}

func (w Where[Q]) Apply(q Q) {
	q.AppendWhere(w.E)
}

type GroupBy[Q interface{ AppendGroup(any) }] struct {
	E any
}

func (f GroupBy[Q]) Apply(q Q) {
	q.AppendGroup(f.E)
}

type GroupWith[Q interface{ SetGroupWith(string) }] string

func (f GroupWith[Q]) Apply(q Q) {
	q.SetGroupWith(string(f))
}

type GroupByDistinct[Q interface{ SetGroupByDistinct(bool) }] bool

func (f GroupByDistinct[Q]) Apply(q Q) {
	q.SetGroupByDistinct(bool(f))
}

type Having[Q interface{ AppendHaving(e ...any) }] []any

func (d Having[Q]) Apply(q Q) {
	q.AppendHaving(d...)
}

type Window[Q interface{ AppendWindow(clause.NamedWindow) }] clause.NamedWindow

func (f Window[Q]) Apply(q Q) {
	q.AppendWindow(clause.NamedWindow(f))
}

type OrderBy[Q interface{ AppendOrder(clause.OrderDef) }] clause.OrderDef

func (f OrderBy[Q]) Apply(q Q) {
	q.AppendOrder(clause.OrderDef(f))
}

type Limit[Q interface{ SetLimit(limit any) }] struct {
	Count any
}

func (f Limit[Q]) Apply(q Q) {
	q.SetLimit(f.Count)
}

type Offset[Q interface{ SetOffset(offset any) }] struct {
	Count any
}

func (f Offset[Q]) Apply(q Q) {
	q.SetOffset(f.Count)
}

type Fetch[Q interface{ SetFetch(clause.Fetch) }] clause.Fetch

func (f Fetch[Q]) Apply(q Q) {
	q.SetFetch(clause.Fetch(f))
}

type Combine[Q interface{ SetCombine(clause.Combine) }] clause.Combine

func (f Combine[Q]) Apply(q Q) {
	q.SetCombine(clause.Combine(f))
}

type For[Q interface{ SetFor(clause.For) }] clause.For

func (f For[Q]) Apply(q Q) {
	q.SetFor(clause.For(f))
}

type Values[Q interface{ AppendValues(vals ...bob.Expression) }] []bob.Expression

func (s Values[Q]) Apply(q Q) {
	q.AppendValues(s...)
}

type Rows[Q interface{ AppendValues(vals ...bob.Expression) }] [][]bob.Expression

func (r Rows[Q]) Apply(q Q) {
	for _, row := range r {
		q.AppendValues(row...)
	}
}

type Returning[Q interface{ AppendReturning(vals ...any) }] []any

func (s Returning[Q]) Apply(q Q) {
	q.AppendReturning(s...)
}

type Set[Q interface{ AppendSet(clauses ...any) }] []string

func (s Set[Q]) To(to any) bob.Mod[Q] {
	return set[Q]{expr.OP("=", expr.Quote(s...), to)}
}

func (s Set[Q]) ToArg(to any) bob.Mod[Q] {
	return set[Q]{expr.OP("=", expr.Quote(s...), expr.Arg(to))}
}

type set[Q interface{ AppendSet(clauses ...any) }] []any

func (s set[Q]) Apply(q Q) {
	q.AppendSet(s...)
}
