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

//nolint:gocyclo,maintidx
func (w *walker) modSelectStatement(stmt *pg.Node_SelectStmt, info nodeInfo) {
	if withInfo, ok := info.children["WithClause"]; ok {
		cteInfos := withInfo.children["Ctes"]
		if stmt.SelectStmt.WithClause.Recursive {
			w.mods.WriteString("q.SetRecursive(true)\n")
		}
		if len(stmt.SelectStmt.WithClause.Ctes) > 0 {
			w.editRules = append(w.editRules,
				internal.RecordPoints(
					int(cteInfos.start),
					int(cteInfos.end-1),
					func(start, end int) error {
						fmt.Fprintf(w.mods, "q.AppendCTE(o.expr(%d, %d))\n", start, end)
						return nil
					},
				)...,
			)
		}
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
							"q.Distinct.On = append(q.Distinct.On,o.expr(%d, %d))\n",
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
					fmt.Fprintf(w.mods, "q.AppendSelect(o.expr(%d, %d))\n", start, end)
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
					fmt.Fprintf(w.mods, "q.SetTable(o.expr(%d, %d))\n", start, end)
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
					fmt.Fprintf(w.mods, "q.AppendWhere(o.expr(%d, %d))\n", start, end)
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
					fmt.Fprintf(w.mods, "q.AppendGroup(o.expr(%d, %d))\n", start, end)
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
					fmt.Fprintf(w.mods, "q.AppendHaving(o.expr(%d, %d))\n", start, end)
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
						fmt.Fprintf(w.mods, "q.AppendWindow(o.expr(%d, %d))\n", start, end)
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
                                Expression: o.expr(%d, %d),
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
					fmt.Fprintf(w.mods, "q.AppendOrder(o.expr(%d, %d))\n", start, end)
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
