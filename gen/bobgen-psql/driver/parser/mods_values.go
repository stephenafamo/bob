package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
	"github.com/stephenafamo/bob/internal"
)

func (w *walker) modValuesStatement(stmt *pg.Node_SelectStmt, info nodeInfo) {
	if orderInfo, ok := info.children["SortClause"]; ok {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(orderInfo.start),
			int(orderInfo.end)-1,
			func(start, end int) error {
				fmt.Fprintf(w.mods, "q.AppendOrder(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}

	if limitInfo, ok := info.children["LimitCount"]; ok {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(limitInfo.start),
			int(limitInfo.end)-1,
			func(start, end int) error {
				switch stmt.SelectStmt.LimitOption {
				case pg.LimitOption_LIMIT_OPTION_COUNT:
					fmt.Fprintf(w.mods, "q.SetLimit(EXPR.subExpr(%d, %d))\n", start, end)
				case pg.LimitOption_LIMIT_OPTION_WITH_TIES:
					w.imports = append(w.imports, []string{"github.com/stephenafamo/bob/clause"})
					fmt.Fprintf(w.mods, "q.SetFetch(clause.Fetch{Count: EXPR.subExpr(%d, %d), WithTies: true})\n", start, end)
				}
				return nil
			},
		)...)
	}

	if offsetInfo, ok := info.children["LimitOffset"]; ok {
		w.editRules = append(w.editRules, internal.RecordPoints(
			int(offsetInfo.start),
			int(offsetInfo.end)-1,
			func(start, end int) error {
				fmt.Fprintf(w.mods, "q.SetOffset(EXPR.subExpr(%d, %d))\n", start, end)
				return nil
			},
		)...)
	}
}
