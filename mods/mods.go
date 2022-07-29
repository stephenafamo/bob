package mods

import (
	"github.com/stephenafamo/bob/clause"
)

type QueryModFunc[T any] func(T)

func (q QueryModFunc[T]) Apply(query T) {
	q(query)
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

type FromItems[Q interface{ AppendFromItem(clause.FromItem) }] clause.FromItem

func (f FromItems[Q]) Apply(q Q) {
	q.AppendFromItem(clause.FromItem(f))
}

type TableAs[Q interface {
	SetTableAlias(alias string, columns ...string)
}] struct {
	Alias   string
	Columns []string
}

func (t TableAs[Q]) Apply(q Q) {
	q.SetTableAlias(t.Alias, t.Columns...)
}

type Join[Q interface{ AppendJoin(clause.Join) }] clause.Join

func (j Join[Q]) Apply(q Q) {
	q.AppendJoin(clause.Join(j))
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

type Values[Q interface{ AppendValues(vals ...any) }] []any

func (s Values[Q]) Apply(q Q) {
	q.AppendValues(s...)
}

type Returning[Q interface{ AppendReturning(vals ...any) }] []any

func (s Returning[Q]) Apply(q Q) {
	q.AppendReturning(s...)
}

type Set[Q interface{ AppendSet(clauses ...any) }] []any

func (s Set[Q]) Apply(q Q) {
	q.AppendSet(s...)
}
