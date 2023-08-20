package bob

import (
	"context"
	"database/sql"
	"io"

	"github.com/qdm12/reprint"
	"github.com/stephenafamo/scan"
)

// To pervent unnecessary allocations
const (
	openPar  = "("
	closePar = ")"
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

var (
	_ Loadable     = BaseQuery[Expression]{}
	_ MapperModder = BaseQuery[Expression]{}
)

// BaseQuery wraps common functionality such as cloning, applying new mods and
// the actual query interface implementation
type BaseQuery[E Expression] struct {
	Expression E
	Dialect    Dialect
}

func (b BaseQuery[E]) Clone() BaseQuery[E] {
	if c, ok := any(b.Expression).(interface{ Clone() E }); ok {
		return BaseQuery[E]{
			Expression: c.Clone(),
			Dialect:    b.Dialect,
		}
	}

	return BaseQuery[E]{
		Expression: reprint.This(b.Expression).(E),
		Dialect:    b.Dialect,
	}
}

func (b BaseQuery[E]) Exec(ctx context.Context, exec Executor) (sql.Result, error) {
	return Exec(ctx, exec, b)
}

func (b BaseQuery[E]) GetLoaders() []Loader {
	if l, ok := any(b.Expression).(Loadable); ok {
		return l.GetLoaders()
	}

	return nil
}

func (b BaseQuery[E]) GetMapperMods() []scan.MapperMod {
	if l, ok := any(b.Expression).(MapperModder); ok {
		return l.GetMapperMods()
	}

	return nil
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
	w.Write([]byte(openPar))
	args, err := b.Expression.WriteSQL(w, b.Dialect, start)
	w.Write([]byte(closePar))

	return args, err
}

// MustBuild builds the query and panics on error
// useful for initializing queries that need to be reused
func (q BaseQuery[E]) MustBuild() (string, []any) {
	return MustBuildN(q, 1)
}

// MustBuildN builds the query and panics on error
// start numbers the arguments from a different point
func (q BaseQuery[E]) MustBuildN(start int) (string, []any) {
	return MustBuildN(q, start)
}

// Convinient function to build query from start
func (q BaseQuery[E]) Build() (string, []any, error) {
	return BuildN(q, 1)
}

// Convinient function to build query from a point
func (q BaseQuery[E]) BuildN(start int) (string, []any, error) {
	return BuildN(q, start)
}

// Convinient function to cache a query
func (q BaseQuery[E]) Cache() (BaseQuery[*cached], error) {
	return CacheN(q, 1)
}

// Convinient function to cache a query from a point
func (q BaseQuery[E]) CacheN(start int) (BaseQuery[*cached], error) {
	return CacheN(q, start)
}
