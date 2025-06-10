package parser

import (
	"fmt"
	"strings"

	"github.com/stephenafamo/bob/internal"
	sqliteparser "github.com/stephenafamo/sqlparser/sqlite"
)

func (v *visitor) modWith_clause(ctx interface {
	With_clause() sqliteparser.IWith_clauseContext
}, sb *strings.Builder,
) {
	with := ctx.With_clause()
	if with == nil {
		return
	}

	if with.RECURSIVE_() != nil {
		sb.WriteString("q.SetRecursive(true)\n")
	}
	for _, cte := range with.AllCommon_table_expression() {
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

func (v *visitor) modSelect_stmt(ctx sqliteparser.ISelect_stmtContext, sb *strings.Builder) [][]string {
	var imports [][]string

	v.modWith_clause(ctx, sb)

	{
		core := ctx.Select_core()

		if distinct := core.DISTINCT_(); distinct != nil {
			sb.WriteString("q.Distinct = true\n")
		}

		allResults := core.AllResult_column()
		if len(allResults) > 0 {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				allResults[0].GetStart().GetStart(),
				allResults[len(allResults)-1].GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendSelect(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}

		if from := core.From_item(); from != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				from.GetStart().GetStart(),
				from.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}

		if where := core.Where_stmt(); where != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				where.WHERE_().GetSymbol().GetStop()+1,
				where.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}

		if groupBy := core.Group_by_stmt(); groupBy != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				groupBy.GetStart().GetStart(),
				groupBy.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendGroup(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}

		if having := core.GetHavingExpr(); having != nil {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				having.GetStart().GetStart(),
				having.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendHaving(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}

		for _, window := range core.AllWindow_stmt() {
			v.StmtRules = append(v.StmtRules, internal.RecordPoints(
				window.GetStart().GetStart(),
				window.GetStop().GetStop(),
				func(start, end int) error {
					fmt.Fprintf(sb, "q.AppendWindow(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...)
		}
	}

	compounds := ctx.AllCompound_select()

	if len(compounds) > 0 {
		imports = append(imports, []string{"github.com/stephenafamo/bob/clause"})
	}

	for _, compound := range compounds {
		strategy := strings.ToUpper(compound.Compound_operator().GetText())
		all := compound.Compound_operator().ALL_() != nil
		if all {
			strategy = strategy[:len(strategy)-3]
		}
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			compound.Select_core().GetStart().GetStart(),
			compound.Select_core().GetStop().GetStop(),
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

	if order := ctx.Order_by_stmt(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY_().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limit := ctx.Limit_stmt(); limit != nil {
		if limit.COMMA() != nil {
			v.Err = fmt.Errorf("LIMIT with comma is not supported")
			return nil
		}

		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			limit.GetFirstExpr().GetStart().GetStart(),
			limit.GetFirstExpr().GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)

		if offset := limit.GetLastExpr(); offset != nil {
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

	return imports
}

func (v *visitor) modInsert_stmt(ctx sqliteparser.IInsert_stmtContext, sb *strings.Builder) {
	v.modWith_clause(ctx, sb)

	if ctx.INSERT_() == nil {
		v.Err = fmt.Errorf("REPLACE INTO is not supported. Use INSERT OR REPLACE INTO instead")
		return
	}

	if or := ctx.GetUpsert_action(); or != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			or.GetStart(), or.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetOr(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	table := ctx.Table_name()
	v.quoteIdentifiable(table)

	tableStart := table.GetStart().GetStart()
	tableStop := table.GetStop().GetStop()

	if schema := ctx.Schema_name(); schema != nil {
		v.quoteIdentifiable(schema)
		tableStart = schema.GetStart().GetStart()
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		tableStart, tableStop,
		func(start, end int) error {
			fmt.Fprintf(sb, "q.TableRef.Expression = EXPR.raw(%d, %d)\n", start, end)
			return nil
		},
	)...)

	if alias := ctx.Table_alias(); alias != nil {
		v.quoteIdentifiable(alias)
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			alias.GetStart().GetStart(),
			alias.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.TableRef.Alias = %q\n", getName(alias))
				return nil
			},
		)...)
	}

	allColumns := ctx.AllColumn_name()
	if len(allColumns) > 0 {
		colNames := make([]string, len(allColumns))
		for i, col := range allColumns {
			colNames[i] = getName(col)
		}
		fmt.Fprintf(sb, "q.TableRef.Columns = %#v\n", colNames)
	}

	if values := ctx.Values_clause(); values != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			values.GetStart().GetStart(),
			values.GetStop().GetStop(),
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

	if selectStmt := ctx.Select_stmt(); selectStmt != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			selectStmt.GetStart().GetStart(),
			selectStmt.GetStop().GetStop(),
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

	if upsert := ctx.Upsert_clause(); upsert != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			upsert.GetStart().GetStart(),
			upsert.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetConflict(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if returning := ctx.Returning_clause(); returning != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			returning.RETURNING_().GetSymbol().GetStop()+1,
			returning.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendReturning(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}

func (v *visitor) modUpdate_stmt(ctx sqliteparser.IUpdate_stmtContext, sb *strings.Builder) {
	v.modWith_clause(ctx, sb)

	if or := ctx.GetUpsert_action(); or != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			or.GetStart(), or.GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetOr(EXPR.raw(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	qualifiedTable := ctx.Qualified_table_name()
	table := qualifiedTable.Table_name()
	v.quoteIdentifiable(table)

	tableStart := table.GetStart().GetStart()
	tableStop := table.GetStop().GetStop()

	if schema := qualifiedTable.Schema_name(); schema != nil {
		v.quoteIdentifiable(schema)
		tableStart = schema.GetStart().GetStart()
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		tableStart, tableStop,
		func(start, end int) error {
			fmt.Fprintf(sb, "q.Table.Expression = EXPR.raw(%d, %d)\n", start, end)
			return nil
		},
	)...)

	if alias := qualifiedTable.Table_alias(); alias != nil {
		v.quoteIdentifiable(alias)
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			alias.GetStart().GetStart(),
			alias.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.Table.Alias = %q\n", getName(alias))
				return nil
			},
		)...)
	}

	indexName := qualifiedTable.Index_name()
	switch {
	case indexName != nil: // INDEXED BY
		fmt.Fprintf(
			sb,
			"index := %q; q.Table.IndexedBy = &index\n",
			getName(indexName),
		)
	case ctx.Qualified_table_name().NOT_() != nil: // NOT INDEXED
		sb.WriteString("index := \"\"; q.Table.IndexedBy = &index\n")
	}

	cols := ctx.AllColumn_name_or_list()
	exprs := ctx.AllExpr()

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		cols[0].GetStart().GetStart(),
		exprs[len(exprs)-1].GetStop().GetStop(),
		func(start, end int) error {
			fmt.Fprintf(sb, "q.AppendSet(EXPR.subExpr(%d, %d))\n", start, end)
			return nil
		},
	)...)

	if from := ctx.From_item(); from != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			from.GetStart().GetStart(),
			from.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if where := ctx.Where_stmt(); where != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			where.WHERE_().GetSymbol().GetStop()+1,
			where.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if returning := ctx.Returning_clause(); returning != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			returning.RETURNING_().GetSymbol().GetStop()+1,
			returning.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendReturning(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if order := ctx.Order_by_stmt(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY_().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limit := ctx.Limit_stmt(); limit != nil {
		if limit.COMMA() != nil {
			v.Err = fmt.Errorf("LIMIT with comma is not supported")
		}

		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			limit.GetFirstExpr().GetStart().GetStart(),
			limit.GetFirstExpr().GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)

		if offset := limit.GetLastExpr(); offset != nil {
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
}

func (v *visitor) modDelete_stmt(ctx sqliteparser.IDelete_stmtContext, sb *strings.Builder) {
	v.modWith_clause(ctx, sb)

	qualifiedTable := ctx.Qualified_table_name()
	table := qualifiedTable.Table_name()
	v.quoteIdentifiable(table)

	tableStart := table.GetStart().GetStart()
	tableStop := table.GetStop().GetStop()

	if schema := qualifiedTable.Schema_name(); schema != nil {
		v.quoteIdentifiable(schema)
		tableStart = schema.GetStart().GetStart()
	}

	v.StmtRules = append(v.StmtRules, internal.RecordPoints(
		tableStart, tableStop,
		func(start, end int) error {
			fmt.Fprintf(sb, "q.TableRef.Expression = EXPR.raw(%d, %d)\n", start, end)
			return nil
		},
	)...)

	if alias := qualifiedTable.Table_alias(); alias != nil {
		v.quoteIdentifiable(alias)
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			alias.GetStart().GetStart(),
			alias.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.TableRef.Alias = %q\n", getName(alias))
				return nil
			},
		)...)
	}

	indexName := qualifiedTable.Index_name()
	switch {
	case indexName != nil: // INDEXED BY
		fmt.Fprintf(
			sb,
			"index := %q; q.TableRef.IndexedBy = &index\n",
			getName(indexName),
		)
	case ctx.Qualified_table_name().NOT_() != nil: // NOT INDEXED
		sb.WriteString("index := \"\"; q.TableRef.IndexedBy = &index\n")
	}

	if where := ctx.Where_stmt(); where != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			where.WHERE_().GetSymbol().GetStop()+1,
			where.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if returning := ctx.Returning_clause(); returning != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			returning.RETURNING_().GetSymbol().GetStop()+1,
			returning.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendReturning(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if order := ctx.Order_by_stmt(); order != nil {
		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			order.BY_().GetSymbol().GetStop()+1,
			order.GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limit := ctx.Limit_stmt(); limit != nil {
		if limit.COMMA() != nil {
			v.Err = fmt.Errorf("LIMIT with comma is not supported")
		}

		v.StmtRules = append(v.StmtRules, internal.RecordPoints(
			limit.GetFirstExpr().GetStart().GetStart(),
			limit.GetFirstExpr().GetStop().GetStop(),
			func(start, end int) error {
				fmt.Fprintf(sb, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)

		if offset := limit.GetLastExpr(); offset != nil {
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
}
