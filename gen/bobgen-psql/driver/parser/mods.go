package parser

import (
	"fmt"
	"strconv"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/internal"
)

type combine struct {
	Operation  pg.SetOperation
	All        bool
	SelectStmt *pg.SelectStmt
	Info       nodeInfo
}

func (w *walker) modWithClause(with *pg.WithClause, info nodeInfo) {
	cteInfos := info.children["Ctes"]
	if with.Recursive {
		w.mods.WriteString("q.SetRecursive(true)\n")
	}
	if len(with.Ctes) > 0 {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(cteInfos.start),
				int(cteInfos.end-1),
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendCTE(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}
}

//nolint:gocyclo,maintidx
func (w *walker) modSelectStatement(stmt *pg.Node_SelectStmt, info nodeInfo) {
	if withInfo, ok := info.children["WithClause"]; ok {
		w.modWithClause(stmt.SelectStmt.WithClause, withInfo)
	}

	main := stmt.SelectStmt
	mainInfo := info

	var combines []combine

	for main.Larg != nil {
		combines = append(combines, combine{
			Operation:  main.Op,
			All:        main.All,
			SelectStmt: main.Rarg,
			Info:       mainInfo.children["Rarg"],
		})
		main = main.Larg
		mainInfo = mainInfo.children["Larg"]
	}

	if distinct := main.DistinctClause; distinct != nil {
		if len(distinct) == 1 && distinct[0].Node == nil {
			fmt.Fprintln(w.mods, "q.Distinct.On = []any{}")
		} else {
			distinctInfo := info.children["DistinctClause"]
			w.editRules = append(w.editRules,
				internal.RecordPoints(
					int(distinctInfo.start),
					int(distinctInfo.end-1),
					func(start, end int) error {
						fmt.Fprintf(
							w.mods,
							"q.Distinct.On = append(q.Distinct.On,EXPR.subExpr(%d, %d))\n",
							start, end,
						)
						return nil
					},
				)...,
			)
		}
	}

	if targetInfo, ok := mainInfo.children["TargetList"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(targetInfo.start),
				int(targetInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendSelect(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if fromInfo, ok := mainInfo.children["FromClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(fromInfo.start),
				int(fromInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if whereInfo, ok := mainInfo.children["WhereClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(whereInfo.start),
				int(whereInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if groupByInfo, ok := mainInfo.children["GroupClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(groupByInfo.start),
				int(groupByInfo.end)-1,
				func(start, end int) error {
					if main.GroupDistinct {
						fmt.Fprint(w.mods, "q.SetGroupByDistinct(true)\n")
					}
					fmt.Fprintf(w.mods, "q.AppendGroup(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if havingInfo, ok := mainInfo.children["HavingClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(havingInfo.start),
				int(havingInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendHaving(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if windowsInfo, ok := mainInfo.children["WindowClause"]; ok {
		for _, windowInfo := range windowsInfo.children {
			nameStart := w.getStartOfTokenBefore(windowInfo.start, pg.Token_IDENT)
			w.editRules = append(w.editRules,
				internal.RecordPoints(
					int(nameStart),
					int(windowInfo.end)-1,
					func(start, end int) error {
						fmt.Fprintf(w.mods, "q.AppendWindow(EXPR.subExpr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}
	}

	if len(combines) > 0 {
		w.imports = append(w.imports, []string{"github.com/stephenafamo/bob/clause"})
	}
	for i := len(combines) - 1; i >= 0; i-- {
		combine := combines[i]

		strategy := ""
		switch combine.Operation {
		case pg.SetOperation_SETOP_UNION:
			strategy = "UNION"
		case pg.SetOperation_SETOP_INTERSECT:
			strategy = "INTERSECT"
		case pg.SetOperation_SETOP_EXCEPT:
			strategy = "EXCEPT"
		}

		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(combine.Info.start),
				int(combine.Info.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, `
                        q.AppendCombine(clause.Combine{
                            Strategy: "%s",
                            All: %t,
                            Query: bob.BaseQuery[bob.Expression]{
                                Expression: EXPR.subExpr(%d, %d),
                                QueryType: bob.QueryTypeSelect,
                                Dialect: dialect.Dialect,
                            },
                        })
                    `, strategy, combine.All, start, end)
					return nil
				},
			)...,
		)
	}

	if limitInfo, ok := info.children["LimitCount"]; ok {
		w.imports = append(w.imports, []string{"github.com/stephenafamo/bob/dialect/psql"})
		rawLimit := w.input[limitInfo.start:limitInfo.end]
		switch stmt.SelectStmt.LimitOption {
		case pg.LimitOption_LIMIT_OPTION_COUNT:
			fmt.Fprintf(w.mods, "q.SetLimit(psql.Raw(%q))\n", rawLimit)
		case pg.LimitOption_LIMIT_OPTION_WITH_TIES:
			w.imports = append(w.imports, []string{"github.com/stephenafamo/bob/clause"})
			fmt.Fprintf(w.mods, `q.SetFetch(clause.Fetch{
					Count: psql.Raw(%q),
					WithTies: true,
				})
			`, rawLimit)
		}
	}

	if offsetInfo, ok := info.children["LimitOffset"]; ok {
		w.imports = append(w.imports, []string{"github.com/stephenafamo/bob/dialect/psql"})
		rawOffset := w.input[offsetInfo.start:offsetInfo.end]
		fmt.Fprintf(w.mods, "q.SetOffset(psql.Raw(%q))\n", rawOffset)
	}

	if orderInfo, ok := info.children["SortClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(orderInfo.start),
				int(orderInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	for i, lockClause := range stmt.SelectStmt.LockingClause {
		lock, ok := lockClause.Node.(*pg.Node_LockingClause)
		if !ok || lock.LockingClause.Strength < 2 {
			continue
		}

		var bobLock clause.Lock

		switch lock.LockingClause.Strength {
		case pg.LockClauseStrength_LCS_FORKEYSHARE:
			bobLock.Strength = "KEY SHARE"
		case pg.LockClauseStrength_LCS_FORSHARE:
			bobLock.Strength = "SHARE"
		case pg.LockClauseStrength_LCS_FORNOKEYUPDATE:
			bobLock.Strength = "NO KEY UPDATE"
		case pg.LockClauseStrength_LCS_FORUPDATE:
			bobLock.Strength = "UPDATE"
		}

		switch lock.LockingClause.WaitPolicy {
		case pg.LockWaitPolicy_LockWaitSkip:
			bobLock.Wait = "SKIP LOCKED"
		case pg.LockWaitPolicy_LockWaitError:
			bobLock.Wait = "NOWAIT"
		}

		if len(lock.LockingClause.LockedRels) > 0 {
			lockInfo := info.children["LockingClause"].children[strconv.Itoa(i)]
			bobLock.Tables = []string{w.input[lockInfo.start:lockInfo.end]}
		}

		w.imports = append(w.imports, []string{"github.com/stephenafamo/bob/clause"})
		fmt.Fprintf(w.mods, "q.AppendLock(%#v)\n", bobLock)
	}
}

func (w *walker) modInsertStatement(stmt *pg.Node_InsertStmt, info nodeInfo) {
	if withInfo, ok := info.children["WithClause"]; ok {
		w.modWithClause(stmt.InsertStmt.WithClause, withInfo)
	}

	if intoInfo, ok := info.children["Relation"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(intoInfo.start),
				int(intoInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.TableRef.Expression = EXPR.subExpr(%d, %d)\n", start, end)
					return nil
				},
			)...,
		)
	}

	colNames := make([]string, len(stmt.InsertStmt.Cols))
	for i := range stmt.InsertStmt.Cols {
		colNameInfo := info.children["Cols"].children[strconv.Itoa(i)]
		colNames[i] = w.names[colNameInfo.position()]
	}
	if len(colNames) > 0 {
		fmt.Fprintf(w.mods, "q.TableRef.Columns = %#v\n", colNames)
	}

	if selectInfo, ok := info.children["SelectStmt"].children["SelectStmt"]; ok {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(selectInfo.start), int(selectInfo.end)-1,
			func(start, end int) error {
				fmt.Fprintf(w.mods, `q.Query = bob.BaseQuery[bob.Expression]{
						Expression: EXPR.subExpr(%d, %d),
						Dialect: dialect.Dialect,
						QueryType: bob.QueryTypeSelect,
						}
					`, start, end)
				return nil
			},
		)...)
	}

	switch stmt.InsertStmt.Override {
	case pg.OverridingKind_OVERRIDING_USER_VALUE:
		fmt.Fprintln(w.mods, `q.Overriding = "USER"`)
	case pg.OverridingKind_OVERRIDING_SYSTEM_VALUE:
		fmt.Fprintln(w.mods, `q.Overriding = "SYSTEM"`)
	}

	if conflictInfo, ok := info.children["OnConflictClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(conflictInfo.start),
				int(conflictInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.SetConflict(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if returnInfo, ok := info.children["ReturningList"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(returnInfo.start),
				int(returnInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendReturning(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}
}

func (w *walker) modUpdateStatement(stmt *pg.Node_UpdateStmt, info nodeInfo) {
	if withInfo, ok := info.children["WithClause"]; ok {
		w.modWithClause(stmt.UpdateStmt.WithClause, withInfo)
	}

	if tableInfo, ok := info.children["Relation"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(tableInfo.start),
				int(tableInfo.end)-1,
				func(start, end int) error {
					if !stmt.UpdateStmt.GetRelation().GetInh() {
						fmt.Fprintln(w.mods, "q.Only = true")
					}
					fmt.Fprintf(w.mods, "q.Table.Expression = EXPR.subExpr(%d, %d)\n", start, end)
					return nil
				},
			)...,
		)
	}

	if targetInfo, ok := info.children["TargetList"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(targetInfo.start),
				int(targetInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendSet(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if fromInfo, ok := info.children["FromClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(fromInfo.start),
				int(fromInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if whereInfo, ok := info.children["WhereClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(whereInfo.start),
				int(whereInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if returnInfo, ok := info.children["ReturningList"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(returnInfo.start),
				int(returnInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendReturning(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}
}

func (w *walker) modDeleteStatement(stmt *pg.Node_DeleteStmt, info nodeInfo) {
	if withInfo, ok := info.children["WithClause"]; ok {
		w.modWithClause(stmt.DeleteStmt.WithClause, withInfo)
	}

	if tableInfo, ok := info.children["Relation"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(tableInfo.start),
				int(tableInfo.end)-1,
				func(start, end int) error {
					if !stmt.DeleteStmt.GetRelation().GetInh() {
						fmt.Fprintln(w.mods, "q.Only = true")
					}
					fmt.Fprintf(w.mods, "q.Table.Expression = EXPR.subExpr(%d, %d)\n", start, end)
					return nil
				},
			)...,
		)
	}

	if usingInfo, ok := info.children["UsingClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(usingInfo.start),
				int(usingInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.SetTable(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if whereInfo, ok := info.children["WhereClause"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(whereInfo.start),
				int(whereInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendWhere(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}

	if returnInfo, ok := info.children["ReturningList"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(returnInfo.start),
				int(returnInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendReturning(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}
}
