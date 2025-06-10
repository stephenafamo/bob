package parser

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/internal"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

func (v *visitor) modWithClause(ctx interface {
	WithClause() mysqlparser.IWithClauseContext
}, sb *strings.Builder,
) {
	with := ctx.WithClause()
	if with == nil {
		return
	}

	if with.RECURSIVE() != nil {
		sb.WriteString("q.SetRecursive(true)\n")
	}
	for _, cte := range with.AllCommonTableExpression() {
		v.StmtRules = append(v.StmtRules,
			internal.RecordPoints(
				cte.GetStart().GetStart(),
				cte.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendCTE(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}
}

func (v *visitor) modSelectStatement(ctx mysqlparser.ISelectStatementContext, sb *strings.Builder) [][]string {
	var imports [][]string

	v.modWithClause(ctx, sb)

	// If there is a base without UNION/INTERSECT/EXCEPT
	v.modSelectStatementBase(ctx.SelectStatementBase(), sb)

	// The first SELECT in a UNION/INTERSECT/EXCEPT
	if begin := ctx.SetQuery(); begin != nil {
		if base := begin.SetQueryBase(); base != nil {
			v.modSetQueryBase(base, sb) // treat the same way
		}
		if inParens := begin.SetQueryInParenthesis(); inParens != nil {
			v.modSetQueryInParenthesis(inParens, sb)
		}
	}

	compounds := ctx.AllSetQueryPart()

	if len(compounds) > 0 {
		imports = append(imports, []string{"github.com/stephenafamo/bob/clause"})
	}

	for _, compound := range compounds {
		strategy := strings.ToUpper(compound.GetSetOp().GetText())
		all := compound.ALL() != nil
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			compound.SetQuery().GetStart().GetStart(),
			compound.SetQuery().GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, `
                        q.AppendCombine(clause.Combine{
                            Strategy: "%s",
                            All: %t,
                            Query: bob.BaseQuery[bob.Expression]{
                                Expression: EXPR.subExpr(%d, %d),
                                QueryType: bob.QueryTypeSelect,
                                Dialect: dialect.Dialect,
                            },
                        })
                    `, strategy, all, start, end)
				return nil
			},
		)...)
	}

	finish := ctx.SelectStatementFinish()
	if finish == nil {
		return imports
	}

	if order := finish.OrderByClause(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.CombinedOrder.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limitClause := finish.LimitClause(); limitClause != nil {
		if limitClause.COMMA() != nil {
			v.Err = fmt.Errorf("LIMIT with comma is not supported")
			return nil
		}
		if limit := limitClause.GetLimit(); limit != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				limit.GetStart().GetStart(),
				limit.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.CombinedLimit.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}
		if offset := limitClause.GetOffset(); offset != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				offset.GetStart().GetStart(),
				offset.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.CombinedOffset.SetOffset(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}
	}

	if lock := finish.LockClause(); lock != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			lock.GetStart().GetStart(),
			lock.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendLock(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	return imports
}

func (v *visitor) modSetQueryBase(ctx mysqlparser.ISetQueryBaseContext, sb *strings.Builder) {
	if ctx == nil {
		return
	}
	v.modSelectStatementBase(ctx.SelectStatementBase(), sb)
}

func (v *visitor) modSelectStatementBase(ctx mysqlparser.ISelectStatementBaseContext, sb *strings.Builder) {
	if ctx == nil {
		return
	}

	for _, spec := range ctx.AllSelectSpec() {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			spec.GetStart().GetStart(), spec.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if elements := ctx.SelectElements(); elements != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			elements.GetStart().GetStart(), elements.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendSelect(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if from := ctx.FromClause(); from != nil {
		sources := from.TableSources()
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			sources.GetStart().GetStart(),
			sources.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if where := ctx.WhereClause(); where != nil {
		whereExpr := where.Expression()
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			whereExpr.GetStart().GetStart(),
			whereExpr.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if groupBy := ctx.GroupByClause(); groupBy != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			groupBy.BY().GetSymbol().GetStop()+1,
			groupBy.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendGroup(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if having := ctx.HavingClause(); having != nil {
		havingExpr := having.Expression()
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			havingExpr.GetStart().GetStart(),
			havingExpr.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendHaving(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if window := ctx.WindowClause(); window != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			window.WINDOW().GetSymbol().GetStop()+1,
			window.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWindow(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modSetQueryInParenthesis(ctx mysqlparser.ISetQueryInParenthesisContext, sb *strings.Builder) {
	if ctx == nil {
		return
	}

	v.modSelectStatementBase(ctx.SelectStatementBase(), sb)

	finish := ctx.SelectStatementFinish()
	if finish == nil {
		return
	}

	if order := finish.OrderByClause(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limitClause := finish.LimitClause(); limitClause != nil {
		if limitClause.COMMA() != nil {
			v.Err = fmt.Errorf("LIMIT with comma is not supported")
			return
		}
		if limit := limitClause.GetLimit(); limit != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				limit.GetStart().GetStart(),
				limit.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}
		if offset := limitClause.GetOffset(); offset != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				offset.GetStart().GetStart(),
				offset.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.SetOffset(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}
	}

	if lock := finish.LockClause(); lock != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			lock.GetStart().GetStart(),
			lock.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendLock(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modInsertStatement(ctx mysqlparser.IInsertStatementContext, sb *strings.Builder) {
	if priority := ctx.GetPriority(); priority != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			priority.GetStart(), priority.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if ignore := ctx.IGNORE(); ignore != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			ignore.GetSymbol().GetStart(), ignore.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if partitions := ctx.GetPartitions(); partitions != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			partitions.GetStart().GetStart(), partitions.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendPartition(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if table := ctx.TableName(); table != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			table.GetStart().GetStart(), table.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.TableRef.Expression = EXPR.raw(%d, %d)\n", start, end)
				return nil
			},
		)...)
	}

	if cols := ctx.GetColumns(); cols != nil {
		allCols := cols.AllFullColumnName()
		colNames := make([]string, len(allCols))
		for i, col := range allCols {
			colNames[i] = v.GetName(col)
		}
		fmt.Fprintf(sb, "q.TableRef.Columns = %#v\n", colNames)
	}

	if stmtOrVals := ctx.InsertStatementValue(); stmtOrVals != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			stmtOrVals.GetStart().GetStart(),
			stmtOrVals.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, `q.Query = bob.BaseQuery[bob.Expression]{
						Expression: EXPR.subExpr(%d, %d),
						Dialect: dialect.Dialect,
						QueryType: bob.QueryTypeSelect,
						}
					`, start, end)
				return nil
			},
		)...)
	}

	if sets := ctx.GetSetElement(); sets != nil {
		for _, set := range sets {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				set.ExpressionOrDefault().GetStart().GetStart(),
				set.ExpressionOrDefault().GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb,
						"q.Sets = append(q.Sets, dialect.Set{Col: %q, Val: EXPR.subExpr(%d, %d)})\n",
						v.GetName(set.FullColumnName()), start, end)
					return nil
				},
			)...)
		}
	}

	if tableAlias := ctx.GetTableAlias(); tableAlias != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			tableAlias.GetStart().GetStart(), tableAlias.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.RowAlias = EXPR.raw(%d, %d)\n", start, end)
				return nil
			},
		)...)
	}

	if colAliases := ctx.GetColAlias(); colAliases != nil {
		aliasNames := make([]string, len(colAliases))
		for i, col := range colAliases {
			aliasNames[i] = v.GetName(col)
		}
		fmt.Fprintf(sb, "q.ColumnAlias = %#v\n", aliasNames)
	}

	if duplicate := ctx.GetDuplicated(); len(duplicate) > 0 {
		first := duplicate[0]
		last := duplicate[len(duplicate)-1]
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			first.GetStart().GetStart(), last.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.DuplicateKeyUpdate.AppendSet(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modUpdateStatement(ctx mysqlparser.IUpdateStatementContext, sb *strings.Builder) {
	if single := ctx.SingleUpdateStatement(); single != nil {
		v.modSingleUpdateStatement(single, sb)
		return
	}

	if multi := ctx.MultipleUpdateStatement(); multi != nil {
		v.modMultipleUpdateStatement(multi, sb)
		return
	}
}

func (v *visitor) modSingleUpdateStatement(ctx mysqlparser.ISingleUpdateStatementContext, sb *strings.Builder) {
	v.modWithClause(ctx, sb)

	if priority := ctx.GetPriority(); priority != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			priority.GetStart(), priority.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if ignore := ctx.IGNORE(); ignore != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			ignore.GetSymbol().GetStart(), ignore.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if table := ctx.TableName(); table != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			table.GetStart().GetStart(), table.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.TableRef.Expression = EXPR.raw(%d, %d)\n", start, end)
				return nil
			},
		)...)
	}

	if partitions := ctx.GetPartitions(); partitions != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			partitions.GetStart().GetStart(), partitions.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendPartition(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if tableAlias := ctx.GetTableAlias(); tableAlias != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			tableAlias.GetStart().GetStart(), tableAlias.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.TableRef.Alias = EXPR.raw(%d, %d)\n", start, end)
				return nil
			},
		)...)
	}

	if sets := ctx.AllUpdatedElement(); len(sets) > 0 {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			sets[0].GetStart().GetStart(),
			sets[len(sets)-1].GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendSet(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if where := ctx.GetWhereExpr(); where != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			where.GetStart().GetStart(),
			where.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if order := ctx.OrderByClause(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limit := ctx.LimitClauseAtom(); limit != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			limit.GetStart().GetStart(),
			limit.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modMultipleUpdateStatement(ctx mysqlparser.IMultipleUpdateStatementContext, sb *strings.Builder) {
	v.modWithClause(ctx, sb)

	if priority := ctx.GetPriority(); priority != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			priority.GetStart(), priority.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if ignore := ctx.IGNORE(); ignore != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			ignore.GetSymbol().GetStart(), ignore.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if tables := ctx.TableSources(); tables != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			tables.GetStart().GetStart(),
			tables.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if sets := ctx.AllUpdatedElement(); len(sets) > 0 {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			sets[0].GetStart().GetStart(),
			sets[len(sets)-1].GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendSet(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if where := ctx.GetWhereExpr(); where != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			where.GetStart().GetStart(),
			where.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modDeleteStatement(ctx mysqlparser.IDeleteStatementContext, sb *strings.Builder) {
	if single := ctx.SingleDeleteStatement(); single != nil {
		v.modSingleDeleteStatement(single, sb)
		return
	}

	if multi := ctx.MultipleDeleteStatement(); multi != nil {
		v.modMultipleDeleteStatement(multi, sb)
		return
	}
}

func (v *visitor) modSingleDeleteStatement(ctx mysqlparser.ISingleDeleteStatementContext, sb *strings.Builder) {
	v.modWithClause(ctx, sb)

	if priority := ctx.GetPriority(); priority != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			priority.GetStart(), priority.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if quick := ctx.QUICK(); quick != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			quick.GetSymbol().GetStart(), quick.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if ignore := ctx.IGNORE(); ignore != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			ignore.GetSymbol().GetStart(), ignore.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if table := ctx.TableName(); table != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			table.GetStart().GetStart(), table.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendTable(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if partitions := ctx.GetPartitions(); partitions != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			partitions.GetStart().GetStart(), partitions.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendPartition(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if where := ctx.GetWhereExpr(); where != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			where.GetStart().GetStart(),
			where.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if order := ctx.OrderByClause(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limit := ctx.LimitClauseAtom(); limit != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			limit.GetStart().GetStart(),
			limit.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modMultipleDeleteStatement(ctx mysqlparser.IMultipleDeleteStatementContext, sb *strings.Builder) {
	if using := ctx.USING(); using == nil {
		v.Err = fmt.Errorf("only the USING form is supported in DELETE statements")
		return
	}

	v.modWithClause(ctx, sb)

	if priority := ctx.GetPriority(); priority != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			priority.GetStart(), priority.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if quick := ctx.QUICK(); quick != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			quick.GetSymbol().GetStart(), quick.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if ignore := ctx.IGNORE(); ignore != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			ignore.GetSymbol().GetStart(), ignore.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	for _, table := range ctx.AllMultipleDeleteTable() {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			table.GetStart().GetStart(), table.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendTable(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if where := ctx.GetWhereExpr(); where != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			where.GetStart().GetStart(),
			where.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}
