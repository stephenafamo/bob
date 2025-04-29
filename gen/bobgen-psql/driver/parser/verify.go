package parser

import (
	"fmt"

	pg "github.com/pganalyze/pg_query_go/v6"
)

func verifySelectStatement(stmt *pg.SelectStmt, info nodeInfo) error {
	if stmt == nil {
		return fmt.Errorf("nil statement")
	}

	if len(stmt.FromClause) > 0 {
		return fmt.Errorf("multiple FROM tables are not supported, convert to a CROSS JOIN")
	}

	if isFetchWithTiesAfterOffset(stmt, info) {
		return fmt.Errorf("FETCH WITH TIES should be placed BEFORE OFFSET")
	}

	return nil
}

func verifyUpdateStatement(stmt *pg.UpdateStmt, _ nodeInfo) error {
	if stmt == nil {
		return fmt.Errorf("nil statement")
	}

	if len(stmt.FromClause) > 0 {
		return fmt.Errorf("multiple FROM tables are not supported, convert to a CROSS JOIN")
	}

	return nil
}

func verifyDeleteStatement(stmt *pg.DeleteStmt, _ nodeInfo) error {
	if stmt == nil {
		return fmt.Errorf("nil statement")
	}

	if len(stmt.UsingClause) > 0 {
		return fmt.Errorf("multiple USING tables are not supported, convert to a CROSS JOIN")
	}

	return nil
}

func isFetchWithTiesAfterOffset(stmt *pg.SelectStmt, info nodeInfo) bool {
	if stmt.LimitOffset == nil {
		return false
	}

	if stmt.LimitOption != pg.LimitOption_LIMIT_OPTION_WITH_TIES {
		return false
	}

	fetchWithTiesPosition := info.children["LimitCount"].start
	offsetPosition := info.children["LimitOffset"].start

	return fetchWithTiesPosition > offsetPosition
}
