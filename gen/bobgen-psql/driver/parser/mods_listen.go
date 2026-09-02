package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
)

func (w *walker) modListenStatement(stmt *pg.Node_ListenStmt, _ nodeInfo) {
	fmt.Fprintf(w.mods, "q.Channel = %q\n", stmt.ListenStmt.Conditionname)
}
