package mods

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

type QueryMods[T any] []bob.Mod[T]

func (q QueryMods[T]) Edit(query T) {
	for _, v := range q {
		v.Edit(query)
	}
}

// This is a generic type for expressions can take extra mods as a function
// allows for some fluent API, for example with functions
type Moddable[T bob.Expression] func(...bob.Mod[T]) T

func (m Moddable[T]) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return m().WriteSQL(ctx, w, d, start)
}

type With[Q interface{ AppendCTE(clause.CTE) }] clause.CTE

func (f With[Q]) Edit(q Q) {
	q.AppendCTE(clause.CTE(f))
}

type Recursive[Q interface{ SetRecursive(bool) }] bool

func (r Recursive[Q]) Edit(q Q) {
	q.SetRecursive(bool(r))
}

type Select[Q interface{ AppendSelect(columns ...any) }] []any

func (s Select[Q]) Edit(q Q) {
	q.AppendSelect(s...)
}

type Preload[Q interface{ AppendPreloadSelect(columns ...any) }] []any

func (s Preload[Q]) Edit(q Q) {
	q.AppendPreloadSelect(s...)
}

type Join[Q interface{ AppendJoin(clause.Join) }] clause.Join

func (j Join[Q]) Edit(q Q) {
	q.AppendJoin(clause.Join(j))
}

type Where[Q interface{ AppendWhere(e any) }] struct {
	E bob.Expression
}

func (w Where[Q]) Edit(q Q) {
	q.AppendWhere(w.E)
}

type WhereCurrentOf[Q interface{ SetCurrentOf(string) }] struct {
	Cursor string
}

func (w WhereCurrentOf[Q]) Edit(q Q) {
	q.SetCurrentOf(w.Cursor)
}

type GroupBy[Q interface{ AppendGroup(any) }] struct {
	E any
}

func (f GroupBy[Q]) Edit(q Q) {
	q.AppendGroup(f.E)
}

type GroupWith[Q interface{ SetGroupWith(string) }] string

func (f GroupWith[Q]) Edit(q Q) {
	q.SetGroupWith(string(f))
}

type GroupByDistinct[Q interface{ SetGroupByDistinct(bool) }] bool

func (f GroupByDistinct[Q]) Edit(q Q) {
	q.SetGroupByDistinct(bool(f))
}

type Having[Q interface{ AppendHaving(e ...any) }] []any

func (d Having[Q]) Edit(q Q) {
	q.AppendHaving(d...)
}

type Window[Q interface{ SetWindow(clause.Window) }] clause.Window

func (w Window[Q]) Edit(q Q) {
	q.SetWindow(clause.Window(w))
}

type NamedWindow[Q interface{ AppendWindow(bob.Expression) }] clause.NamedWindow

func (w NamedWindow[Q]) Edit(q Q) {
	q.AppendWindow(clause.NamedWindow(w))
}

type OrderBy[Q interface{ AppendOrder(clause.OrderDef) }] clause.OrderDef

func (f OrderBy[Q]) Edit(q Q) {
	q.AppendOrder(clause.OrderDef(f))
}

type Limit[Q interface{ SetLimit(limit any) }] struct {
	Count any
}

func (f Limit[Q]) Edit(q Q) {
	q.SetLimit(f.Count)
}

type Offset[Q interface{ SetOffset(offset any) }] struct {
	Count any
}

func (f Offset[Q]) Edit(q Q) {
	q.SetOffset(f.Count)
}

type Fetch[Q interface{ SetFetch(clause.Fetch) }] clause.Fetch

func (f Fetch[Q]) Edit(q Q) {
	q.SetFetch(clause.Fetch(f))
}

type Combine[Q interface{ AppendCombine(clause.Combine) }] clause.Combine

func (f Combine[Q]) Edit(q Q) {
	q.AppendCombine(clause.Combine(f))
}

type For[Q interface{ SetFor(clause.Lock) }] clause.Lock

func (f For[Q]) Edit(q Q) {
	q.SetFor(clause.Lock(f))
}

type Values[Q interface{ AppendValues(vals ...bob.Expression) }] []bob.Expression

func (s Values[Q]) Edit(q Q) {
	q.AppendValues(s...)
}

type Rows[Q interface{ AppendValues(vals ...bob.Expression) }] [][]bob.Expression

func (r Rows[Q]) Edit(q Q) {
	for _, row := range r {
		q.AppendValues(row...)
	}
}

// BatchRows gives an INSERT its rows in one call.
type BatchRows[Q interface{ AppendRows(...[]bob.Expression) }] [][]bob.Expression

func (r BatchRows[Q]) Edit(q Q) { q.AppendRows(r...) }

type Returning[Q interface{ AppendReturning(vals ...any) }] []any

func (s Returning[Q]) Edit(q Q) {
	q.AppendReturning(s...)
}

func (s Returning[Q]) WithOldAs(alias string) returningWithAliases[Q] {
	return returningWithAliases[Q]{
		clauses:  append([]any(nil), s...),
		oldAlias: alias,
	}
}

func (s Returning[Q]) WithNewAs(alias string) returningWithAliases[Q] {
	return returningWithAliases[Q]{
		clauses:  append([]any(nil), s...),
		newAlias: alias,
	}
}

type returningWithAliases[Q interface{ AppendReturning(vals ...any) }] struct {
	clauses  []any
	oldAlias string
	newAlias string
}

func (s returningWithAliases[Q]) Edit(q Q) {
	q.AppendReturning(s.clauses...)

	if s.oldAlias != "" {
		if setter, ok := any(q).(interface{ SetOldAlias(string) }); ok {
			setter.SetOldAlias(s.oldAlias)
		}
	}

	if s.newAlias != "" {
		if setter, ok := any(q).(interface{ SetNewAlias(string) }); ok {
			setter.SetNewAlias(s.newAlias)
		}
	}
}

func (s returningWithAliases[Q]) WithOldAs(alias string) returningWithAliases[Q] {
	s.oldAlias = alias
	return s
}

func (s returningWithAliases[Q]) WithNewAs(alias string) returningWithAliases[Q] {
	s.newAlias = alias
	return s
}

// Set builds a single-column assignment for SET / DO UPDATE SET clauses.
// Col is rendered via bob.Express; use expr.Quote (or dialect.Quote) for SQL identifiers.
type Set[Q interface{ AppendSet(clauses ...any) }] struct {
	Col any
}

func (s Set[Q]) To(to any) set[Q] {
	return set[Q]{expr: expr.OP("=", s.Col, to)}
}

func (s Set[Q]) ToArg(to any) set[Q] {
	return set[Q]{expr: expr.OP("=", s.Col, expr.Arg(to))}
}

// set is a single SET assignment usable as bob.Mod (Edit) or bob.Expression (WriteSQL).
type set[Q interface{ AppendSet(clauses ...any) }] struct {
	expr bob.Expression
}

func (s set[Q]) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.Express(ctx, w, d, start, s.expr)
}

func (s set[Q]) Edit(q Q) {
	q.AppendSet(s.expr)
}

type Hook[Q interface{ AppendHooks(...bob.Hook[Q]) }] []bob.Hook[Q]

func (h Hook[Q]) Edit(q Q) {
	q.AppendHooks(h...)
}
