package psql

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
)

func RawQuery(q string, args ...any) bob.ExpressionQuery[expr.Clause] {
	return expr.RawQuery(dialect.Dialect, q, args...)
}
