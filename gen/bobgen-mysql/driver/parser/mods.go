package parser

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/internal"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

func (v *visitor) modSelectStatement(ctx mysqlparser.ISelectStatementContext, sb *strings.Builder) [][]string {
	return nil
}

func (v *visitor) modInsertStatement(ctx mysqlparser.IInsertStatementContext, sb *strings.Builder) {
	if priority := ctx.GetPriority(); priority != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			priority.GetStart(), priority.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(o.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if ignore := ctx.IGNORE(); ignore != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			ignore.GetSymbol().GetStart(), ignore.GetSymbol().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendModifier(o.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if partitions := ctx.GetPartitions(); partitions != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			partitions.GetStart().GetStart(), partitions.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendPartition(o.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if table := ctx.TableName(); table != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			table.GetStart().GetStart(), table.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.TableRef.Expression = o.raw(%d, %d)\n", start, end)
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
		if selectStmt := stmtOrVals.SelectStatement(); selectStmt != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				selectStmt.GetStart().GetStart(),
				selectStmt.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, `q.Query = bob.BaseQuery[bob.Expression]{
						Expression: o.expr(%d, %d),
						Dialect: dialect.Dialect,
						QueryType: bob.QueryTypeSelect,
						}
					`, start, end)
					return nil
				},
			)...)
		}

		if values := stmtOrVals.AllExpressionsWithDefaults(); len(values) > 0 {
			for _, value := range values {
				v.StmtRules = append(v.StmtRules, internal.RecordPoints(
					value.LR_BRACKET().GetSymbol().GetStart()+1,
					value.RR_BRACKET().GetSymbol().GetStop()-1,
					func(start, end int) error {
						fmt.Fprintf(sb, "q.AppendValues(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...)
			}
		}
	}

	if sets := ctx.GetSetElement(); sets != nil {
		for _, set := range sets {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				set.ExpressionOrDefault().GetStart().GetStart(),
				set.ExpressionOrDefault().GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb,
						"q.Sets = append(q.Sets, dialect.Set{Col: %q, Val: o.expr(%d, %d)})\n",
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
				fmt.Fprintf(sb, "q.RowAlias = o.raw(%d, %d)\n", start, end)
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
				fmt.Fprintf(sb, "q.DuplicateKeyUpdate.AppendSet(o.expr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modUpdateStatement(ctx mysqlparser.IUpdateStatementContext, sb *strings.Builder) {
}

func (v *visitor) modDeleteStatement(ctx mysqlparser.IDeleteStatementContext, sb *strings.Builder) {
}
