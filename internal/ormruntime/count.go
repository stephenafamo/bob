package ormruntime

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type Countable[Q any] interface {
	SetSelect(...any)
	SetPreloadSelect(...any)
	ResetGroupBy(...any)
	ClearOrderBy()
	SetLimit(any)
	SetOffset(any)
	AppendContextualModFunc(func(context.Context, Q) (context.Context, error))
}

func Count[Q Countable[Q]](ctx context.Context, exec bob.Executor, q orm.ExecQuery[Q]) (int64, error) {
	return bob.One(ctx, exec, CountQuery(q.QueryBuilder, []any{"count(1)"}), scan.SingleColumnMapper[int64])
}

// CountQuery applies the count projection after query hooks have run, whether
// used as a standalone count or as a correlated subquery.
func CountQuery[Q Countable[Q]](q bob.QueryBuilder[Q], columns []any, groups ...any) bob.Query {
	return countQuery[Q]{q, columns, groups}
}

type countQuery[Q Countable[Q]] struct {
	bob.QueryBuilder[Q]
	columns, groups []any
}

func (c countQuery[Q]) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, bob.Query, error) {
	ctx, stmt, err := c.QueryBuilder.RunHooks(ctx, exec)
	if err != nil {
		return ctx, nil, err
	}
	rewrite := func(s Q) {
		s.SetSelect(c.columns...)
		s.SetPreloadSelect()
		s.ResetGroupBy(c.groups...)
		s.ClearOrderBy()
		s.SetLimit(nil)
		s.SetOffset(nil)
		if f, ok := any(s).(interface{ SetFetch(clause.Fetch) }); ok {
			f.SetFetch(clause.Fetch{})
		}
	}
	q := stmt.(bob.QueryBuilder[Q]).WithMods(bob.ModFunc[Q](func(s Q) {
		rewrite(s)
		// A contextual mod may add ordering or pagination at build time.
		s.AppendContextualModFunc(func(ctx context.Context, s Q) (context.Context, error) {
			rewrite(s)
			return ctx, nil
		})
	}))
	// Only render SQL. Model loaders and mapper mods expect model rows,
	// and neither table hooks nor embedded hooks should run a second time.
	return ctx, struct{ bob.Query }{q}, nil
}
