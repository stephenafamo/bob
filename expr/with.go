package expr

import (
	"io"

	"github.com/stephenafamo/typesql/query"
)

type With struct {
	CTEs []CTE
}

func (w *With) AppendWith(cte CTE) {
	w.CTEs = append(w.CTEs, cte)
}

func (o With) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return nil, nil
}

type CTE struct {
	Query        query.Query // SQL standard says only select, postgres allows insert/update/delete
	Recursive    bool
	Name         string
	Columns      []string
	Materialized bool
	Search       *CTESearch
	Cycle        *CTECycle
}

func (c CTE) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return nil, nil
}

type searchOrder int

const (
	SearchBreadth searchOrder = iota
	SearchDepth
)

type CTESearch struct {
	Order   searchOrder
	Columns []string
	Set     string
}

type CTECycle struct {
	Columns []string
	Set     string
	To      cycleValue
	Using   string
}

type cycleValue struct {
	Detected   any
	DefaultVal any
}
