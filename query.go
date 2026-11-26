package bob

import (
	"context"
	"database/sql"
	"io"
	"slices"

	"github.com/stephenafamo/scan"
)

// To pervent unnecessary allocations
const (
	openPar    = "("
	closePar   = ")"
	commaSpace = ", "
)

type QueryType int

const (
	QueryTypeUnknown QueryType = iota
	QueryTypeSelect
	QueryTypeInsert
	QueryTypeUpdate
	QueryTypeDelete
	QueryTypeValues
	QueryTypeMerge
)

func (q QueryType) String() string {
	switch q {
	case QueryTypeSelect:
		return "SELECT"
	case QueryTypeInsert:
		return "INSERT"
	case QueryTypeUpdate:
		return "UPDATE"
	case QueryTypeDelete:
		return "DELETE"
	case QueryTypeValues:
		return "VALUES"
	case QueryTypeMerge:
		return "MERGE"
	default:
		return "UNKNOWN"
	}
}

type Query interface {
	// It should satisfy the Expression interface so that it can be used
	// in places such as a sub-select
	// However, it is allowed for a query to use its own dialect and not
	// the dialect given to it
	Expression
	// start is the index of the args, usually 1.
	// it is present to allow re-indexing in cases of a subquery
	// The method returns the value of any args placed
	WriteQuery(ctx context.Context, w io.StringWriter, start int) (args []any, err error)
	// Type returns the query type
	Type() QueryType
}

type queryBody[E Expression] struct {
	expression E
	dialect    Dialect
	queryType  QueryType
}

func (b queryBody[E]) Type() QueryType {
	return b.queryType
}

func (b queryBody[E]) RunEmbeddedHooks(ctx context.Context, exec Executor) (context.Context, error) {
	return runEmbeddedHooks(ctx, exec, b.expression)
}

func (b queryBody[E]) GetLoaders(ctx context.Context) []Loader {
	return slices.Clone(loadersFor(ctx, b.expression))
}

func (b queryBody[E]) GetMapperMods(ctx context.Context) []scan.MapperMod {
	return slices.Clone(mapperModsFor(ctx, b.expression))
}

func (b queryBody[E]) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	// If it a query, just call its WriteQuery method
	if e, ok := any(b.expression).(interface {
		WriteQuery(context.Context, io.StringWriter, int) ([]any, error)
	}); ok {
		return e.WriteQuery(ctx, w, start)
	}

	return b.expression.WriteSQL(ctx, w, b.dialect, start)
}

// As wraps the query in parentheses and aliases it: (<query>) AS "alias".
// Useful when using a query as a correlated subquery in a column list.
func (b QueryBuilder[E]) As(alias string) Expression {
	return aliasedQuery{inner: b, alias: alias}
}

func (b ExpressionQuery[E]) As(alias string) Expression {
	return aliasedQuery{inner: b, alias: alias}
}

type aliasedQuery struct {
	inner Expression
	alias string
}

func (a aliasedQuery) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	args, err := Express(ctx, w, d, start, a.inner)
	if err != nil {
		return args, err
	}
	w.WriteString(" AS ")
	d.WriteQuoted(w, a.alias)
	return args, nil
}

// Satisfies the Expression interface, but uses its own dialect instead
// of the dialect passed to it
func (b queryBody[E]) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	// If it a query, don't wrap it in parentheses
	// it may already do this on its own and we don't want to double wrap
	if e, ok := any(b.expression).(Query); ok {
		return e.WriteSQL(ctx, w, d, start)
	}

	w.WriteString(openPar)
	args, err := b.expression.WriteSQL(ctx, w, b.dialect, start)
	w.WriteString(closePar)

	return args, err
}

func BindNamed[Arg any](ctx context.Context, q Query, args Arg) BoundQuery[Arg] {
	return BoundQuery[Arg]{Query: q, namedArgs: args}
}

type BoundQuery[Arg any] struct {
	Query
	namedArgs Arg
}

func (b BoundQuery[Arg]) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	args, err := b.Query.WriteQuery(ctx, w, start)
	if err != nil {
		return nil, err
	}

	return bindArgs(args, b.namedArgs)
}

// Satisfies the Expression interface, but uses its own dialect instead
// of the dialect passed to it
func (b BoundQuery[E]) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	args, err := b.Query.WriteSQL(ctx, w, d, start)
	if err != nil {
		return nil, err
	}

	return bindArgs(args, b.namedArgs)
}

func (b BoundQuery[E]) Exec(ctx context.Context, exec Executor) (sql.Result, error) {
	return Exec(ctx, exec, b)
}

func (b BoundQuery[E]) RunHooks(ctx context.Context, exec Executor) (context.Context, Query, error) {
	ctx, q, err := RunQueryHooks(ctx, exec, b.Query)
	b.Query = q
	return ctx, b, err
}

func (b BoundQuery[E]) RunEmbeddedHooks(ctx context.Context, exec Executor) (context.Context, error) {
	return runEmbeddedHooks(ctx, exec, b.Query)
}

func (b BoundQuery[E]) GetLoaders(ctx context.Context) []Loader {
	return loadersFor(ctx, b.Query)
}

func (b BoundQuery[E]) GetMapperMods(ctx context.Context) []scan.MapperMod {
	return mapperModsFor(ctx, b.Query)
}
