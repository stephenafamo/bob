package query

import (
	"io"

	"github.com/jinzhu/copier"
)

type Query interface {
	// It should satisfy the Expression interface so that it can be used
	// in places such as a sub-select
	// However, it is allowed for a query to use its own dialect and not
	// the dialect given to it
	Expression
	// start is the index of the args, usually 1.
	// it is present to allow re-indexing in cases of a subquery
	// The method returns the value of any args placed
	WriteQuery(w io.Writer, start int) (args []any, err error)
}

type Mod[T any] interface {
	Apply(T)
}

// BaseQuery wraps common functionality such as cloning, applying new mods and
// the actual query interface implementation
type BaseQuery[E Expression] struct {
	Expression E
	Dialect    Dialect
}

func (b BaseQuery[E]) Clone() BaseQuery[E] {
	var b2 BaseQuery[E]
	copier.CopyWithOption(b2, b, copier.Option{
		IgnoreEmpty: true,
		DeepCopy:    true,
	})

	return b2
}

func (b BaseQuery[E]) Apply(mods ...Mod[E]) {
	for _, mod := range mods {
		mod.Apply(b.Expression)
	}
}

func (b BaseQuery[E]) WriteQuery(w io.Writer, start int) ([]any, error) {
	return b.Expression.WriteSQL(w, b.Dialect, start)
}

// Satisfies the Expression interface, but uses its own dialect instead
// of the dialect passed to it
func (b BaseQuery[E]) WriteSQL(w io.Writer, _ Dialect, start int) ([]any, error) {
	return b.Expression.WriteSQL(w, b.Dialect, start)
}
