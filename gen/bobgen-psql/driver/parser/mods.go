package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
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
