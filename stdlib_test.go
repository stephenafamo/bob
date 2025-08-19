package bob

var (
	_ Preparer[StdPrepared] = DB{}
	_ Executor              = DB{}
	_ Transactor[Tx]        = DB{}
)

var (
	_ Preparer[StdPrepared] = Conn{}
	_ Executor              = Conn{}
	_ Transactor[Tx]        = Conn{}
)

var (
	_ Preparer[StdPrepared]  = Tx{}
	_ Executor               = Tx{}
	_ Transaction            = Tx{}
	_ txForStmt[StdPrepared] = Tx{}
)
