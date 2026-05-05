package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/internal"
)

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
