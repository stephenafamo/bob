package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
)

func copyAnySlice(values []any) []any {
	if values == nil {
		return nil
	}
	return append([]any(nil), values...)
}

func copyExpressionSlice(values []bob.Expression) []bob.Expression {
	if values == nil {
		return nil
	}
	return append([]bob.Expression(nil), values...)
}

func copyTableRef(from clause.TableRef) clause.TableRef {
	from.Columns = append([]string(nil), from.Columns...)
	from.Partitions = append([]string(nil), from.Partitions...)
	from.IndexHints = append([]clause.IndexHint(nil), from.IndexHints...)
	from.Joins = append([]clause.Join(nil), from.Joins...)
	for i := range from.Joins {
		from.Joins[i].On = append([]bob.Expression(nil), from.Joins[i].On...)
		from.Joins[i].Using = append([]string(nil), from.Joins[i].Using...)
		from.Joins[i].To = copyTableRef(from.Joins[i].To)
	}
	return from
}

func deriveSelect(base *dialect.SelectQuery, queryMods ...bob.Mod[*dialect.SelectQuery]) (*dialect.SelectQuery, bool) {
	next := *base
	var cloneWith, cloneSelect, clonePreload, cloneWhere, cloneGroup, cloneHaving, cloneOrder, cloneWindows, cloneLocks, cloneJoins, cloneCombines, cloneCombinedOrder bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*dialect.SelectQuery]:
			next.With.Recursive = bool(m)
		case dialect.CTEChain[*dialect.SelectQuery]:
			if !cloneWith {
				next.With.CTEs = copyExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case dialect.DistinctMod:
			next.Distinct.On = append(make([]any, 0, len(m.On)), m.On...)
		case mods.Select[*dialect.SelectQuery]:
			if !cloneSelect {
				next.SelectList.Columns = copyAnySlice(base.SelectList.Columns)
				cloneSelect = true
			}
			next.SelectList.Columns = append(next.SelectList.Columns, []any(m)...)
		case mods.Preload[*dialect.SelectQuery]:
			if !clonePreload {
				next.SelectList.PreloadColumns = copyAnySlice(base.SelectList.PreloadColumns)
				clonePreload = true
			}
			next.SelectList.PreloadColumns = append(next.SelectList.PreloadColumns, []any(m)...)
		case mods.Where[*dialect.SelectQuery]:
			if !cloneWhere {
				next.Where.Conditions = copyAnySlice(base.Where.Conditions)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.GroupBy[*dialect.SelectQuery]:
			if !cloneGroup {
				next.GroupBy.Groups = copyAnySlice(base.GroupBy.Groups)
				cloneGroup = true
			}
			next.GroupBy.Groups = append(next.GroupBy.Groups, m.E)
		case mods.GroupByDistinct[*dialect.SelectQuery]:
			next.GroupBy.Distinct = bool(m)
		case mods.GroupWith[*dialect.SelectQuery]:
			next.GroupBy.With = string(m)
		case mods.Having[*dialect.SelectQuery]:
			if !cloneHaving {
				next.Having.Conditions = copyAnySlice(base.Having.Conditions)
				cloneHaving = true
			}
			next.Having.Conditions = append(next.Having.Conditions, []any(m)...)
		case mods.Limit[*dialect.SelectQuery]:
			next.Limit.Count = m.Count
		case mods.Offset[*dialect.SelectQuery]:
			next.Offset.Count = m.Count
		case mods.Fetch[*dialect.SelectQuery]:
			next.Fetch = clause.Fetch(m)
		case dialect.OrderBy[*dialect.SelectQuery]:
			if !cloneOrder {
				next.OrderBy.Expressions = copyExpressionSlice(base.OrderBy.Expressions)
				cloneOrder = true
			}
			next.OrderBy.Expressions = append(next.OrderBy.Expressions, m())
		case mods.Join[*dialect.SelectQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, clause.Join(m))
		case dialect.CrossJoinChain[*dialect.SelectQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, m())
		case mods.NamedWindow[*dialect.SelectQuery]:
			if !cloneWindows {
				next.Windows.Windows = copyExpressionSlice(base.Windows.Windows)
				cloneWindows = true
			}
			next.Windows.Windows = append(next.Windows.Windows, clause.NamedWindow(m))
		case dialect.LockChain[*dialect.SelectQuery]:
			if !cloneLocks {
				next.Locks.Locks = copyExpressionSlice(base.Locks.Locks)
				cloneLocks = true
			}
			next.Locks.Locks = append(next.Locks.Locks, m())
		case mods.Combine[*dialect.SelectQuery]:
			if !cloneCombines {
				next.Combines.Queries = append([]clause.Combine(nil), base.Combines.Queries...)
				cloneCombines = true
			}
			next.Combines.Queries = append(next.Combines.Queries, clause.Combine(m))
		case dialect.OrderCombined:
			if !cloneCombinedOrder {
				next.CombinedOrder.Expressions = copyExpressionSlice(base.CombinedOrder.Expressions)
				cloneCombinedOrder = true
			}
			next.CombinedOrder.Expressions = append(next.CombinedOrder.Expressions, m())
		case dialect.LimitCombined:
			next.CombinedLimit.Count = m.Count
		case dialect.OffsetCombined:
			next.CombinedOffset.Count = m.Count
		case dialect.FetchCombined:
			next.CombinedFetch.Count = m.Count
			next.CombinedFetch.WithTies = m.WithTies
		case dialect.FromChain[*dialect.SelectQuery]:
			next.TableRef = copyTableRef(m())
		default:
			return nil, false
		}
	}

	return &next, true
}

func deriveUpdate(base *dialect.UpdateQuery, queryMods ...bob.Mod[*dialect.UpdateQuery]) (*dialect.UpdateQuery, bool) {
	next := *base
	var cloneWith, cloneSet, cloneWhere, cloneReturning, cloneJoins bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*dialect.UpdateQuery]:
			next.With.Recursive = bool(m)
		case dialect.CTEChain[*dialect.UpdateQuery]:
			if !cloneWith {
				next.With.CTEs = copyExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case dialect.UpdateOnly:
			next.Only = bool(m)
		case dialect.UpdateTable:
			next.Table = copyTableRef(clause.TableRef(m))
		case dialect.UpdateSet:
			if !cloneSet {
				next.Set.Set = copyAnySlice(base.Set.Set)
				cloneSet = true
			}
			next.Set.Set = append(next.Set.Set, []any(m)...)
		case mods.Where[*dialect.UpdateQuery]:
			if !cloneWhere {
				next.Where.Conditions = copyAnySlice(base.Where.Conditions)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.Returning[*dialect.UpdateQuery]:
			if !cloneReturning {
				next.Returning.Expressions = copyAnySlice(base.Returning.Expressions)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case dialect.FromChain[*dialect.UpdateQuery]:
			next.TableRef = copyTableRef(m())
		case mods.Join[*dialect.UpdateQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, clause.Join(m))
		case dialect.CrossJoinChain[*dialect.UpdateQuery]:
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

func deriveDelete(base *dialect.DeleteQuery, queryMods ...bob.Mod[*dialect.DeleteQuery]) (*dialect.DeleteQuery, bool) {
	next := *base
	var cloneWith, cloneWhere, cloneReturning, cloneJoins bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*dialect.DeleteQuery]:
			next.With.Recursive = bool(m)
		case dialect.CTEChain[*dialect.DeleteQuery]:
			if !cloneWith {
				next.With.CTEs = copyExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case dialect.DeleteOnly:
			next.Only = bool(m)
		case dialect.DeleteTable:
			next.Table = copyTableRef(clause.TableRef(m))
		case mods.Where[*dialect.DeleteQuery]:
			if !cloneWhere {
				next.Where.Conditions = copyAnySlice(base.Where.Conditions)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.Returning[*dialect.DeleteQuery]:
			if !cloneReturning {
				next.Returning.Expressions = copyAnySlice(base.Returning.Expressions)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case dialect.FromChain[*dialect.DeleteQuery]:
			next.TableRef = copyTableRef(m())
		case mods.Join[*dialect.DeleteQuery]:
			if !cloneJoins {
				next.TableRef.Joins = append([]clause.Join(nil), base.TableRef.Joins...)
				cloneJoins = true
			}
			next.TableRef.Joins = append(next.TableRef.Joins, clause.Join(m))
		case dialect.CrossJoinChain[*dialect.DeleteQuery]:
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

func deriveInsert(base *dialect.InsertQuery, queryMods ...bob.Mod[*dialect.InsertQuery]) (*dialect.InsertQuery, bool) {
	next := *base
	var cloneWith, cloneReturning, cloneVals bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Recursive[*dialect.InsertQuery]:
			next.With.Recursive = bool(m)
		case dialect.CTEChain[*dialect.InsertQuery]:
			if !cloneWith {
				next.With.CTEs = copyExpressionSlice(base.With.CTEs)
				cloneWith = true
			}
			next.With.CTEs = append(next.With.CTEs, m())
		case dialect.InsertTable:
			next.TableRef = copyTableRef(clause.TableRef(m))
		case dialect.InsertOverriding:
			next.Overriding = string(m)
		case dialect.InsertQuerySource:
			next.Values.Query = m.Query
		case mods.Returning[*dialect.InsertQuery]:
			if !cloneReturning {
				next.Returning.Expressions = copyAnySlice(base.Returning.Expressions)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case mods.Values[*dialect.InsertQuery]:
			if !cloneVals {
				next.Values.Vals = append([]clause.Value(nil), base.Values.Vals...)
				cloneVals = true
			}
			next.Values.Vals = append(next.Values.Vals, clause.Value(m))
		case mods.Rows[*dialect.InsertQuery]:
			if !cloneVals {
				next.Values.Vals = append([]clause.Value(nil), base.Values.Vals...)
				cloneVals = true
			}
			for _, row := range m {
				next.Values.Vals = append(next.Values.Vals, clause.Value(row))
			}
		case mods.Conflict[*dialect.InsertQuery]:
			next.Conflict.Expression = m()
		default:
			return nil, false
		}
	}

	return &next, true
}
