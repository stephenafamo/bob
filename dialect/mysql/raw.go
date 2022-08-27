package mysql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
)

func RawQuery(q string, args ...any) bob.BaseQuery[expr.Raw] {
	return expr.RawQuery(dialect.Dialect, q, args...)
}
