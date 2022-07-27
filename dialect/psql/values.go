package psql

import "github.com/stephenafamo/bob/clause"

type ValuesQuery struct {
	// insert the group
	clause.OrderBy
	clause.Limit
	clause.Offset
	clause.Fetch
}
