package sectionload

// Unit coverage for declaredCountPreloadsShim, including the multi-hop
// (through) case: sectionload's own declaredCountPreloads/declaredCountPreloadsShim never
// look at the CONTENTS of a PreloadColumns element's rendered SQL —only
// at the trailing `AS "<alias>"` header — so a correlated subquery with
// an extra INNER JOIN inside it (what bob's generated countPreloader
// emits for a two-hop through relationship) must be read exactly the same
// way as a single-hop one. This is a hand-written fixture (a fake
// bob.Expression), not a real generated PreloadCount call, because a
// generated PreloadCount is not necessarily a through (multi-hop) count
// in any given schema.

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
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "comments" WHERE "comments"."post_id" = "posts"."id") AS "__count_Comments"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].RelName != "Comments" || got[0].Alias != "__count_Comments" {
		t.Fatalf("got %+v, want {RelName: Comments, Alias: __count_Comments}", got[0])
	}
}

// TestDeclaredCountPreloadsShim_MultiHop: a two-hop through relationship's correlated subquery is a
// SINGLE column in PreloadColumns whose body happens to contain an
// INNER JOIN — declaredCountPreloadsShim must read its trailing
// AS "<alias>" the exact same way it does for a single-hop count,
// because it never inspects the subquery body at all (only the
// countPreloadAliasRe match against the string's own tail).
func TestDeclaredCountPreloadsShim_MultiHop(t *testing.T) {
	cols := []any{
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "author_teams" INNER JOIN "authors" ON "authors"."id" = "author_teams"."author_id" WHERE "author_teams"."team_id" = "teams"."id") AS "__count_Author"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len(got) = %d, want 1", len(got))
	}
	if got[0].RelName != "Author" || got[0].Alias != "__count_Author" {
		t.Fatalf("got %+v, want {RelName: Author, Alias: __count_Author}", got[0])
	}
}

func TestDeclaredCountPreloadsShim_IgnoresNonCountColumns(t *testing.T) {
	cols := []any{
		// A JOIN Preload's own column: has an AS alias, but not the
		// "__count_" prefix — must be skipped, not misread as a count.
		fakeCountPreloadExpr{sql: `"authors"."name" AS "author_name"`},
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
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "comments" WHERE "comments"."post_id" = "posts"."id") AS "__count_Comments"`},
		fakeCountPreloadExpr{sql: `(SELECT count(*) FROM "post_tags" WHERE "post_tags"."post_id" = "posts"."id") AS "__count_PostTags"`},
	}
	got, err := declaredCountPreloadsShim(context.Background(), cols, dialect.Dialect)
	if err != nil {
		t.Fatalf("declaredCountPreloadsShim: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].RelName != "Comments" || got[1].RelName != "PostTags" {
		t.Fatalf("got %+v, want [Comments, PostTags] in order", got)
	}
}
