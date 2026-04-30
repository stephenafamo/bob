package parser

import (
	"fmt"
	"strconv"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/internal"
)

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
