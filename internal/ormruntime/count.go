package ormruntime

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type Countable[Q any] interface {
	bob.Expression
	SetSelect(...any)
	SetPreloadSelect(...any)
	SetGroups(...any)
	SetGroupByDistinct(bool)
	SetGroupWith(string)
	ClearOrderBy()
	SetLimit(any)
	SetOffset(any)
	AppendContextualModFunc(func(context.Context, Q) (context.Context, error))
}

func Count[Q Countable[Q]](ctx context.Context, exec bob.Executor, q orm.ExecQuery[Q]) (int64, error) {
	ctx, err := q.RunHooks(ctx, exec)
	if err != nil {
		return 0, err
	}
	return bob.One(ctx, exec, CountQuery(q.BaseQuery.Clone(), []any{"count(1)"}), scan.SingleColumnMapper[int64])
}

// CountQuery consumes a private query after the model's hooks have run.
// Keep its filters and joins, but count all matches, independent of how
// the caller selected, ordered, grouped or paginated the model rows.
func CountQuery[Q Countable[Q]](q bob.BaseQuery[Q], columns []any, groups ...any) bob.Query {
	rewrite := func(s Q) {
		s.SetSelect(columns...)
		s.SetPreloadSelect()
		s.SetGroups(groups...)
		s.SetGroupByDistinct(false)
		s.SetGroupWith("")
		s.ClearOrderBy()
		s.SetLimit(nil)
		s.SetOffset(nil)
		if f, ok := any(s).(interface{ SetFetch(clause.Fetch) }); ok {
			f.SetFetch(clause.Fetch{})
		}
	}
	rewrite(q.Expression)
	// A contextual mod may add ordering or pagination at build time.
	q.Expression.AppendContextualModFunc(func(ctx context.Context, s Q) (context.Context, error) {
		rewrite(s)
		return ctx, nil
	})
	// Only render SQL. Model loaders and mapper mods expect model rows,
	// and neither table hooks nor embedded hooks should run a second time.
	return struct{ bob.Query }{q}
}
