package mods

import (
	"github.com/stephenafamo/typesql/expr"
)

type QueryMod[T any] interface {
	Apply(T)
}

type QueryModFunc[T any] func(T)

func (q QueryModFunc[T]) Apply(query T) {
	q(query)
}

type With[Q interface{ AppendWith(expr.CTE) }] expr.CTE

func (f With[Q]) Apply(q Q) {
	q.AppendWith(expr.CTE(f))
}

type Recursive[Q interface{ SetRecursive(bool) }] bool

func (r Recursive[Q]) Apply(q Q) {
	q.SetRecursive(bool(r))
}

type Distinct[Q interface{ SetDistinct(expr.Distinct) }] expr.Distinct

func (d Distinct[Q]) Apply(q Q) {
	q.SetDistinct(expr.Distinct(d))
}

type Select[Q interface{ AppendSelect(columns ...any) }] []any

func (s Select[Q]) Apply(q Q) {
	q.AppendSelect(s...)
}

type From[Q interface{ AppendFrom(...any) }] []any

func (f From[Q]) Apply(q Q) {
	q.AppendFrom(f...)
}

type Join[Q interface{ AppendJoin(expr.Join) }] expr.Join

func (j Join[Q]) Apply(q Q) {
	q.AppendJoin(expr.Join(j))
}

type Where[Q interface{ AppendWhere(e ...any) }] []any

func (d Where[Q]) Apply(q Q) {
	q.AppendWhere(d...)
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

type Window[Q interface{ AppendWindow(expr.NamedWindow) }] expr.NamedWindow

func (f Window[Q]) Apply(q Q) {
	q.AppendWindow(expr.NamedWindow(f))
}

type OrderBy[Q interface{ AppendOrder(expr.OrderDef) }] expr.OrderDef

func (f OrderBy[Q]) Apply(q Q) {
	q.AppendOrder(expr.OrderDef(f))
}

type Limit[Q interface{ SetLimit(expr.Limit) }] expr.Limit

func (f Limit[Q]) Apply(q Q) {
	q.SetLimit(expr.Limit(f))
}

type Offset[Q interface{ SetOffset(expr.Offset) }] expr.Offset

func (f Offset[Q]) Apply(q Q) {
	q.SetOffset(expr.Offset(f))
}

type Fetch[Q interface{ SetFetch(expr.Fetch) }] expr.Fetch

func (f Fetch[Q]) Apply(q Q) {
	q.SetFetch(expr.Fetch(f))
}

type Combine[Q interface{ SetCombine(expr.Combine) }] expr.Combine

func (f Combine[Q]) Apply(q Q) {
	q.SetCombine(expr.Combine(f))
}

type For[Q interface{ SetFor(expr.For) }] expr.For

func (f For[Q]) Apply(q Q) {
	q.SetFor(expr.For(f))
}

type Values[Q interface{ AppendValues(vals ...any) }] []any

func (s Values[Q]) Apply(q Q) {
	q.AppendValues(s...)
}

type Returning[Q interface{ AppendReturning(vals ...any) }] []any

func (s Returning[Q]) Apply(q Q) {
	q.AppendReturning(s...)
}

type Set[Q interface{ AppendSet(exprs ...any) }] []any

func (s Set[Q]) Apply(q Q) {
	q.AppendSet(s...)
}
