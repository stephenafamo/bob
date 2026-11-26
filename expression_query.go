package bob

import (
	"context"
	"database/sql"
)

// ExpressionQuery wraps a renderable expression as a Query.
type ExpressionQuery[E Expression] struct {
	queryBody[E]
}

// QueryFromExpression wraps a read-only expression.
// As with expressions passed to a mod, do not modify e after construction.
func QueryFromExpression[E Expression](e E, d Dialect, typ QueryType) ExpressionQuery[E] {
	return ExpressionQuery[E]{queryBody: queryBody[E]{expression: e, dialect: d, queryType: typ}}
}

// RunHooks returns the resolved query. The original retains its hooks.
func (q ExpressionQuery[E]) RunHooks(ctx context.Context, exec Executor) (context.Context, Query, error) {
	// Keep the outer query type but use the inner hook's SQL and callbacks.
	if inner, ok := any(q.expression).(HookableQuery); ok {
		ctx, stmt, err := RunQueryHooks(ctx, exec, inner)
		return ctx, QueryFromExpression(stmt, q.dialect, q.queryType), err
	}
	return ctx, q, nil
}

func (q ExpressionQuery[E]) Exec(ctx context.Context, exec Executor) (sql.Result, error) {
	return Exec(ctx, exec, q)
}

func (q ExpressionQuery[E]) Cache(ctx context.Context, exec Executor) (CachedQuery, error) {
	return CacheN(ctx, exec, q, 1)
}

func (q ExpressionQuery[E]) CacheN(ctx context.Context, exec Executor, start int) (CachedQuery, error) {
	return CacheN(ctx, exec, q, start)
}

// MustBuild builds the query and panics on error
// useful for initializing queries that need to be reused
func (q ExpressionQuery[E]) MustBuild(ctx context.Context, exec Executor) (string, []any) {
	return MustBuildN(ctx, exec, q, 1)
}

// MustBuildN builds the query and panics on error
// start numbers the arguments from a different point
func (q ExpressionQuery[E]) MustBuildN(ctx context.Context, exec Executor, start int) (string, []any) {
	return MustBuildN(ctx, exec, q, start)
}

// Convenient function to build query from start
func (q ExpressionQuery[E]) Build(ctx context.Context, exec Executor) (string, []any, error) {
	return BuildN(ctx, exec, q, 1)
}

// Convenient function to build query from a point
func (q ExpressionQuery[E]) BuildN(ctx context.Context, exec Executor, start int) (string, []any, error) {
	return BuildN(ctx, exec, q, start)
}
