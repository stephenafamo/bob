package bob

import (
	"context"
	"database/sql"
	"slices"
)

// QuerySource renders a query and can derive a new source without changing the original.
type QuerySource[E any] interface {
	Expression
	WithMods([]Mod[E]) QuerySource[E]
}

// QueryBuilder is a query that accepts mods for editor E.
type QueryBuilder[E any] struct {
	queryBody[QuerySource[E]]
	hooks []QueryHook[E]
}

func NewQueryBuilder[E any](source QuerySource[E], d Dialect, typ QueryType, hooks ...QueryHook[E]) QueryBuilder[E] {
	return QueryBuilder[E]{queryBody: queryBody[QuerySource[E]]{expression: source, dialect: d, queryType: typ}, hooks: slices.Clone(hooks)}
}

// WithMods returns a new query. All mods in the batch use the same editor.
func (s QueryBuilder[E]) WithMods(mods ...Mod[E]) QueryBuilder[E] {
	if len(mods) != 0 {
		s.expression = s.expression.WithMods(mods)
	}
	return s
}

func (s QueryBuilder[E]) RunHooks(ctx context.Context, exec Executor) (context.Context, Query, error) {
	if len(s.hooks) == 0 {
		return ctx, s, nil
	}
	hooks := s.hooks
	s.hooks = nil
	if skip, _ := ctx.Value(SkipQueryHooksKey{}).(bool); skip {
		return ctx, s, nil
	}
	var err error
	s = s.WithMods(ModFunc[E](func(e E) {
		ctx, err = runHooks(ctx, exec, e, hooks)
	}))
	return ctx, s, err
}

func (s QueryBuilder[E]) Exec(ctx context.Context, exec Executor) (sql.Result, error) {
	return Exec(ctx, exec, s)
}

func (s QueryBuilder[E]) Cache(ctx context.Context, exec Executor) (CachedQuery, error) {
	return Cache(ctx, exec, s)
}

func (s QueryBuilder[E]) CacheN(ctx context.Context, exec Executor, start int) (CachedQuery, error) {
	return CacheN(ctx, exec, s, start)
}

// MustBuild builds the query and panics on error
// useful for initializing queries that need to be reused
func (q QueryBuilder[E]) MustBuild(ctx context.Context, exec Executor) (string, []any) {
	return MustBuildN(ctx, exec, q, 1)
}

// MustBuildN builds the query and panics on error
// start numbers the arguments from a different point
func (q QueryBuilder[E]) MustBuildN(ctx context.Context, exec Executor, start int) (string, []any) {
	return MustBuildN(ctx, exec, q, start)
}

// Convenient function to build query from start
func (q QueryBuilder[E]) Build(ctx context.Context, exec Executor) (string, []any, error) {
	return BuildN(ctx, exec, q, 1)
}

// Convenient function to build query from a point
func (q QueryBuilder[E]) BuildN(ctx context.Context, exec Executor, start int) (string, []any, error) {
	return BuildN(ctx, exec, q, start)
}
