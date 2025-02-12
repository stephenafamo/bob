package mysql

import (
	"github.com/twitter-payments/bob"
	"github.com/twitter-payments/bob/dialect/mysql/dialect"
	"github.com/twitter-payments/bob/expr"
)

func RawQuery(q string, args ...any) bob.BaseQuery[expr.Clause] {
	return expr.RawQuery(dialect.Dialect, q, args...)
}
