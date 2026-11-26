package main

// Features name the appendable state fields and the editor methods that use them.
// A method without a body forwards to the statement.
func featureSet() map[string]feature {
	return map[string]feature{
		"with": {"CTEs", `
AppendCTE(x bob.Expression)
SetRecursive(v bool)
`},
		"select": {"SelectList.Columns PreloadColumns", `
CountSelectCols() int
SetSelect(xs ...any)
AppendSelect(xs ...any)
SetPreloadSelect(xs ...any)
AppendPreloadSelect(xs ...any)
`},
		"where": {"Where.Conditions", `
AppendWhere(x any)
SetWhereAt(i int, x any) | if !e.ownedWhere { q.Where.Conditions = slices.Clone(q.Where.Conditions); e.ownedWhere = true }; q.Where.Conditions[i] = x
`},
		"group": {"Groups", `
AppendGroup(x any)
SetGroups(xs ...any)
ResetGroupBy(xs ...any) | q.GroupBy = clause.GroupBy{}; q.SetGroups(xs...)
`},
		"rollup": {"", `
SetGroupWith(v string)
`},
		"groupDistinct": {"", `
SetGroupByDistinct(v bool)
`},
		"having": {"Having.Conditions", `
AppendHaving(xs ...any)
`},
		"windows": {"Windows.Windows", `
AppendWindow(x bob.Expression)
`},
		"combines": {"Queries", `
AppendCombine(x clause.Combine)
`},
		"order": {"OrderBy.Expressions", `
AppendOrder(x bob.Expression)
ClearOrderBy()
`},
		"limit": {"", `
SetLimit(x any)
`},
		"offset": {"", `
SetOffset(x any)
`},
		"fetch": {"", `
SetFetch(x clause.Fetch)
`},
		"locks": {"Locks.Locks", `
AppendLock(x bob.Expression)
`},
		"combined": {"CombinedOrder.Expressions", `
AppendCombinedOrder(x bob.Expression) | q.CombinedOrder.AppendOrder(x)
SetCombinedLimit(x any) | q.CombinedLimit.SetLimit(x)
SetCombinedOffset(x any) | q.CombinedOffset.SetOffset(x)
`},
		"combinedFetch": {"", `
SetCombinedFetch(x clause.Fetch) | q.CombinedFetch.SetFetch(x)
`},
		"table": {"", `
SetTable(x any)
SetTableAlias(alias string, columns ...string)
`},
		"join": {"TableRef.Joins", `
AppendJoin(j clause.Join)
`},
		"insertTable": {"", `
SetTable(x any)
SetInsertAlias(alias string) | q.TableRef.Alias = alias
SetTableColumns(cols ...string) | q.TableRef.Columns = slices.Clone(cols)
SetTableRef(ref clause.TableRef) | q.TableRef = ref.Copy()
`},
		"target": {"", `
SetTarget(ref clause.TableRef) | q.Table = ref.Copy()
SetTargetName(x any) | q.Table.Expression = x
SetTargetAlias(alias string) | q.Table.Alias = alias
`},
		"indexed": {"", `
SetIndexedBy(p *string)
`},
		"targetIndexed": {"", `
SetTargetIndexedBy(p *string) | q.Table.SetIndexedBy(p)
`},
		"mysqlFrom": {"TableRef.Partitions TableRef.IndexHints", `
SetLateral(v bool)
AppendPartition(xs ...string)
AppendIndexHint(x clause.IndexHint)
`},
		"pgFrom": {"", `
AppendTableRef(x clause.TableRef)
`},
		"pgMultiFrom": {"", `
AppendTableRef(x clause.TableRef) | e.ownRefs(); q.AppendTableRef(x)
AppendJoin(x clause.Join) | e.ownRefs(); q.AppendJoin(x)
`},
		"set": {"Set.Set", `
AppendSet(xs ...any)
`},
		"only": {"", `
SetOnly(v bool) | q.Only = v
`},
		"returning": {"Returning.Expressions", `
AppendReturning(xs ...any)
SetReturning(xs ...any)
HasReturning() bool
`},
		"returningAliases": {"", `
SetOldAlias(v string)
SetNewAlias(v string)
`},
		"callbacks": {"Mods Hooks", `
AppendHooks(fs ...func(context.Context, bob.Executor) (context.Context, error))
SetHooks(fs ...func(context.Context, bob.Executor) (context.Context, error))
AppendLoader(fs ...bob.Loader)
SetLoaders(fs ...bob.Loader)
AppendMapperMod(f scan.MapperMod)
SetMapperMods(fs ...scan.MapperMod)
NextUniqueInt() uint64
AppendContextualMod(ms ...bob.ContextualMod[EDITOR])
AppendContextualModFunc(f func(context.Context, EDITOR) (context.Context, error))
`},
		"conflict": {"", `
SetConflict(x bob.Expression)
`},
		"or": {"", `
SetOr(v string)
`},
		"cursor": {"", `
SetCurrentOf(v string)
`},
		"insert": {"Vals", `
AppendValues(xs ...bob.Expression)
AppendRows(xs ...[]bob.Expression)
SetInsertQuery(x bob.Query) | q.Query = x
ColumnNames() []string | return slices.Clone(q.TableRef.Columns)
RowCount() int | return len(q.Vals)
SetValue(row int, col int, x bob.Expression) | if !e.ownedRows { q.Vals = internal.CloneRows(q.Vals); e.ownedRows = true }; q.Vals[row][col] = x
`},
		"values": {"RowVals", `
AppendValues(xs ...bob.Expression)
RowCount() int | return len(q.RowVals)
SetValue(row int, col int, x bob.Expression) | if !e.ownedRows { q.RowVals = internal.CloneRows(q.RowVals); e.ownedRows = true }; q.RowVals[row][col] = x
`},
		"overriding": {"", `
SetOverriding(v OverridingType) | q.Overriding = v
`},
		"pgDistinct": {"Distinct.On", `
SetDistinctOn(xs ...any) | q.Distinct.On = append([]any{}, xs...)
AppendDistinctOn(x any) | q.Distinct.On = append(q.Distinct.On, x)
`},
		"distinct": {"", `
SetDistinct(v bool) | q.Distinct = v
`},
		"merge": {"When", `
SetUsing(x MergeUsing) | q.Using = x
SetUsingOnly(v bool) | q.Using.Only = v
SetUsingSource(x any) | q.Using.Source = x
SetUsingCondition(x bob.Expression) | q.Using.Condition = x
AppendWhen(x MergeWhen) | x.Action.Columns = slices.Clone(x.Action.Columns); x.Action.Set = slices.Clone(x.Action.Set); x.Action.Values = slices.Clone(x.Action.Values); q.When = append(q.When, x)
`},
		"hint": {"hints.hints", `
AppendHint(v string)
`},
		"modAny": {"modifiers.modifiers", `
AppendModifier(v any)
`},
		"modString": {"modifiers.modifiers", `
AppendModifier(v string)
`},
		"into": {"", `
SetInto(x any)
`},
		"mysqlInsert": {"TableRef.Partitions Sets DuplicateKeyUpdate.Set", `
AppendPartition(xs ...string)
SetRowAlias(v string) | q.RowAlias = v
SetColumnAliases(xs ...string) | q.ColumnAlias = slices.Clone(xs)
AppendAssignment(x Set) | q.Sets = append(q.Sets, x)
AppendDuplicateKeyUpdate(xs ...any) | q.DuplicateKeyUpdate.AppendSet(xs...)
HasInsertQuery() bool | return q.Query != nil
HasDuplicateKeyUpdate() bool | return len(q.DuplicateKeyUpdate.Set) != 0
HasAssignments() bool | return len(q.Sets) != 0
Rows() []clause.Value | return internal.CloneRows(q.Vals)
SetRows(xs ...[]bob.Expression) | q.Vals = nil; q.AppendRows(xs...)
ResolveContextualMods(ctx context.Context) (context.Context, error) | ctx, err := q.RunContextualMods(ctx, e); q.Mods = nil; return ctx, err
`},
		"mysqlDelete": {"Tables Partitions", `
AppendTable(x bob.Expression)
AppendDeleteTable(ref clause.TableRef) | q.Tables = append(q.Tables, ref.Copy())
SetPartitions(xs ...string) | q.Partitions = slices.Clone(xs)
`},
	}
}

func dialects() map[string][]statement {
	return map[string][]statement{
		"psql": {
			{
				"Select",
				"with select table join where group having windows combines order limit offset callbacks groupDistinct locks fetch combined combinedFetch pgDistinct pgFrom",
			},
			{"Insert", "with insertTable insert conflict returning returningAliases callbacks overriding"},
			{"Update", "with target set where only cursor pgMultiFrom returning returningAliases callbacks"},
			{"Delete", "with target where only cursor pgMultiFrom returning returningAliases callbacks"},
			{"Values", "values order limit offset fetch"},
			{"Merge", "with target only merge returning returningAliases callbacks"},
		},
		"mysql": {
			{
				"Select",
				"with select table join where group having windows combines order limit offset callbacks rollup hint modAny into locks combined mysqlFrom",
			},
			{"Insert", "insertTable insert hint modString mysqlInsert callbacks"},
			{"Update", "with table join where set order limit hint modAny mysqlFrom callbacks"},
			{"Delete", "with table join where order limit hint modString mysqlFrom mysqlDelete callbacks"},
			{"Values", "values order limit"},
		},
		"sqlite": {
			{"Select", "with select table join where group having windows combines order limit offset callbacks distinct indexed"},
			{"Insert", "with insertTable insert conflict or returning callbacks"},
			{"Update", "with target targetIndexed table join indexed or where set returning limit offset callbacks"},
			{"Delete", "with table indexed where returning limit offset callbacks"},
			{"Values", "values"},
		},
	}
}
