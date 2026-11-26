package sqlite

import (
	"context"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/dialect/sqlite/um"
	"github.com/stephenafamo/bob/internal/ormruntime"
)

// UpdateRow updates a row by primary key and copies the returned columns into it.
func (t *Table[T, S, Set, C]) UpdateRow(ctx context.Context, exec bob.Executor, row T, setter Set) error {
	q := t.Update(
		setter.UpdateMod(),
		um.Where(ormruntime.UpdateKey(&t.rowOperations, row)),
		bob.ModFunc[dialect.UpdateEditor](func(q dialect.UpdateEditor) {
			q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
				return t.BeforeUpdateHooks.RunHooks(ctx, exec, S{row})
			})
		}),
	)
	return ormruntime.Refresh(&t.rowOperations, ctx, exec, row, q.One)
}

func (v *View[T, S, C]) RunAfterQueryHooks(ctx context.Context, exec bob.Executor, typ bob.QueryType, rows S) error {
	return (ormruntime.AfterModelHooks[S]{Select: &v.AfterSelectHooks}).Run(ctx, exec, typ, rows)
}

func (t *Table[T, S, Set, C]) RunAfterQueryHooks(ctx context.Context, exec bob.Executor, typ bob.QueryType, rows S) error {
	return (ormruntime.AfterModelHooks[S]{
		Select: &t.AfterSelectHooks, Insert: &t.AfterInsertHooks, Update: &t.AfterUpdateHooks,
		Delete: &t.AfterDeleteHooks,
	}).Run(ctx, exec, typ, rows)
}

func (v *View[T, S, C]) lookup(where bob.Expression, cols ...string) *ViewQuery[T, S] {
	q := v.Query(sm.Where(where))
	if len(cols) != 0 {
		q = q.WithMods(sm.Columns(v.ColumnsExpr().Only(cols...)))
	}
	return q
}
