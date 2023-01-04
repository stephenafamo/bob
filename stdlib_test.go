package bob

import "database/sql"

// To make sure they satisfy the interface
var (
	_ Executor = common[*sql.DB]{}
	_ Executor = common[*sql.Tx]{}
	_ Executor = common[*sql.Conn]{}
)
