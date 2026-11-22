package sectionload

// outerParentOrd (run.go) lets the outer placement (row_number() outside the
// parent subquery, keeping LIMIT's top-N heapsort) be used even when the
// parent query's ORDER BY contains elements that are not bare references to
// the parent table's own columns (columns of a joined table, EXISTS
// expressions). Only those elements are hoisted onto the end of the parent
// query's SELECT list as "_sl_row_sort<i>" (sortColPrefix), and the outer
// window's ORDER BY is rewritten to reference that alias. The parent's own
// bare column references stay as they are (the mixed approach).
//
// Like outer_ord_test.go, these are Build()-only (no DB) SQL-text
// assertions on fragments of the bob.Build output.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// hoistSortExistsExpr builds a correlated EXISTS(...) expression with exactly one
// bind parameter (the "= 1" comparison). A sort key with bind parameters
// would be rejected by a plain len(args) != 0 check, so hoisting must accept
// it.
func hoistSortExistsExpr() bob.Expression {
	return psql.Exists(psql.Select(
		sm.Columns(psql.S("1")),
		sm.From(psql.Quote("other_table")),
		sm.Where(psql.Quote("other_table", "root_id").EQ(psql.Arg(1))),
	))
}

func TestHoistSort_JoinedColumnUsesOuterWindow(t *testing.T) {
	parent := outerOrdParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")))

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `"other_table"."name" AS "_sl_row_sort0"`) {
		t.Fatalf("expected the joined column hoisted onto the parent's own SELECT list:\n%s", built.SQL)
	}
	if !strings.Contains(built.SQL, `row_number()OVER ( ORDER BY "tk_root"."_sl_row_sort0") AS "_sl_ord"`) {
		t.Fatalf("expected _sl_src's row_number() to ORDER BY the hoisted alias:\n%s", built.SQL)
	}
	// The parent subquery itself (the "FROM (SELECT ... ) AS "tk_root""
	// span, not _sl_src's own wrapping SELECT which carries the outer
	// row_number()) must carry no row_number() of its own — that would put
	// a WindowAgg back inside the LIMIT-ed subquery and defeat the top-N
	// sort this whole mechanism exists to keep.
	parentSubqueryStart := strings.Index(built.SQL, "FROM (SELECT")
	parentSubqueryEnd := strings.Index(built.SQL, ") AS \"tk_root\"")
	if parentSubqueryStart < 0 || parentSubqueryEnd < 0 || parentSubqueryEnd < parentSubqueryStart {
		t.Fatalf(`missing parent subquery span ("FROM (SELECT" .. ") AS \"tk_root\""):`+"\n%s", built.SQL)
	}
	if strings.Count(built.SQL[parentSubqueryStart:parentSubqueryEnd], "row_number()") != 0 {
		t.Fatalf("parent subquery must not carry its own row_number():\n%s", built.SQL)
	}
}

func TestHoistSort_ExistsExprUsesOuterWindow(t *testing.T) {
	parent := outerOrdParentQuery(
		10,
		sm.OrderBy(hoistSortExistsExpr()).Desc(),
		sm.OrderBy(psql.Quote("tk_root", "id")),
	)

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `AS "_sl_row_sort0"`) {
		t.Fatalf("expected the EXISTS expression hoisted onto the parent's own SELECT list:\n%s", built.SQL)
	}
	wantWindow := `ORDER BY "tk_root"."_sl_row_sort0" DESC, "tk_root"."id") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected the outer window to ORDER BY the hoisted alias then the bare parent column, missing %q:\n%s", wantWindow, built.SQL)
	}
	// The EXISTS expression is rendered twice (parent's own ORDER BY, plus
	// the hoisted SELECT column) — an accepted tradeoff rather than
	// rewriting the parent's own ORDER BY.
	// Each rendering carries its own "= $N" bind parameter, so two EXISTS
	// renderings means two bind parameters in the built query.
	if got := strings.Count(built.SQL, "EXISTS"); got != 2 {
		t.Fatalf("expected the EXISTS expression to appear twice (parent ORDER BY + hoisted SELECT column), got %d occurrences:\n%s", got, built.SQL)
	}
	if got := len(built.Args); got != 2 {
		t.Fatalf("expected 2 bind arguments (one per EXISTS rendering), got %d: %#v\n%s", got, built.Args, built.SQL)
	}
}

func TestHoistSort_MixedKeysHoistOnlyForeign(t *testing.T) {
	parent := outerOrdParentQuery(
		10,
		sm.OrderBy(psql.Quote("other_table", "name")).Desc(),
		sm.OrderBy(psql.Quote("tk_root", "id")),
	)

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `"other_table"."name" AS "_sl_row_sort0"`) {
		t.Fatalf("expected the joined column hoisted as _sl_row_sort0:\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, "_sl_row_sort1") {
		t.Fatalf("the parent's own bare column must not be hoisted a second time as _sl_row_sort1:\n%s", built.SQL)
	}
	wantWindow := `ORDER BY "tk_root"."_sl_row_sort0" DESC, "tk_root"."id") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected the outer window's 2nd element to stay the bare parent column reference, missing %q:\n%s", wantWindow, built.SQL)
	}
}

func TestHoistSort_DirectionAndNullsPreserved(t *testing.T) {
	parent := outerOrdParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")).Desc().NullsLast())

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	wantWindow := `row_number()OVER ( ORDER BY "tk_root"."_sl_row_sort0" DESC NULLS LAST) AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected Direction/Nulls carried over from the original clause.OrderDef onto the hoisted alias, missing %q:\n%s", wantWindow, built.SQL)
	}
}

func TestHoistSort_NoLimitFallsBackToInside(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("tk_root", "id")),
		sm.From(psql.Quote("tk_root")),
		sm.OrderBy(psql.Quote("other_table", "name")),
	)

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_sort0") {
		t.Fatalf("without a LIMIT, outerParentOrd must decline (no hoisting) and fall back to installParentOrdColumn:\n%s", built.SQL)
	}
	wantInner := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantInner) {
		t.Fatalf("expected the inside installParentOrdColumn fallback, missing %q:\n%s", wantInner, built.SQL)
	}
}

func TestHoistSort_DistinctFallsBackToInside(t *testing.T) {
	parent := psql.Select(
		sm.Distinct(),
		sm.Columns(psql.Quote("tk_root", "id")),
		sm.From(psql.Quote("tk_root")),
		sm.OrderBy(psql.Quote("other_table", "name")),
		sm.Limit(10),
	)

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_sort0") {
		t.Fatalf("DISTINCT must not trigger hoisting either (the same opt-out outerParentOrd already has):\n%s", built.SQL)
	}
	if !strings.Contains(built.SQL, `row_number() OVER () AS "_sl_ord"`) {
		t.Fatalf("DISTINCT must keep the no-ORDER-BY-guarantee fallback row_number() OVER ():\n%s", built.SQL)
	}
}

func TestHoistSort_RestoreTrimsHoistedColumn(t *testing.T) {
	parent := outerOrdParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")))

	built1, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build (1st): %v", err)
	}
	built2, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build (2nd): %v", err)
	}

	if built1.SQL != built2.SQL {
		t.Fatalf("a second Build on the same parent query value must produce byte-identical SQL (hoistParentSortKeys's restore func must trim the mod back off):\nfirst:\n%s\nsecond:\n%s", built1.SQL, built2.SQL)
	}
	if strings.Count(built2.SQL, "_sl_row_sort0") != strings.Count(built1.SQL, "_sl_row_sort0") {
		t.Fatalf("the hoisted column must not be added a second time:\n%s", built2.SQL)
	}
}

func TestHoistSort_BareParentColumnsUnchanged(t *testing.T) {
	parent := outerOrdParentQuery(
		10,
		sm.OrderBy(psql.Quote("tk_root", "id")).Desc(),
		sm.OrderBy(psql.Quote("tk_root", "ssvc_priority")),
	)

	built, err := Build(context.Background(), parent, outerOrdRootParts(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_sort") {
		t.Fatalf("an ORDER BY made entirely of bare parent column references must not hoist anything (the output is unchanged from the non-hoisting form):\n%s", built.SQL)
	}
	wantWindow := `"tk_root".*, row_number()OVER ( ORDER BY "tk_root"."id" DESC, "tk_root"."ssvc_priority") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected the same outer-window SQL outer_ord_test.go's TestOuterOrd_LimitUsesOuterWindow produces, missing %q:\n%s", wantWindow, built.SQL)
	}
}

// hoistSortRootPartsWithReservedSortName is outerOrdRootParts's variant that also
// declares a real parent column literally named "_sl_row_sort0" — the
// exact name outerParentOrd would assign the first hoisted sort key
// (sortColPrefix + "0"). It pins that outerParentOrd must decline the whole outer placement
// rather than collide with (or silently resolve to) this column.
func hoistSortRootPartsWithReservedSortName() TableParts[thenKeyRoot] {
	return TableParts[thenKeyRoot]{
		Table: "tk_root",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "ssvc_priority", DBType: "text"},
			{Name: "_sl_row_sort0", DBType: "text"},
		},
		NewBuf: func() Buf[thenKeyRoot] { return &thenKeyRootBuf{} },
	}
}

func TestHoistSort_ReservedNameCollisionFallsBackToInside(t *testing.T) {
	// A joined-table column ORDER BY key would normally be hoisted onto
	// the parent's own SELECT list as "_sl_row_sort0" (see
	// TestHoistSort_JoinedColumnUsesOuterWindow above) — but the parent
	// table here already has a real column of that exact name (see
	// hoistSortRootPartsWithReservedSortName), so outerParentOrd must decline
	// the outer placement entirely and fall back to
	// installParentOrdColumn's inside placement instead.
	parent := outerOrdParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")))

	built, err := Build(context.Background(), parent, hoistSortRootPartsWithReservedSortName(), NewNode(thenKeyRootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	// "_sl_row_sort0" itself legitimately appears in the built SQL here —
	// hoistSortRootPartsWithReservedSortName declares it as a real parent
	// column, so it is selected (and NULL-padded on the sibling UNION ALL
	// branch, "(CAST(NULL AS text)) AS \"_sl_row_sort0\"") like any other
	// column of tk_root. What must NOT appear is hoistParentSortKeys's
	// own rendering of the hoisted ORDER BY expression under that exact
	// alias — the same "<joined column> AS \"_sl_row_sort0\"" shape
	// TestHoistSort_JoinedColumnUsesOuterWindow above asserts DOES appear
	// when there is no name collision.
	if strings.Contains(built.SQL, `"other_table"."name" AS "_sl_row_sort0"`) {
		t.Fatalf("outerParentOrd must not hoist \"other_table\".\"name\" under an alias that collides with the parent's own \"_sl_row_sort0\" column:\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, `ORDER BY "tk_root"."_sl_row_sort0"`) || strings.Contains(built.SQL, `ORDER BY "q"."_sl_row_sort0"`) {
		t.Fatalf("the outer window must not ORDER BY the reserved-name alias once outerParentOrd declines the outer placement:\n%s", built.SQL)
	}
	wantInner := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantInner) {
		t.Fatalf("expected the inside installParentOrdColumn fallback, missing %q:\n%s", wantInner, built.SQL)
	}
}
