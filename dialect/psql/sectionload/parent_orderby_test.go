package sectionload

// A top-level All/Build call numbers _sl_src rows with row_number(). When
// the caller-supplied parentQuery carries its own ORDER BY, row_number()
// OVER () alone would have no SQL-level guarantee of following it.
// installParentOrdColumn (run.go) therefore appends a row_number() window
// ordered by the same ORDER BY onto the parent query itself, and
// composeIsland's parentOrdCol plumbing makes _sl_src's own row_number()
// follow it explicitly.
//
// These are Build()-only (no DB) SQL-text assertions, the same style as
// multihop_baseline_test.go and builder_route4_test.go: bob.Build the composed
// query and check for the rendered fragments, rather than executing
// against a live database.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// parentOrderParentQuery returns a fresh psql.Select parent query over tk_root
// with a 2-key ORDER BY (id DESC, other_key NULLS LAST) — psqlSelectModdable
// (unlike psql.RawQuery, which multihop_baseline_test.go's fixtures use and
// which installParentOrdColumn treats as opaque), so installParentOrdColumn
// can actually introspect and extend it.
func parentOrderParentQuery() bob.Query {
	return psql.Select(
		sm.Columns(psql.Quote("id")),
		sm.From(psql.Quote("tk_root")),
		sm.OrderBy("id").Desc(),
		sm.OrderBy("other_key").NullsLast(),
	)
}

// TestParentOrderBy_Followed is the positive case: the parent has an
// ORDER BY, so _sl_src's row_number() must follow it explicitly instead of
// leaning on materialization order.
func TestParentOrderBy_Followed(t *testing.T) {
	built, err := Build(
		context.Background(),
		parentOrderParentQuery(),
		multiHopBaselineRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	// The parent query's own row_number() window, ordered by exactly the
	// ORDER BY expressions it declared (DESC / NULLS LAST preserved),
	// aliased _sl_row_ord. sm.OrderBy("id")/("other_key") render their
	// column names unquoted (bob's own rendering for this mod, matching
	// the identical construction already used by compose.go's
	// collapseOrderBy path) — this asserts the exact text, not a
	// normalized form.
	wantParentWindow := `row_number()OVER ( ORDER BY id DESC, other_key NULLS LAST) AS "_sl_row_ord"`
	if !strings.Contains(built.SQL, wantParentWindow) {
		t.Fatalf("missing parent row_number window %q:\n%s", wantParentWindow, built.SQL)
	}

	// The parent's own ORDER BY must still be present verbatim, in
	// addition to the row_number window above: installParentOrdColumn
	// adds a column, it does not replace or remove the ORDER BY (e.g. a
	// LIMIT on the parent must still see the original order).
	wantParentOrderBy := "\nORDER BY id DESC, other_key NULLS LAST\n"
	if strings.Count(built.SQL, wantParentOrderBy) != 1 {
		t.Fatalf("parent ORDER BY must survive unchanged (exactly once):\n%s", built.SQL)
	}

	// _sl_src's own row_number(), now ordered by the parent's _sl_row_ord
	// column instead of row_number() OVER () with no ORDER BY.
	wantSrcOrd := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantSrcOrd) {
		t.Fatalf("missing _sl_src row_number ordered by _sl_row_ord %q:\n%s", wantSrcOrd, built.SQL)
	}

	// The old no-ORDER-BY form must not appear at all.
	if strings.Contains(built.SQL, `row_number() OVER () AS "_sl_ord"`) {
		t.Fatalf("must not fall back to row_number() OVER () when parent has an ORDER BY:\n%s", built.SQL)
	}
}

// TestParentOrderBy_AbsentUnchanged is the negative case: a parent query
// with no ORDER BY must render byte-for-byte the same SQL as before this
// change (installParentOrdColumn's early return, parentOrdCol == "").
func TestParentOrderBy_AbsentUnchanged(t *testing.T) {
	parentNoOrder := psql.Select(
		sm.Columns(psql.Quote("id")),
		sm.From(psql.Quote("tk_root")),
	)

	built, err := Build(
		context.Background(),
		parentNoOrder,
		multiHopBaselineRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `row_number() OVER () AS "_sl_ord"`) {
		t.Fatalf("expected the no-ORDER-BY fallback row_number() OVER (), got:\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, "_sl_row_ord") {
		t.Fatalf("must not add a parent row_number window when the parent has no ORDER BY:\n%s", built.SQL)
	}
}

// TestParentOrderBy_BuildTwiceIsIdempotent guards installParentOrdColumn's
// restore func: calling Build twice on the SAME parent query object must
// not accumulate a second _sl_row_ord window (the mod added by the first
// Build call must be trimmed back off before the second one runs).
func TestParentOrderBy_BuildTwiceIsIdempotent(t *testing.T) {
	parent := parentOrderParentQuery()

	built1, err := Build(
		context.Background(),
		parent,
		multiHopBaselineRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build (1st): %v", err)
	}

	built2, err := Build(
		context.Background(),
		parent,
		multiHopBaselineRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build (2nd): %v", err)
	}

	if built1.SQL != built2.SQL {
		t.Fatalf("Build must be idempotent on the same parent query object:\n1st:\n%s\n2nd:\n%s", built1.SQL, built2.SQL)
	}
	if n := strings.Count(built2.SQL, "_sl_row_ord"); n != 2 {
		// Once in the parent's own SELECT list (as the window's alias),
		// once in _sl_src's ORDER BY referencing it — never more (which
		// would mean the window got added twice).
		t.Fatalf("_sl_row_ord must appear exactly twice (declared + referenced), got %d:\n%s", n, built2.SQL)
	}
}
