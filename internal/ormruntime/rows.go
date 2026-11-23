// Package ormruntime implements model operations shared by Bob's dialects.
package ormruntime

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
)

type rowQuery[T, S any] interface {
	One(context.Context, bob.Executor) (T, error)
	All(context.Context, bob.Executor) (S, error)
	Exists(context.Context, bob.Executor) (bool, error)
}

// Rows provides the methods promoted by each dialect's Table.
type Rows[T any, S ~[]T, Q rowQuery[T, S]] struct {
	mapping orm.RowMapping[T, S]
	lookup  func(bob.Expression, ...string) Q
}

func NewRows[T any, S ~[]T, Q rowQuery[T, S]](m orm.RowMapping[T, S], lookup func(bob.Expression, ...string) Q) Rows[T, S, Q] {
	return Rows[T, S, Q]{mapping: m, lookup: lookup}
}

func (r *Rows[T, S, Q]) equalValues(values bob.Expression) bob.Expression {
	return expr.OP("=", r.mapping.KeyColumns.RowValue(), values)
}

// UpdateKey reads the key after before-update hooks have run.
func UpdateKey[T any, S ~[]T, Q rowQuery[T, S]](r *Rows[T, S, Q], row T) bob.Expression {
	return r.equalValues(bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return r.mapping.KeyValues(row).WriteSQL(ctx, w, d, start)
	}))
}

// Refresh copies only columns, and only after a successful query.
func Refresh[T any, S ~[]T, Q rowQuery[T, S]](
	r *Rows[T, S, Q],
	ctx context.Context,
	exec bob.Executor,
	row T,
	one func(context.Context, bob.Executor) (T, error),
) error {
	fresh, err := one(ctx, exec)
	if err == nil {
		r.mapping.CopyColumns(row, fresh)
	}
	return err
}

func (r *Rows[T, S, Q]) PrimaryKey() expr.ColumnsExpr {
	return r.mapping.KeyColumns
}

func (r *Rows[T, S, Q]) FindByKey(ctx context.Context, exec bob.Executor, key bob.Expression, cols ...string) (T, error) {
	return r.lookup(r.equalValues(key), cols...).One(ctx, exec)
}

func (r *Rows[T, S, Q]) ExistsByKey(ctx context.Context, exec bob.Executor, key bob.Expression) (bool, error) {
	return r.lookup(r.equalValues(key)).Exists(ctx, exec)
}

func (r *Rows[T, S, Q]) ReloadRow(ctx context.Context, exec bob.Executor, row T) error {
	key := r.equalValues(r.mapping.KeyValues(row))
	return Refresh(r, ctx, exec, row, r.lookup(key).One)
}

func (r *Rows[T, S, Q]) ReloadRows(ctx context.Context, exec bob.Executor, rows S) error {
	if len(rows) == 0 {
		return nil
	}
	fresh, err := r.lookup(r.mapping.In(rows)).All(ctx, exec)
	if err == nil {
		r.mapping.CopyMatching(rows, fresh...)
	}
	return err
}
