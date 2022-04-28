package psql

import "github.com/stephenafamo/bob/expr"

type ValuesQuery struct {
	// insert the group
	expr.OrderBy
	expr.Limit
	expr.Offset
	expr.Fetch
}
