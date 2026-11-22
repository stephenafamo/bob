package orm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
)

// RowMapping describes the schema-specific parts of refreshing a model.
// KeyColumns and KeyValues follow primary-key constraint order.
type RowMapping[T any, S ~[]T] struct {
	KeyColumns   expr.ColumnsExpr
	KeyValues    func(T) bob.Expression
	CopyColumns  func(T, T)
	CopyMatching func(S, ...T)
}

// Equal builds a primary-key predicate. Read the row's key at build time,
// so before-update hooks can still change it before the SQL is executed.
func (m RowMapping[T, S]) Equal(row T) bob.Expression {
	return m.EqualValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return m.KeyValues(row).WriteSQL(ctx, w, d, start)
	}))
}

// EqualValues matches an already bound key, or a tuple for a composite key.
func (m RowMapping[T, S]) EqualValues(values bob.Expression) bob.Expression {
	return expr.OP("=", m.KeyColumns.RowValue(), values)
}

// In builds a primary-key predicate for a batch, matching nothing when empty.
func (m RowMapping[T, S]) In(rows S) bob.Expression {
	if len(rows) == 0 {
		return expr.Raw("NULL")
	}
	return expr.OP(
		"IN",
		m.KeyColumns.RowValue(),
		bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
			keys := make([]bob.Expression, len(rows))
			for i, row := range rows {
				keys[i] = m.KeyValues(row)
			}
			return bob.ExpressSlice(ctx, w, d, start, keys, "(", ", ", ")")
		}),
	)
}

// RefreshRow copies only columns, and only after a successful query.
func (m RowMapping[T, S]) RefreshRow(ctx context.Context, exec bob.Executor, row T, one func(context.Context, bob.Executor) (T, error)) error {
	fresh, err := one(ctx, exec)
	if err == nil {
		m.CopyColumns(row, fresh)
	}
	return err
}

// RowQuery is the part of a dialect's select query used for primary-key lookups.
type RowQuery[T, S any] interface {
	One(context.Context, bob.Executor) (T, error)
	All(context.Context, bob.Executor) (S, error)
	Exists(context.Context, bob.Executor) (bool, error)
}

// RowOperations implements primary-key lookup and reload using the dialect's select query.
type RowOperations[T any, S ~[]T] struct {
	mapping RowMapping[T, S]
	lookup  func(bob.Expression, ...string) RowQuery[T, S]
}

func NewRowOperations[T any, S ~[]T](
	m RowMapping[T, S],
	lookup func(bob.Expression, ...string) RowQuery[T, S],
) RowOperations[T, S] {
	return RowOperations[T, S]{mapping: m, lookup: lookup}
}

// RowMapping returns the mapping used for both lookups and dialect-specific updates.
func (r *RowOperations[T, S]) RowMapping() RowMapping[T, S] {
	return r.mapping
}

// PrimaryKey returns the table's qualified key columns in constraint order.
func (r *RowOperations[T, S]) PrimaryKey() expr.ColumnsExpr {
	return r.mapping.KeyColumns
}

// FindByKey looks up a bound primary key, optionally selecting only cols.
func (r *RowOperations[T, S]) FindByKey(ctx context.Context, exec bob.Executor, key bob.Expression, cols ...string) (T, error) {
	return r.lookup(r.mapping.EqualValues(key), cols...).One(ctx, exec)
}

func (r *RowOperations[T, S]) ExistsByKey(ctx context.Context, exec bob.Executor, key bob.Expression) (bool, error) {
	return r.lookup(r.mapping.EqualValues(key)).Exists(ctx, exec)
}

func (r *RowOperations[T, S]) ReloadRow(ctx context.Context, exec bob.Executor, row T) error {
	key := r.mapping.EqualValues(r.mapping.KeyValues(row))
	return r.mapping.RefreshRow(ctx, exec, row, r.lookup(key).One)
}

func (r *RowOperations[T, S]) ReloadRows(ctx context.Context, exec bob.Executor, rows S) error {
	if len(rows) == 0 {
		return nil
	}
	fresh, err := r.lookup(r.mapping.In(rows)).All(ctx, exec)
	if err == nil {
		r.mapping.CopyMatching(rows, fresh...)
	}
	return err
}

// AfterModelHooks dispatches to the table's registered hooks. A nil set
// denotes an operation not supported by that table or view.
type AfterModelHooks[S any] struct {
	Select, Insert, Update, Delete, Merge *bob.Hooks[S, bob.SkipModelHooksKey]
}

func (h AfterModelHooks[S]) Run(ctx context.Context, exec bob.Executor, typ bob.QueryType, rows S) error {
	var hooks *bob.Hooks[S, bob.SkipModelHooksKey]
	switch typ {
	case bob.QueryTypeSelect:
		hooks = h.Select
	case bob.QueryTypeInsert:
		hooks = h.Insert
	case bob.QueryTypeUpdate:
		hooks = h.Update
	case bob.QueryTypeDelete:
		hooks = h.Delete
	case bob.QueryTypeMerge:
		hooks = h.Merge
	}
	if hooks == nil {
		return nil
	}
	_, err := hooks.RunHooks(ctx, exec, rows)
	return err
}
