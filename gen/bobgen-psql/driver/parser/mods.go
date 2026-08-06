package parser

import (
	"fmt"
	"strconv"

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
	for i := range with.Ctes {
		cteInfo, ok := cteInfos.children[strconv.Itoa(i)]
		if !ok {
			continue
		}

		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(cteInfo.start),
				int(cteInfo.end-1),
				func(start, end int) error {
					fmt.Fprintf(w.mods, "q.AppendCTE(EXPR.subExpr(%d, %d))\n", start, end)
					return nil
				},
			)...,
		)
	}
}

func (w *walker) modAppendTableRefItems(clauseInfo nodeInfo, items []*pg.Node) {
	for i, item := range items {
		if item == nil {
			continue
		}

		itemInfo, ok := clauseInfo.children[strconv.Itoa(i)]
		if !ok {
			continue
		}

		w.editRules = append(w.editRules,
			internal.RecordPoints(
				int(itemInfo.start),
				int(itemInfo.end)-1,
				func(start, end int) error {
					fmt.Fprintf(
						w.mods,
						"q.AppendTableRef(clause.TableRef{Expression: EXPR.subExpr(%d, %d)})\n",
						start, end,
					)
					return nil
				},
			)...,
		)
	}
}
