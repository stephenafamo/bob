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

type QueryType int

const (
	QueryTypeUnknown QueryType = iota
	QueryTypeSelect
	QueryTypeInsert
	QueryTypeUpdate
	QueryTypeDelete
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
	WriteQuery(ctx context.Context, w io.Writer, start int) (args []any, err error)
	// Type returns the query type
	Type() QueryType
}

// BaseQuery wraps common functionality such as cloning, applying new mods and
// the actual query interface implementation
type BaseQuery[E Expression] struct {
	Expression E
	Dialect    Dialect
	QueryType  QueryType
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

func (b BaseQuery[E]) Type() QueryType {
	return b.QueryType
}

func (b BaseQuery[E]) Exec(ctx context.Context, exec Executor) (sql.Result, error) {
	return Exec(ctx, exec, b)
}

func (b BaseQuery[E]) RunHooks(ctx context.Context, exec Executor) (context.Context, error) {
	if l, ok := any(b.Expression).(HookableQuery); ok {
		return l.RunHooks(ctx, exec)
	}

	return ctx, nil
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

func (b BaseQuery[E]) WriteQuery(ctx context.Context, w io.Writer, start int) ([]any, error) {
	return b.Expression.WriteSQL(ctx, w, b.Dialect, start)
}

// Satisfies the Expression interface, but uses its own dialect instead
// of the dialect passed to it
func (b BaseQuery[E]) WriteSQL(ctx context.Context, w io.Writer, _ Dialect, start int) ([]any, error) {
	w.Write([]byte(openPar))
	args, err := b.Expression.WriteSQL(ctx, w, b.Dialect, start)
	w.Write([]byte(closePar))

	return args, err
}

// MustBuild builds the query and panics on error
// useful for initializing queries that need to be reused
func (q BaseQuery[E]) MustBuild(ctx context.Context) (string, []any) {
	return MustBuildN(ctx, q, 1)
}

// MustBuildN builds the query and panics on error
// start numbers the arguments from a different point
func (q BaseQuery[E]) MustBuildN(ctx context.Context, start int) (string, []any) {
	return MustBuildN(ctx, q, start)
}

// Convinient function to build query from start
func (q BaseQuery[E]) Build(ctx context.Context) (string, []any, error) {
	return BuildN(ctx, q, 1)
}

// Convinient function to build query from a point
func (q BaseQuery[E]) BuildN(ctx context.Context, start int) (string, []any, error) {
	return BuildN(ctx, q, start)
}
