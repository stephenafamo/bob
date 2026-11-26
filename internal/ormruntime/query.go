package ormruntime

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// ResolveQueryBuilder prepares the MySQL insert batch before it is split
// into writes. Model callbacks run before the query hooks edit the batch.
func ResolveQueryBuilder[E any](ctx context.Context, exec bob.Executor, q bob.QueryBuilder[E]) (context.Context, bob.QueryBuilder[E], error) {
	ctx, err := q.RunEmbeddedHooks(ctx, exec)
	if err != nil {
		return ctx, q, err
	}
	ctx, stmt, err := q.RunHooks(ctx, exec)
	if err != nil {
		return ctx, q, err
	}
	return ctx, stmt.(bob.QueryBuilder[E]), nil
}

// ModelQuery attaches the table's row mapper to a statement.
func ModelQuery[Ts ~[]T, T, Q any](
	q bob.QueryBuilder[Q],
	mapper scan.Mapper[T],
) *orm.Query[Q, T, Ts, bob.SliceTransformer[T, Ts]] {
	return &orm.Query[Q, T, Ts, bob.SliceTransformer[T, Ts]]{
		ExecQuery: orm.ExecQuery[Q]{QueryBuilder: q},
		Scanner:   mapper,
	}
}
