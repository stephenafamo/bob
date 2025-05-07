package bob

import "database/sql"

// To make sure they satisfy the interface
var (
	_ Executor = common[*sql.DB]{}
	_ Executor = common[*sql.Tx]{}
	_ Executor = common[*sql.Conn]{}
)

var (
	_ Executor   = DB{}
	_ Transactor = DB{}
)

var (
	_ Executor               = Tx{}
	_ txForStmt[StdPrepared] = Tx{}
	_ Preparer[StdPrepared]  = Tx{}
	_ Transaction            = Tx{}
)

var (
	_ Executor   = Conn{}
	_ Transactor = Conn{}
)
