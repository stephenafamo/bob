package psql

import (
	"github.com/stephenafamo/bob/builder"
	"github.com/stephenafamo/bob/query"
)

func Raw(q string, args ...any) query.BaseQuery[builder.Raw] {
	return builder.RawQuery(dialect, q, args...)
}
