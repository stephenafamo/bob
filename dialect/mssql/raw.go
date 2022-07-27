package mssql

import (
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/query"
)

func Raw(q string, args ...any) query.BaseQuery[expr.Raw] {
	return expr.RawQuery(dialect, q, args...)
}
