package genruntime

import (
	"context"
	"io"
	"slices"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/internal/ormruntime"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// CountByKey applies the count projection after the related table's hooks.
func CountByKey[R any, Q ormruntime.Countable[Q]](ctx context.Context, exec bob.Executor, q bob.QueryBuilder[Q], columns, groups []any) ([]R, error) {
	count := ormruntime.CountQuery(q, append(slices.Clip(columns), "count(*) AS count"), groups...)
	return bob.All(ctx, exec, count, scan.StructMapper[R]())
}

// AssignCounts looks up each parent by its Go key and sets zero for misses.
func AssignCounts[P, R any, K comparable](parents []*P, results []R, parentKey func(*P) (K, bool), result func(R) (K, int64), set func(*P, int64)) {
	counts := make(map[K]int64, len(results))
	for _, row := range results {
		k, n := result(row)
		counts[k] = n
	}
	for _, p := range parents {
		if p == nil {
			continue
		}
		var count int64
		if k, ok := parentKey(p); ok {
			count = counts[k]
		}
		set(p, count)
	}
}

// PreloadCount adds a correlated count to the select list and delivers it
// to the model. An absent column or SQL NULL leaves the count unloaded.
func PreloadCount[Q interface {
	orm.PreloadableQuery
	ormruntime.Countable[Q]
}](name string, subquery func(string) bob.QueryBuilder[Q]) orm.Preloader[Q] {
	column := "__count_" + name
	return func(parent string) (bob.Mod[Q], scan.MapperMod, []bob.Loader) {
		mod := bob.ModFunc[Q](func(q Q) {
			value := ormruntime.CountQuery(subquery(parent), []any{"count(*)"})
			q.AppendPreloadSelect(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
				args, err := bob.Express(ctx, w, d, start, value)
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
