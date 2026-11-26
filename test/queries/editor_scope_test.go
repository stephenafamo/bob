package queries_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stephenafamo/bob"
	my "github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/psql"
	pg "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	sq "github.com/stephenafamo/bob/dialect/sqlite/dialect"
)

func TestEditorScope(t *testing.T) {
	t.Run("mysql/select", checkScope(my.NewSelect))
	t.Run("mysql/insert", checkScope(my.NewInsert))
	t.Run("mysql/update", checkScope(my.NewUpdate))
	t.Run("mysql/delete", checkScope(my.NewDelete))
	t.Run("mysql/values", checkScope(my.NewValues))
	t.Run("postgres/select", checkScope(pg.NewSelect))
	t.Run("postgres/insert", checkScope(pg.NewInsert))
	t.Run("postgres/update", checkScope(pg.NewUpdate))
	t.Run("postgres/delete", checkScope(pg.NewDelete))
	t.Run("postgres/values", checkScope(pg.NewValues))
	t.Run("postgres/merge", checkScope(pg.NewMerge))
	t.Run("sqlite/select", checkScope(sq.NewSelect))
	t.Run("sqlite/insert", checkScope(sq.NewInsert))
	t.Run("sqlite/update", checkScope(sq.NewUpdate))
	t.Run("sqlite/delete", checkScope(sq.NewDelete))
	t.Run("sqlite/values", checkScope(sq.NewValues))
}

func TestContextualEditorClosesOnError(t *testing.T) {
	var saved psql.SelectEditor
	q := psql.Select(sm.Columns("1"), bob.ModFunc[psql.SelectEditor](func(e psql.SelectEditor) {
		e.AppendContextualModFunc(func(ctx context.Context, e psql.SelectEditor) (context.Context, error) {
			saved = e
			e.AppendWhere("false")
			return ctx, context.Canceled
		})
	}))
	if _, _, err := q.Build(t.Context(), nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("build error: got %v, want context.Canceled", err)
	}
	if saved == nil {
		t.Fatal("contextual mod did not run")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("editor remained usable after a failed build")
		}
	}()
	saved.AppendWhere("true")
}

func checkScope[E any](build func([]bob.QueryHook[E], ...bob.Mod[E]) bob.QueryBuilder[E]) func(*testing.T) {
	return func(t *testing.T) {
		var first E
		var second E
		base := build(nil)
		child := base.WithMods(bob.ModFunc[E](func(e E) { first = e }), bob.ModFunc[E](func(e E) { second = e }))
		if any(first) != any(second) {
			t.Fatal("mods in one batch received different editors")
		}
		if _, ok := any(first).(bob.Expression); ok {
			t.Fatal("editor exposes a renderable statement")
		}
		_ = child.WithMods(bob.ModFunc[E](func(e E) {
			if any(e) == any(first) {
				t.Fatal("another batch reused the first editor")
			}
		}))
		defer func() {
			if recover() == nil {
				t.Fatal("editor remained usable after WithMods")
			}
		}()
		switch e := any(first).(type) {
		case interface{ AppendWhere(any) }:
			e.AppendWhere("false")
		case interface{ AppendValues(...bob.Expression) }:
			e.AppendValues(nil)
		case pg.MergeEditor:
			e.SetOnly(true)
		default:
			t.Fatal("no operation to test")
		}
	}
}
