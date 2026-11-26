package internal

import (
	"slices"

	"github.com/stephenafamo/bob"
)

// CloneRows copies both slice levels of a VALUES list and keeps the expressions.
// Each row's capacity equals its length so an append cannot change the next row.
func CloneRows[R ~[]bob.Expression](rows []R) []R {
	if len(rows) == 0 {
		return slices.Clone(rows)
	}
	size := 0
	for _, row := range rows {
		size += len(row)
	}
	cells := make([]bob.Expression, size)
	result := make([]R, len(rows))
	for i, row := range rows {
		if row != nil {
			n := copy(cells, row)
			result[i], cells = cells[:n:n], cells[n:]
		}
	}
	return result
}
