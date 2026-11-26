package clause

import (
	"slices"

	"github.com/stephenafamo/bob"
)

// CopyList copies a clause list and its mutable fields.
// Nested queries, expression operands and custom expressions stay read-only.
func CopyList(list []bob.Expression) []bob.Expression {
	list = slices.Clone(list)
	for i, e := range list {
		list[i] = copyClause(e)
	}
	return list
}

func copyClause(e bob.Expression) bob.Expression {
	switch c := e.(type) {
	case CTE:
		return c.copy()
	case *CTE:
		return copyPtr(c, CTE.copy)
	case NamedWindow:
		return c.copy()
	case *NamedWindow:
		return copyPtr(c, NamedWindow.copy)
	case Lock:
		return c.copy()
	case *Lock:
		return copyPtr(c, Lock.copy)
	case ConflictClause:
		return c.Copy()
	case *ConflictClause:
		return copyPtr(c, ConflictClause.Copy)
	case *OrderDef:
		if c != nil {
			order := *c
			return &order
		}
		return c
	default:
		return e
	}
}

func copyPtr[T any](p *T, copyValue func(T) T) *T {
	if p == nil {
		return nil
	}
	v := copyValue(*p)
	return &v
}

func (c CTE) copy() CTE {
	c.Columns = slices.Clone(c.Columns)
	c.Search.Columns = slices.Clone(c.Search.Columns)
	c.Cycle.Columns = slices.Clone(c.Cycle.Columns)
	if c.Materialized != nil {
		v := *c.Materialized
		c.Materialized = &v
	}
	return c
}

func (w NamedWindow) copy() NamedWindow {
	w.Definition.partitionBy = slices.Clone(w.Definition.partitionBy)
	w.Definition.OrderBy.Expressions = CopyList(w.Definition.OrderBy.Expressions)
	return w
}

func (l Lock) copy() Lock {
	l.Tables = slices.Clone(l.Tables)
	return l
}
