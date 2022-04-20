package psql

import "github.com/stephenafamo/typesql/expr"

type ValuesQuery struct {
	// insert the group
	expr.OrderBy
	expr.Limit
	expr.Offset
	expr.Fetch
}
