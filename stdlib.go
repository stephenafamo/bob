package bob

import (
	"database/sql"
	"database/sql/driver"

	"github.com/stephenafamo/bob/scanto"
)

func New[T scanto.StdInterface](wrapped T) scanto.Common[T] {
	return scanto.New(wrapped)
}

func Open(driverName string, dataSource string) (scanto.DB, error) {
	return scanto.Open(driverName, dataSource)
}

func OpenDB(c driver.Connector) scanto.DB {
	return scanto.OpenDB(c)
}

func NewDB(db *sql.DB) scanto.DB {
	return scanto.NewDB(db)
}

func NewTx(tx *sql.Tx) scanto.Tx {
	return scanto.NewTx(tx)
}

func NewConn(conn *sql.Conn) scanto.Conn {
	return scanto.NewConn(conn)
}
