package mssql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

func RawQuery(q string, args ...any) bob.BaseQuery[expr.Clause] {
	return expr.RawQuery(dialect, q, args...)
}
