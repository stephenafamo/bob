package sectionload

// P-13: unit coverage for declaredCountPreloadsShim, including the
// multi-hop (through) case §8 #6 requires (P-13-preload-count.md §6.9):
// sectionload's own declaredCountPreloads/declaredCountPreloadsShim never
// look at the CONTENTS of a PreloadColumns element's rendered SQL —only
// at the trailing `AS "<alias>"` header — so a correlated subquery with
// an extra INNER JOIN inside it (what bob's generated countPreloader
// emits for a two-hop through relationship, 115_counts.go.tpl) must be
// read exactly the same way as a single-hop one. This is a hand-written
// fixture (a fake bob.Expression), not a real dbmodels PreloadCount call,
// because no real fvuls relationship's generated PreloadCount happens to
// be a through (multi-hop) count today (P-13-preload-count.md §8 #6's
// own note).

import (
	"context"
	"io"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
)

// fakeCountPreloadExpr is a hand-written bob.Expression standing in for
// whatever countPreloader.applyCount (bob_counts.bob.go.tpl) appends to
// PreloadColumns: WriteSQL just emits sql verbatim, exactly like the real
// aliasedExpr/orm.CountPreloadColumn does for its own correlated
// subquery + " AS " + quoted alias.
type fakeCountPreloadExpr struct{ sql string }

func (f fakeCountPreloadExpr) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	_, err := w.WriteString(f.sql)
	return nil, err
}

func TestDeclaredCountPreloadsShim_SingleHop(t *testing.T) {
	cols := []any{
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "task_comments" WHERE "task_comments"."task_id" = "tasks"."id") AS "__count_TaskComments"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].RelName != "TaskComments" || got[0].Alias != "__count_TaskComments" {
		t.Fatalf("got %+v, want {RelName: TaskComments, Alias: __count_TaskComments}", got[0])
	}
}

// TestDeclaredCountPreloadsShim_MultiHop (§8 #6, P-13-preload-count.md
// §6.9): a two-hop through relationship's correlated subquery is a
// SINGLE column in PreloadColumns whose body happens to contain an
// INNER JOIN — declaredCountPreloadsShim must read its trailing
// AS "<alias>" the exact same way it does for a single-hop count,
// because it never inspects the subquery body at all (only the
// countPreloadAliasRe match against the string's own tail).
func TestDeclaredCountPreloadsShim_MultiHop(t *testing.T) {
	cols := []any{
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "group_servers" INNER JOIN "servers" ON "servers"."id" = "group_servers"."server_id" WHERE "group_servers"."group_id" = "groups"."id") AS "__count_Server"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].RelName != "Server" || got[0].Alias != "__count_Server" {
		t.Fatalf("got %+v, want {RelName: Server, Alias: __count_Server}", got[0])
	}
}

func TestDeclaredCountPreloadsShim_IgnoresNonCountColumns(t *testing.T) {
	cols := []any{
		// A JOIN Preload's own column: has an AS alias, but not the
		// "__count_" prefix — must be skipped, not misread as a count.
		fakeCountPreloadExpr{sql: `"servers"."name" AS "server_name"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("len(got) = %d, want 0 (non-count PreloadColumns entry must be skipped)", len(got))
	}
}

func TestDeclaredCountPreloadsShim_MultipleDeclarations(t *testing.T) {
	cols := []any{
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "task_comments" WHERE "task_comments"."task_id" = "tasks"."id") AS "__count_TaskComments"`},
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "tasks_detection_methods" WHERE "tasks_detection_methods"."task_id" = "tasks"."id") AS "__count_TasksDetectionMethods"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].RelName != "TaskComments" || got[1].RelName != "TasksDetectionMethods" {
		t.Fatalf("got %+v, want [TaskComments, TasksDetectionMethods] in order", got)
	}
}
