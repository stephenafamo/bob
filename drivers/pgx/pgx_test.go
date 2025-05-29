package pgx

import (
	"database/sql"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

var (
	_ sql.Result = result{}
	_ scan.Rows  = rows{}
)

var (
	_ bob.Executor   = Pool{}
	_ bob.Transactor = Pool{}
)

var (
	_ bob.Executor   = Conn{}
	_ bob.Transactor = Conn{}
)

var (
	_ bob.Executor    = Tx{}
	_ bob.Transaction = Tx{}
)
