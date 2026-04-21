package dialect

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func cloneSlice[T any](values []T) []T {
	if values == nil {
		return nil
	}
	return append([]T(nil), values...)
}

func appendDerived[T any](target *[]T, base []T, cloned *bool, values ...T) {
	if !*cloned {
		*target = cloneSlice(base)
		*cloned = true
	}

	*target = append(*target, values...)
}

func (base *SelectQuery) Derive(queryMods ...bob.Mod[*SelectQuery]) (*SelectQuery, bool) {
	next := *base
	var cloneWith, cloneSelect, clonePreload, cloneWhere, cloneGroup, cloneHaving, cloneOrder, cloneWindows, cloneLocks, cloneJoins, cloneCombines, cloneCombinedOrder bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*SelectQuery]:
			next.With.Recursive = bool(m)
		case CTEChain[*SelectQuery]:
			appendDerived[bob.Expression](&next.With.CTEs, base.With.CTEs, &cloneWith, m())
		case mods.Distinct[*SelectQuery]:
			next.SetDistinctValues([]any(m))
		case mods.Select[*SelectQuery]:
			appendDerived(&next.SelectList.Columns, base.SelectList.Columns, &cloneSelect, []any(m)...)
		case mods.Preload[*SelectQuery]:
			appendDerived(&next.SelectList.PreloadColumns, base.SelectList.PreloadColumns, &clonePreload, []any(m)...)
		case mods.Where[*SelectQuery]:
			appendDerived[any](&next.Where.Conditions, base.Where.Conditions, &cloneWhere, m.E)
		case mods.GroupBy[*SelectQuery]:
			appendDerived(&next.GroupBy.Groups, base.GroupBy.Groups, &cloneGroup, m.E)
		case mods.GroupByDistinct[*SelectQuery]:
			next.GroupBy.Distinct = bool(m)
		case mods.GroupWith[*SelectQuery]:
			next.GroupBy.With = string(m)
		case mods.Having[*SelectQuery]:
			appendDerived(&next.Having.Conditions, base.Having.Conditions, &cloneHaving, []any(m)...)
		case mods.Limit[*SelectQuery]:
			next.Limit.Count = m.Count
		case mods.Offset[*SelectQuery]:
			next.Offset.Count = m.Count
		case mods.Fetch[*SelectQuery]:
			next.Fetch = clause.Fetch(m)
		case OrderBy[*SelectQuery]:
			appendDerived[bob.Expression](&next.OrderBy.Expressions, base.OrderBy.Expressions, &cloneOrder, m())
		case mods.Join[*SelectQuery]:
			appendDerived(&next.TableRef.Joins, base.TableRef.Joins, &cloneJoins, clause.Join(m))
		case CrossJoinChain[*SelectQuery]:
			appendDerived(&next.TableRef.Joins, base.TableRef.Joins, &cloneJoins, m())
		case mods.NamedWindow[*SelectQuery]:
			appendDerived[bob.Expression](&next.Windows.Windows, base.Windows.Windows, &cloneWindows, clause.NamedWindow(m))
		case LockChain[*SelectQuery]:
			appendDerived[bob.Expression](&next.Locks.Locks, base.Locks.Locks, &cloneLocks, m())
		case mods.Combine[*SelectQuery]:
			appendDerived(&next.Combines.Queries, base.Combines.Queries, &cloneCombines, clause.Combine(m))
		case OrderCombined:
			appendDerived[bob.Expression](&next.CombinedOrder.Expressions, base.CombinedOrder.Expressions, &cloneCombinedOrder, m())
		case LimitCombined:
			next.CombinedLimit.Count = m.Count
		case OffsetCombined:
			next.CombinedOffset.Count = m.Count
		case FetchCombined:
			next.CombinedFetch.Count = m.Count
			next.CombinedFetch.WithTies = m.WithTies
		case FromChain[*SelectQuery]:
			next.TableRef = cloneTableRef(m())
		default:
			return nil, false
		}
	}

	return &next, true
}

func (base *DeleteQuery) Derive(queryMods ...bob.Mod[*DeleteQuery]) (*DeleteQuery, bool) {
	next := *base
	var cloneWith, cloneWhere, cloneReturning, cloneJoins bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*DeleteQuery]:
			next.With.Recursive = bool(m)
		case CTEChain[*DeleteQuery]:
			appendDerived[bob.Expression](&next.With.CTEs, base.With.CTEs, &cloneWith, m())
		case mods.TargetOnly[*DeleteQuery]:
			next.Only = bool(m)
		case mods.TargetTable[*DeleteQuery]:
			next.Table = cloneTableRef(clause.TableRef(m))
		case mods.Where[*DeleteQuery]:
			appendDerived[any](&next.Where.Conditions, base.Where.Conditions, &cloneWhere, m.E)
		case mods.Returning[*DeleteQuery]:
			appendDerived(&next.Returning.Expressions, base.Returning.Expressions, &cloneReturning, []any(m)...)
		case FromChain[*DeleteQuery]:
			next.TableRef = cloneTableRef(m())
		case mods.Join[*DeleteQuery]:
			appendDerived(&next.TableRef.Joins, base.TableRef.Joins, &cloneJoins, clause.Join(m))
		case CrossJoinChain[*DeleteQuery]:
			appendDerived(&next.TableRef.Joins, base.TableRef.Joins, &cloneJoins, m())
		default:
			return nil, false
		}
	}

	return &next, true
}

func (base *InsertQuery) Derive(queryMods ...bob.Mod[*InsertQuery]) (*InsertQuery, bool) {
	next := *base
	var cloneWith, cloneReturning, cloneVals bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*InsertQuery]:
			next.With.Recursive = bool(m)
		case CTEChain[*InsertQuery]:
			appendDerived[bob.Expression](&next.With.CTEs, base.With.CTEs, &cloneWith, m())
		case mods.TargetTable[*InsertQuery]:
			next.TableRef = cloneTableRef(clause.TableRef(m))
		case mods.Overriding[*InsertQuery]:
			next.Overriding = string(m)
		case mods.QuerySource[*InsertQuery]:
			next.Values.Query = m.Query
		case mods.Returning[*InsertQuery]:
			appendDerived(&next.Returning.Expressions, base.Returning.Expressions, &cloneReturning, []any(m)...)
		case mods.Values[*InsertQuery]:
			appendDerived(&next.Values.Vals, base.Values.Vals, &cloneVals, clause.Value(m))
		case mods.Rows[*InsertQuery]:
			if !cloneVals {
				next.Values.Vals = cloneSlice(base.Values.Vals)
				cloneVals = true
			}
			for _, row := range m {
				next.Values.Vals = append(next.Values.Vals, clause.Value(row))
			}
		case mods.Conflict[*InsertQuery]:
			next.Conflict.Expression = m()
		default:
			return nil, false
		}
	}

	return &next, true
}
