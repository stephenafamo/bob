package dialect

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

func cloneAnySlice(values []any) []any {
	if values == nil {
		return nil
	}
	return append([]any(nil), values...)
}

func cloneStringSlice(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string(nil), values...)
}

func cloneExpressionSlice(values []bob.Expression) []bob.Expression {
	if values == nil {
		return nil
	}
	return append([]bob.Expression(nil), values...)
}

func cloneWith(with clause.With) clause.With {
	return clause.With{
		Recursive: with.Recursive,
		CTEs:      cloneExpressionSlice(with.CTEs),
	}
}

func cloneSelectList(list clause.SelectList) clause.SelectList {
	return clause.SelectList{
		Columns:        cloneAnySlice(list.Columns),
		PreloadColumns: cloneAnySlice(list.PreloadColumns),
	}
}

func cloneWhere(where clause.Where) clause.Where {
	return clause.Where{Conditions: cloneAnySlice(where.Conditions)}
}

func cloneGroupBy(groupBy clause.GroupBy) clause.GroupBy {
	return clause.GroupBy{
		Groups:   cloneAnySlice(groupBy.Groups),
		Distinct: groupBy.Distinct,
		With:     groupBy.With,
	}
}

func cloneHaving(having clause.Having) clause.Having {
	return clause.Having{Conditions: cloneAnySlice(having.Conditions)}
}

func cloneWindows(windows clause.Windows) clause.Windows {
	return clause.Windows{Windows: cloneExpressionSlice(windows.Windows)}
}

func cloneOrderBy(orderBy clause.OrderBy) clause.OrderBy {
	return clause.OrderBy{Expressions: cloneExpressionSlice(orderBy.Expressions)}
}

func cloneLocks(locks clause.Locks) clause.Locks {
	return clause.Locks{Locks: cloneExpressionSlice(locks.Locks)}
}

func cloneLimit(limit clause.Limit) clause.Limit {
	return clause.Limit{Count: limit.Count}
}

func cloneOffset(offset clause.Offset) clause.Offset {
	return clause.Offset{Count: offset.Count}
}

func cloneFetch(fetch clause.Fetch) clause.Fetch {
	return clause.Fetch{
		Count:    fetch.Count,
		WithTies: fetch.WithTies,
	}
}

func cloneReturning(returning clause.Returning) clause.Returning {
	return clause.Returning{Expressions: cloneAnySlice(returning.Expressions)}
}

func cloneSet(set clause.Set) clause.Set {
	return clause.Set{Set: cloneAnySlice(set.Set)}
}

func cloneConflict(conflict clause.Conflict) clause.Conflict {
	return clause.Conflict{Expression: conflict.Expression}
}

func cloneValues(values clause.Values) clause.Values {
	cloned := clause.Values{
		Query: values.Query,
		Vals:  make([]clause.Value, 0, len(values.Vals)),
	}

	for _, row := range values.Vals {
		cloned.Vals = append(cloned.Vals, append(clause.Value(nil), row...))
	}

	return cloned
}

func cloneCombines(combines clause.Combines) clause.Combines {
	if combines.Queries == nil {
		return clause.Combines{}
	}

	queries := make([]clause.Combine, 0, len(combines.Queries))
	for _, combine := range combines.Queries {
		queries = append(queries, clause.Combine{
			Strategy: combine.Strategy,
			Query:    combine.Query,
			All:      combine.All,
		})
	}

	return clause.Combines{Queries: queries}
}

func cloneTableRef(ref clause.TableRef) clause.TableRef {
	var indexedBy *string
	if ref.IndexedBy != nil {
		indexed := *ref.IndexedBy
		indexedBy = &indexed
	}

	indexHints := make([]clause.IndexHint, 0, len(ref.IndexHints))
	for _, hint := range ref.IndexHints {
		indexHints = append(indexHints, clause.IndexHint{
			Type:    hint.Type,
			Indexes: cloneStringSlice(hint.Indexes),
			For:     hint.For,
		})
	}

	joins := make([]clause.Join, 0, len(ref.Joins))
	for _, join := range ref.Joins {
		joins = append(joins, clause.Join{
			Type:    join.Type,
			Natural: join.Natural,
			To:      cloneTableRef(join.To),
			On:      cloneExpressionSlice(join.On),
			Using:   cloneStringSlice(join.Using),
		})
	}

	return clause.TableRef{
		Expression:     ref.Expression,
		Alias:          ref.Alias,
		Columns:        cloneStringSlice(ref.Columns),
		Only:           ref.Only,
		Lateral:        ref.Lateral,
		WithOrdinality: ref.WithOrdinality,
		IndexedBy:      indexedBy,
		Partitions:     cloneStringSlice(ref.Partitions),
		IndexHints:     indexHints,
		Joins:          joins,
	}
}

func cloneLoad(load bob.Load) bob.Load {
	var cloned bob.Load
	cloned.SetLoaders(load.GetLoaders()...)
	cloned.SetMapperMods(load.GetMapperMods()...)
	return cloned
}

func cloneEmbeddedHook(hook bob.EmbeddedHook) bob.EmbeddedHook {
	return bob.EmbeddedHook{
		Hooks: append([]func(context.Context, bob.Executor) (context.Context, error){}, hook.Hooks...),
	}
}

func cloneContextualModdable[T any](mods bob.ContextualModdable[T]) bob.ContextualModdable[T] {
	return bob.ContextualModdable[T]{
		Mods: append([]bob.ContextualMod[T](nil), mods.Mods...),
	}
}

func (s *SelectQuery) Clone() *SelectQuery {
	if s == nil {
		return nil
	}

	return &SelectQuery{
		With:               cloneWith(s.With),
		SelectList:         cloneSelectList(s.SelectList),
		Distinct:           Distinct{On: cloneAnySlice(s.Distinct.On)},
		TableRef:           cloneTableRef(s.TableRef),
		Where:              cloneWhere(s.Where),
		GroupBy:            cloneGroupBy(s.GroupBy),
		Having:             cloneHaving(s.Having),
		Windows:            cloneWindows(s.Windows),
		Combines:           cloneCombines(s.Combines),
		OrderBy:            cloneOrderBy(s.OrderBy),
		Limit:              cloneLimit(s.Limit),
		Offset:             cloneOffset(s.Offset),
		Fetch:              cloneFetch(s.Fetch),
		Locks:              cloneLocks(s.Locks),
		Load:               cloneLoad(s.Load),
		EmbeddedHook:       cloneEmbeddedHook(s.EmbeddedHook),
		ContextualModdable: cloneContextualModdable(s.ContextualModdable),
		CombinedOrder:      cloneOrderBy(s.CombinedOrder),
		CombinedLimit:      cloneLimit(s.CombinedLimit),
		CombinedFetch:      cloneFetch(s.CombinedFetch),
		CombinedOffset:     cloneOffset(s.CombinedOffset),
	}
}

func (u *UpdateQuery) Clone() *UpdateQuery {
	if u == nil {
		return nil
	}

	return &UpdateQuery{
		With:               cloneWith(u.With),
		Only:               u.Only,
		Table:              cloneTableRef(u.Table),
		Set:                cloneSet(u.Set),
		TableRef:           cloneTableRef(u.TableRef),
		Where:              cloneWhere(u.Where),
		Returning:          cloneReturning(u.Returning),
		Load:               cloneLoad(u.Load),
		EmbeddedHook:       cloneEmbeddedHook(u.EmbeddedHook),
		ContextualModdable: cloneContextualModdable(u.ContextualModdable),
	}
}

func (d *DeleteQuery) Clone() *DeleteQuery {
	if d == nil {
		return nil
	}

	return &DeleteQuery{
		With:               cloneWith(d.With),
		Only:               d.Only,
		Table:              cloneTableRef(d.Table),
		TableRef:           cloneTableRef(d.TableRef),
		Where:              cloneWhere(d.Where),
		Returning:          cloneReturning(d.Returning),
		Load:               cloneLoad(d.Load),
		EmbeddedHook:       cloneEmbeddedHook(d.EmbeddedHook),
		ContextualModdable: cloneContextualModdable(d.ContextualModdable),
	}
}

func (i *InsertQuery) Clone() *InsertQuery {
	if i == nil {
		return nil
	}

	return &InsertQuery{
		With:               cloneWith(i.With),
		Overriding:         i.Overriding,
		TableRef:           cloneTableRef(i.TableRef),
		Values:             cloneValues(i.Values),
		Conflict:           cloneConflict(i.Conflict),
		Returning:          cloneReturning(i.Returning),
		Load:               cloneLoad(i.Load),
		EmbeddedHook:       cloneEmbeddedHook(i.EmbeddedHook),
		ContextualModdable: cloneContextualModdable(i.ContextualModdable),
	}
}
