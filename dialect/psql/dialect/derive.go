package dialect

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func (base *SelectQuery) Derive(queryMods ...bob.Mod[*SelectQuery]) (*SelectQuery, bool) {
	next := *base
	var cloneWith, cloneSelect, clonePreload, cloneWhere, cloneGroup, cloneHaving, cloneOrder, cloneWindows, cloneLocks, cloneJoins, cloneCombines, cloneCombinedOrder bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*SelectQuery]:
			next.With.Recursive = bool(m)
		case CTEChain[*SelectQuery]:
			if !cloneWith {
				next.With.CTEs = cloneExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case mods.Distinct[*SelectQuery]:
			next.SetDistinctValues([]any(m))
		case mods.Select[*SelectQuery]:
			if !cloneSelect {
				next.SelectList.Columns = cloneAnySlice(base.SelectList.Columns)
				cloneSelect = true
			}
			next.SelectList.Columns = append(next.SelectList.Columns, []any(m)...)
		case mods.Preload[*SelectQuery]:
			if !clonePreload {
				next.SelectList.PreloadColumns = cloneAnySlice(base.SelectList.PreloadColumns)
				clonePreload = true
			}
			next.SelectList.PreloadColumns = append(next.SelectList.PreloadColumns, []any(m)...)
		case mods.Where[*SelectQuery]:
			if !cloneWhere {
				next.Where.Conditions = cloneAnySlice(base.Where.Conditions)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.GroupBy[*SelectQuery]:
			if !cloneGroup {
				next.GroupBy.Groups = cloneAnySlice(base.GroupBy.Groups)
				cloneGroup = true
			}
			next.GroupBy.Groups = append(next.GroupBy.Groups, m.E)
		case mods.GroupByDistinct[*SelectQuery]:
			next.GroupBy.Distinct = bool(m)
		case mods.GroupWith[*SelectQuery]:
			next.GroupBy.With = string(m)
		case mods.Having[*SelectQuery]:
			if !cloneHaving {
				next.Having.Conditions = cloneAnySlice(base.Having.Conditions)
				cloneHaving = true
			}
			next.Having.Conditions = append(next.Having.Conditions, []any(m)...)
		case mods.Limit[*SelectQuery]:
			next.Limit.Count = m.Count
		case mods.Offset[*SelectQuery]:
			next.Offset.Count = m.Count
		case mods.Fetch[*SelectQuery]:
			next.Fetch = clause.Fetch(m)
		case OrderBy[*SelectQuery]:
			if !cloneOrder {
				next.OrderBy.Expressions = cloneExpressionSlice(base.OrderBy.Expressions)
				cloneOrder = true
			}
			next.OrderBy.Expressions = append(next.OrderBy.Expressions, m())
		case mods.Join[*SelectQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, clause.Join(m))
		case CrossJoinChain[*SelectQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, m())
		case mods.NamedWindow[*SelectQuery]:
			if !cloneWindows {
				next.Windows.Windows = cloneExpressionSlice(base.Windows.Windows)
				cloneWindows = true
			}
			next.Windows.Windows = append(next.Windows.Windows, clause.NamedWindow(m))
		case LockChain[*SelectQuery]:
			if !cloneLocks {
				next.Locks.Locks = cloneExpressionSlice(base.Locks.Locks)
				cloneLocks = true
			}
			next.Locks.Locks = append(next.Locks.Locks, m())
		case mods.Combine[*SelectQuery]:
			if !cloneCombines {
				next.Combines.Queries = append([]clause.Combine(nil), base.Combines.Queries...)
				cloneCombines = true
			}
			next.Combines.Queries = append(next.Combines.Queries, clause.Combine(m))
		case OrderCombined:
			if !cloneCombinedOrder {
				next.CombinedOrder.Expressions = cloneExpressionSlice(base.CombinedOrder.Expressions)
				cloneCombinedOrder = true
			}
			next.CombinedOrder.Expressions = append(next.CombinedOrder.Expressions, m())
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

func (base *UpdateQuery) Derive(queryMods ...bob.Mod[*UpdateQuery]) (*UpdateQuery, bool) {
	next := *base
	var cloneWith, cloneSet, cloneWhere, cloneReturning, cloneJoins bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*UpdateQuery]:
			next.With.Recursive = bool(m)
		case CTEChain[*UpdateQuery]:
			if !cloneWith {
				next.With.CTEs = cloneExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case UpdateOnly:
			next.Only = bool(m)
		case UpdateTable:
			next.Table = cloneTableRef(clause.TableRef(m))
		case mods.SetExprs[*UpdateQuery]:
			if !cloneSet {
				next.Set.Set = cloneAnySlice(base.Set.Set)
				cloneSet = true
			}
			next.Set.Set = append(next.Set.Set, []any(m)...)
		case mods.Where[*UpdateQuery]:
			if !cloneWhere {
				next.Where.Conditions = cloneAnySlice(base.Where.Conditions)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.Returning[*UpdateQuery]:
			if !cloneReturning {
				next.Returning.Expressions = cloneAnySlice(base.Returning.Expressions)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case FromChain[*UpdateQuery]:
			next.TableRef = cloneTableRef(m())
		case mods.Join[*UpdateQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, clause.Join(m))
		case CrossJoinChain[*UpdateQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, m())
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
			if !cloneWith {
				next.With.CTEs = cloneExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case DeleteOnly:
			next.Only = bool(m)
		case DeleteTable:
			next.Table = cloneTableRef(clause.TableRef(m))
		case mods.Where[*DeleteQuery]:
			if !cloneWhere {
				next.Where.Conditions = cloneAnySlice(base.Where.Conditions)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.Returning[*DeleteQuery]:
			if !cloneReturning {
				next.Returning.Expressions = cloneAnySlice(base.Returning.Expressions)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case FromChain[*DeleteQuery]:
			next.TableRef = cloneTableRef(m())
		case mods.Join[*DeleteQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, clause.Join(m))
		case CrossJoinChain[*DeleteQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, m())
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
			if !cloneWith {
				next.With.CTEs = cloneExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case InsertTable:
			next.TableRef = cloneTableRef(clause.TableRef(m))
		case mods.Overriding[*InsertQuery]:
			next.Overriding = string(m)
		case mods.QuerySource[*InsertQuery]:
			next.Values.Query = m.Query
		case mods.Returning[*InsertQuery]:
			if !cloneReturning {
				next.Returning.Expressions = cloneAnySlice(base.Returning.Expressions)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case mods.Values[*InsertQuery]:
			if !cloneVals {
				next.Values.Vals = append([]clause.Value(nil), base.Values.Vals...)
				cloneVals = true
			}
			next.Values.Vals = append(next.Values.Vals, clause.Value(m))
		case mods.Rows[*InsertQuery]:
			if !cloneVals {
				next.Values.Vals = append([]clause.Value(nil), base.Values.Vals...)
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
