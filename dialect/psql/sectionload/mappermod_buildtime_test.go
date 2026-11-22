package sectionload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

type (
	mapperModModel struct{ ID int }
	mapperModBuf   struct{ id int }
)

func (b *mapperModBuf) TargetsFor(cols []Column) []any             { return []any{&b.id} }
func (b *mapperModBuf) BuildFor(modelCols []Column) mapperModModel { return mapperModModel{ID: b.id} }

func mapperModParts() TableParts[mapperModModel] {
	return TableParts[mapperModModel]{
		Table:   "mm_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[mapperModModel] { return &mapperModBuf{} },
	}
}

type mapperModCountingExecutor struct{ queries int }

func (e *mapperModCountingExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	e.queries++
	return nil, errors.New("mapperMod: reached the database")
}

func (e *mapperModCountingExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("mapperMod: ExecContext not implemented")
}

// dummyMapperMod: scan.MapperMod is func(context.Context, cols)
// (BeforeFunc, AfterMod), and cols is the type alias `type cols = []string`
// inside package scan, so callers outside the package can write []string
// directly. The body is a no-op because the tests only count mods.
func dummyMapperMod(ctx context.Context, c []string) (scan.BeforeFunc, scan.AfterMod) {
	return nil, nil
}

// TestCheckMapperMods_DoesNotDetectSameCountSubstitution: the 1:1 check in
// checkMapperMods (run.go) compares three counts (declared count preloads,
// MapperMods, and total PreloadColumns) and cannot verify the origin of a
// MapperMod, i.e. whether it really came from a PreloadCount. If
// SetMapperMods swaps the count preload's mod for an unrelated MapperMod
// (dummyMapperMod) of the same count, while the total PreloadColumns
// stays the same (the swap only touches the MapperMod and the SELECT
// columns are left in place), checkMapperMods cannot reject it and lets it
// through (declared=1, GetMapperMods()==1, PreloadColumns total=1). This
// characterization test pins that behavior.
//
// This is a known residual risk rather than a bug: bob's API offers no way
// to prove a MapperMod's origin by type. If bob's API ever lets a MapperMod
// carry identifying metadata, this test should flip to "must be rejected",
// and should be rewritten at that point.
func TestCheckMapperMods_DoesNotDetectSameCountSubstitution(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("mm_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	// PreloadColumns stays at one entry (simulating the correlated subquery
	// column the count preload added remaining in the SELECT after
	// SetMapperMods). The contents are not inspected, so a dummy is fine.
	inner.AppendPreloadSelect(psql.Raw("1"))
	// Attach exactly one unrelated MapperMod that did not come from a count
	// preload (simulating a deliberate swap via SetMapperMods; the swap
	// itself is not the point, only the final state of "one MapperMod,
	// whatever its origin").
	inner.AppendMapperMod(dummyMapperMod)

	// declared simulates "one PreloadCount is declared" (the check only
	// looks at the number of declarations, not their contents).
	declared := []declaredCountPreload{{RelName: "Comments", Alias: "__count_Comments"}}

	err := checkMapperMods(parent, declared, len(inner.PreloadColumns))
	if err != nil {
		t.Fatalf(
			"checkMapperMods only compares the declared count, the MapperMod "+
				"count and the total PreloadColumns and does not verify the "+
				"origin, so it cannot reject a same-count swap to a non-count "+
				"MapperMod (with PreloadColumns total unchanged); this test "+
				"pins that limitation, so if an error is now returned the "+
				"behavior has flipped: %v",
			err,
		)
	}
}

// TestCheckMapperMods_RejectsWhenPreloadColumnCountMismatches: when the
// declared count and the MapperMod count agree but the total PreloadColumns
// differs from the declared count (simulating a user who adds unrelated
// PreloadColumns on top of the count declaration), the check must reject it.
// Without this branch such a case would slip through for the same reason as
// in TestCheckMapperMods_DoesNotDetectSameCountSubstitution, so this is
// direct evidence that the PreloadColumns-total check works.
func TestCheckMapperMods_RejectsWhenPreloadColumnCountMismatches(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("mm_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	// Two PreloadColumns entries (a number that disagrees with the one declaration).
	inner.AppendPreloadSelect(psql.Raw("1"), psql.Raw("2"))
	inner.AppendMapperMod(dummyMapperMod)

	declared := []declaredCountPreload{{RelName: "Comments", Alias: "__count_Comments"}}

	err := checkMapperMods(parent, declared, len(inner.PreloadColumns))
	if err == nil {
		t.Fatal("expected *UnsupportedMapperModsError (PreloadColumns total differs from the declared count), got nil")
	}
	var mmErr *UnsupportedMapperModsError
	if !errors.As(err, &mmErr) {
		t.Fatalf("expected *UnsupportedMapperModsError, got %T: %v", err, err)
	}
	if mmErr.Count != 1 {
		t.Fatalf("Count: got %d, want 1", mmErr.Count)
	}
}

// TestAllRejectsMapperModAddedByContextualMod: a minimal reproduction of the
// path the real Preloader.Apply takes (adding a MapperMod inside
// AppendContextualModFunc), pinning that the build-time check rejects it.
func TestAllRejectsMapperModAddedByContextualMod(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("mm_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	inner.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.SelectQuery) (context.Context, error) {
			q.AppendMapperMod(dummyMapperMod)
			return ctx, nil
		},
	)

	exec := &mapperModCountingExecutor{}
	_, err := All(context.Background(), exec, parent, mapperModParts())
	if err == nil {
		t.Fatal("expected UnsupportedMapperModsError, got nil")
	}
	var mmErr *UnsupportedMapperModsError
	if !errors.As(err, &mmErr) {
		t.Fatalf("expected *UnsupportedMapperModsError, got %T: %v", err, err)
	}
	if mmErr.Count != 1 {
		t.Fatalf("Count: got %d, want 1", mmErr.Count)
	}
	if exec.queries != 0 {
		t.Fatalf("must be rejected before any DB call, got %d queries", exec.queries)
	}
}

// TestBuildTimeMapperModCheckSkipsNonModdableParent: the concrete type of
// psql.RawQuery, bob.BaseQuery[expr.Clause], is allowed as a known
// non-contextual query: it does not produce UncheckableParentQueryError and
// proceeds to the executor as usual (regression test for the allow-list).
func TestBuildTimeMapperModCheckSkipsNonModdableParent(t *testing.T) {
	exec := &mapperModCountingExecutor{}
	_, err := All(context.Background(), exec, psql.RawQuery("SELECT 1"), mapperModParts())
	var mmErr *UnsupportedMapperModsError
	if errors.As(err, &mmErr) {
		t.Fatalf("non-moddable parent must be skipped, not rejected: %v", err)
	}
	if exec.queries != 1 {
		t.Fatalf("executor should have been reached exactly once, got %d", exec.queries)
	}
}

// TestBuildTimeMapperModCheckRestoresContextualMods verifies restoration by
// behavior: All temporarily wraps a contextual mod that adds a MapperMod and
// rejects it, but a direct bob.Build after All must run the original unwrapped
// mod successfully. If the restore assignment is removed, the direct build
// encounters the leaked checking wrapper and returns UnsupportedMapperModsError.
func TestBuildTimeMapperModCheckRestoresContextualMods(
	t *testing.T,
) {
	ctx := context.Background()
	parent := psql.Select(sm.From(psql.Raw("mm_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			inner = q
		},
	))

	var modRuns []string
	inner.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			modRuns = append(modRuns, "first")
			return ctx, nil
		},
	)
	inner.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			modRuns = append(modRuns, "mapper")
			built.AppendMapperMod(dummyMapperMod)
			return ctx, nil
		},
	)

	wantRuns := []string{"first", "mapper"}
	for call := 1; call <= 2; call++ {
		modRuns = nil
		_, err := All(
			ctx,
			&mapperModCountingExecutor{},
			parent,
			mapperModParts(),
		)

		var mmErr *UnsupportedMapperModsError
		if !errors.As(err, &mmErr) {
			t.Fatalf(
				"call %d: expected *UnsupportedMapperModsError from All, got %T: %v",
				call,
				err,
				err,
			)
		}
		if mmErr.Count != 1 {
			t.Fatalf(
				"call %d: MapperMod count: got %d, want 1",
				call,
				mmErr.Count,
			)
		}
		if !slices.Equal(modRuns, wantRuns) {
			t.Fatalf(
				"call %d: wrapped contextual-mod runs: got %v, want %v",
				call,
				modRuns,
				wantRuns,
			)
		}
		if got := len(inner.ContextualModdable.Mods); got != len(wantRuns) {
			t.Fatalf(
				"call %d: restored contextual-mod count: got %d, want %d",
				call,
				got,
				len(wantRuns),
			)
		}

		modRuns = nil
		if _, _, err := bob.Build(ctx, parent); err != nil {
			t.Fatalf(
				"call %d: direct build after All must use the restored contextual mod, got %T: %v",
				call,
				err,
				err,
			)
		}
		if !slices.Equal(modRuns, wantRuns) {
			t.Fatalf(
				"call %d: direct build contextual-mod runs: got %v, want %v",
				call,
				modRuns,
				wantRuns,
			)
		}
	}
}

type mapperModWrappedQuery struct {
	bob.Query
}

func TestMapperModRejectsWrappedQueryThatHidesContextualMods(t *testing.T) {
	inner := psql.Select(sm.From(psql.Raw("mm_parents")))

	var selectQuery *dialect.SelectQuery
	inner.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	selectQuery.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			built.AppendMapperMod(dummyMapperMod)
			return ctx, nil
		},
	)

	exec := &mapperModCountingExecutor{}
	_, err := All(
		context.Background(),
		exec,
		mapperModWrappedQuery{Query: inner},
		mapperModParts(),
	)

	var checkErr *UncheckableParentQueryError
	if !errors.As(err, &checkErr) {
		t.Fatalf(
			"expected *UncheckableParentQueryError, got %T: %v",
			err,
			err,
		)
	}
	if exec.queries != 0 {
		t.Fatalf(
			"wrapped query must be rejected before the database, got %d queries",
			exec.queries,
		)
	}
}

func TestMapperModRejectsMapperModBeforeLaterContextualModClearsIt(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("mm_parents")))

	var selectQuery *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	selectQuery.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			built.AppendMapperMod(dummyMapperMod)
			return ctx, nil
		},
	)
	selectQuery.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			built.SetMapperMods()
			return ctx, nil
		},
	)

	exec := &mapperModCountingExecutor{}
	_, err := All(
		context.Background(),
		exec,
		parent,
		mapperModParts(),
	)

	var mmErr *UnsupportedMapperModsError
	if !errors.As(err, &mmErr) {
		t.Fatalf(
			"expected *UnsupportedMapperModsError, got %T: %v",
			err,
			err,
		)
	}
	if mmErr.Count != 1 {
		t.Fatalf("Count: got %d, want 1", mmErr.Count)
	}
	if exec.queries != 0 {
		t.Fatalf(
			"transient mapper mod must be rejected before the database, got %d queries",
			exec.queries,
		)
	}
}
