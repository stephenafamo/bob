package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
)

func verifySelectStatement(stmt *pg.SelectStmt, _ nodeInfo) error {
	if stmt == nil {
		return fmt.Errorf("nil statement")
	}

	if len(stmt.FromClause) > 1 {
		return fmt.Errorf("multiple FROM tables are not supported, convert to a CROSS JOIN")
	}

	return nil
}

func verifyUpdateStatement(stmt *pg.UpdateStmt, _ nodeInfo) error {
	if stmt == nil {
		return fmt.Errorf("nil statement")
	}

	if len(stmt.FromClause) > 1 {
		return fmt.Errorf("multiple FROM tables are not supported, convert to a CROSS JOIN")
	}

	return nil
}

func verifyDeleteStatement(stmt *pg.DeleteStmt, _ nodeInfo) error {
	if stmt == nil {
		return fmt.Errorf("nil statement")
	}

	if len(stmt.UsingClause) > 1 {
		return fmt.Errorf("multiple USING tables are not supported, convert to a CROSS JOIN")
	}

	return nil
}
