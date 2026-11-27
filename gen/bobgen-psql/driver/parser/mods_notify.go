package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
)

func (w *walker) modNotifyStatement(stmt *pg.Node_NotifyStmt, _ nodeInfo) {
	fmt.Fprintf(w.mods, "q.Channel = %q\n", stmt.NotifyStmt.Conditionname)
	if stmt.NotifyStmt.Payload != "" {
		fmt.Fprintf(w.mods, "q.Payload = %q\n", stmt.NotifyStmt.Payload)
	}
}
