package genruntime

import (
	"context"
	"io"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal/ormruntime"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/bob/types"
	"github.com/stephenafamo/scan"
)

// ParentCounts consumes a fresh relationship query. Group by the physical
// key after hooks run, and deliver each group's count to all its owners.
func ParentCounts[Q ormruntime.Countable[Q]](ctx context.Context, exec bob.Executor, q orm.ExecQuery[Q],
	n int, owners bob.Expression, groups ...any,
) ([]int64, error) {
	ctx, err := q.RunHooks(ctx, exec)
	if err != nil {
		return nil, err
	}
	type result struct {
		Parents types.JSON[[]int]
		Count   int64
	}
	count := ormruntime.CountQuery(q.BaseQuery, []any{owners, "count(*) AS count"}, groups...)
	rows, err := bob.All(ctx, exec, count, scan.StructMapper[result]())
	if err != nil {
		return nil, err
	}
	counts := make([]int64, n)
	for _, r := range rows {
		for _, i := range r.Parents.Val {
			counts[i] += r.Count
		}
	}
	return counts, nil
}

// PreloadCount adds a correlated count to the select list and delivers it
// to the model. An absent column or SQL NULL leaves the count unloaded.
func PreloadCount[Q interface {
	orm.PreloadableQuery
	ormruntime.Countable[Q]
}](name string, subquery func(string) bob.BaseQuery[Q]) orm.Preloader[Q] {
	column := "__count_" + name
	return func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader) {
		mod := bob.ModFunc[Q](func(q Q) {
			value := ormruntime.CountQuery(subquery(parent), []any{"count(*)"})
			q.AppendPreloadSelect(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
				args, err := value.WriteSQL(ctx, w, d, start)
				if err != nil {
					return nil, err
				}
				if _, err := w.WriteString(" AS "); err != nil {
					return nil, err
				}
				d.WriteQuoted(w, column)
				return args, nil
			}))
		})
		mapper := func(_ context.Context, cols []string) (scan.BeforeFunc, scan.AfterMod) {
			idx := slices.Index(cols, column)
			return func(r *scan.Row) (any, error) {
					if idx < 0 {
						return nil, nil
					}
					var count *int64
					r.ScheduleScanByIndex(idx, &count)
					return &count, nil
				}, func(link, model any) error {
					if count, ok := link.(**int64); ok && *count != nil {
						if m, ok := model.(interface{ PreloadCount(string, int64) error }); ok {
							return m.PreloadCount(name, **count)
						}
					}
					return nil
				}
		}
		return mod, mapper, nil
	}
}
