package bob

var (
	_ Preparer[StdPrepared] = DB{}
	_ Executor              = DB{}
	_ Transactor            = DB{}
)

var (
	_ Preparer[StdPrepared] = Conn{}
	_ Executor              = Conn{}
	_ Transactor            = Conn{}
)

var (
	_ Preparer[StdPrepared]  = Tx{}
	_ Executor               = Tx{}
	_ Transaction            = Tx{}
	_ txForStmt[StdPrepared] = Tx{}
)
