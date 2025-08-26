package pgx

import (
	"database/sql"

	"github.com/jackc/pgx/v5"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

var (
	_ sql.Result = result{}
	_ scan.Rows  = rows{}
)

var (
	_ bob.Executor       = Pool{}
	_ bob.Transactor[Tx] = Pool{}
)

var (
	_ bob.Executor       = Conn{}
	_ bob.Transactor[Tx] = Conn{}
)

var (
	_ bob.Executor    = Tx{}
	_ bob.Transaction = Tx{}
)

// Ensure that our Tx satisfies pgx.Tx interface
// we wrap the given pgx.Tx, to add auto rollback based on context cancellation
// so we do not simply embed it
var _ pgx.Tx = Tx{}
