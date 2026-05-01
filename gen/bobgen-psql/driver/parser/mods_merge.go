package parser

import (
	"fmt"
	"strconv"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/internal"
)

func (w *walker) modMergeStatement(stmt *pg.Node_MergeStmt, info nodeInfo) {
	if withInfo, ok := info.children["WithClause"]; ok {
		w.modWithClause(stmt.MergeStmt.WithClause, withInfo)
	}

	if tableInfo, ok := info.children["Relation"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(tableInfo.start),
				int(tableInfo.end)-1,
				func(start, end int) error {
					if !stmt.MergeStmt.GetRelation().GetInh() {
						fmt.Fprintln(w.mods, "q.Only = true")
					}
					fmt.Fprintf(w.mods, "q.Table.Expression = EXPR.subExpr(%d, %d)\n", start, end)
					return nil
				},
			)...,
		)
	}

	if sourceInfo, ok := info.children["SourceRelation"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(sourceInfo.start),
				int(sourceInfo.end)-1,
				func(start, end int) error {
					if src := stmt.MergeStmt.GetSourceRelation(); src != nil {
						if rangeVar, ok := src.Node.(*pg.Node_RangeVar); ok && rangeVar.RangeVar != nil && !rangeVar.RangeVar.GetInh() {
							fmt.Fprintln(w.mods, "q.Using.Only = true")
						}
					}
					fmt.Fprintf(w.mods, "q.Using.Source = EXPR.subExpr(%d, %d)\n", start, end)
					return nil
				},
			)...,
		)
	}

	if onInfo, ok := info.children["JoinCondition"]; ok {
		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(onInfo.start),
				int(onInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.Using.Condition = EXPR.subExpr(%d, %d)\n", start, end)
					return nil
				},
			)...,
		)
	}

	whenInfos := info.children["MergeWhenClauses"]
	whenCount := 0
	for i, whenNode := range stmt.MergeStmt.MergeWhenClauses {
		when := whenNode.GetMergeWhenClause()
		if when == nil {
			continue
		}

		whenType, ok := mergeMatchKindToDialectWhenType(when.MatchKind)
		if !ok {
			w.errors = append(w.errors, fmt.Errorf("unsupported MERGE match kind: %s", when.MatchKind.String()))
			continue
		}

		actionType, ok := mergeCommandTypeToDialectActionType(when.CommandType)
		if !ok {
			w.errors = append(w.errors, fmt.Errorf("unsupported MERGE command type: %s", when.CommandType.String()))
			continue
		}

		fmt.Fprintf(w.mods,
			"q.When = append(q.When, dialect.MergeWhen{Type: %s, Action: dialect.MergeAction{Type: %s}})\n",
			whenType,
			actionType,
		)
		currentWhenIndex := whenCount
		whenCount++

		if overriding, ok := mergeOverridingToDialect(when.Override); ok {
			fmt.Fprintf(w.mods, "q.When[%d].Action.Overriding = %s\n", currentWhenIndex, overriding)
		}

		if when.CommandType == pg.CmdType_CMD_INSERT {
			if cols := mergeInsertColumns(when); len(cols) > 0 {
				fmt.Fprintf(w.mods, "q.When[%d].Action.Columns = %#v\n", currentWhenIndex, cols)
			}
		}

		whenInfo, hasWhenInfo := whenInfos.children[strconv.Itoa(i)]
		if wrappedInfo, ok := whenInfo.children["MergeWhenClause"]; ok {
			whenInfo = wrappedInfo
		}

		if hasWhenInfo {
			if conditionInfo, ok := whenInfo.children["Condition"]; ok {
				w.editRules = append(w.editRules,
					internal.RecordPoints(
						int(conditionInfo.start),
						int(conditionInfo.end)-1,
						func(start, end int) error {
							fmt.Fprintf(w.mods, "q.When[%d].Condition = EXPR.subExpr(%d, %d)\n", currentWhenIndex, start, end)
							return nil
						},
					)...,
				)
			}

			switch when.CommandType {
			case pg.CmdType_CMD_INSERT:
				if valuesInfo, ok := whenInfo.children["Values"]; ok {
					w.editRules = append(w.editRules,
						internal.RecordPoints(
							int(valuesInfo.start),
							int(valuesInfo.end)-1,
							func(start, end int) error {
								fmt.Fprintf(w.mods, "q.When[%d].Action.Values = append(q.When[%d].Action.Values, EXPR.subExpr(%d, %d))\n", currentWhenIndex, currentWhenIndex, start, end)
								return nil
							},
						)...,
					)
				}

			case pg.CmdType_CMD_UPDATE:
				if setInfo, ok := whenInfo.children["TargetList"]; ok {
					w.editRules = append(w.editRules,
						internal.RecordPoints(
							int(setInfo.start),
							int(setInfo.end)-1,
							func(start, end int) error {
								fmt.Fprintf(w.mods, "q.When[%d].Action.Set = append(q.When[%d].Action.Set, EXPR.subExpr(%d, %d))\n", currentWhenIndex, currentWhenIndex, start, end)
								return nil
							},
						)...,
					)
				}
			}
		}
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

func mergeMatchKindToDialectWhenType(kind pg.MergeMatchKind) (string, bool) {
	switch kind {
	case pg.MergeMatchKind_MERGE_WHEN_MATCHED:
		return "dialect.MergeWhenMatched", true
	case pg.MergeMatchKind_MERGE_WHEN_NOT_MATCHED_BY_SOURCE:
		return "dialect.MergeWhenNotMatchedBySource", true
	case pg.MergeMatchKind_MERGE_WHEN_NOT_MATCHED_BY_TARGET:
		// pg_query parses both "NOT MATCHED" and "NOT MATCHED BY TARGET" as
		// MERGE_WHEN_NOT_MATCHED_BY_TARGET. We use MergeWhenNotMatched which
		// writes "NOT MATCHED" for backward compatibility with PG 15-16.
		return "dialect.MergeWhenNotMatched", true
	default:
		return "", false
	}
}

func mergeCommandTypeToDialectActionType(cmd pg.CmdType) (string, bool) {
	switch cmd {
	case pg.CmdType_CMD_NOTHING:
		return "dialect.MergeActionDoNothing", true
	case pg.CmdType_CMD_DELETE:
		return "dialect.MergeActionDelete", true
	case pg.CmdType_CMD_INSERT:
		return "dialect.MergeActionInsert", true
	case pg.CmdType_CMD_UPDATE:
		return "dialect.MergeActionUpdate", true
	default:
		return "", false
	}
}

func mergeOverridingToDialect(override pg.OverridingKind) (string, bool) {
	switch override {
	case pg.OverridingKind_OVERRIDING_USER_VALUE:
		return "dialect.OverridingUser", true
	case pg.OverridingKind_OVERRIDING_SYSTEM_VALUE:
		return "dialect.OverridingSystem", true
	default:
		return "", false
	}
}

func mergeInsertColumns(when *pg.MergeWhenClause) []string {
	cols := make([]string, 0, len(when.TargetList))
	for _, target := range when.TargetList {
		resTarget := target.GetResTarget()
		if resTarget == nil || resTarget.Name == "" {
			continue
		}

		cols = append(cols, resTarget.Name)
	}

	return cols
}
