package sectionload

// Tests for the single-stage unionBranch SELECT.
//
// TestOwnSectionColumnsMatchesChildColsOrder pins that ownSectionColumns
// renders the same column names in the same order as childCols(), for
// single-hop, multi-hop and hasKids=true edges (checking only the column
// count would miss reordering and missing or extra .As()).
//
// TestUnionBranchIsSingleStage pins that the rendered SQL contains no derived
// table (`) AS "_sl_t"`).
//
// TestUnionBranchExtractsOptionsOnce pins that unionBranch's route 2 calls
// extractOptionsFragment once and shares the result between
// unionBranchOrdColumn (the row_number() behind _sl_ord) and
// buildEdgeSelect (the applied WHERE/ORDER BY). A bob.ModFunc that counts its
// calls is passed in opts.mods, and the test checks the call count, that the
// _sl_ord side and the WHERE side reference the same extraction (same marker
// column name), and, individually, the row_number() ordering expression, the
// WHERE clause and built.Args (markers that merely appear and agree would not
// detect a missing frag.applyTo call).
//
// TestComposeIslandHasKidsExtractsOptionsOnce and
// TestRunThenExtractsOptionsOnce pin that route 3 (a node with children) and
// a Then boundary call extractOptionsFragment once while building the route 1
// CTE body and the route 4 parent query respectively, and share the result
// (the fragment and whether an ordering exists) with the _sl_ord decision of
// route 3 and with the parentOrdCol decision and post-execution reorder of a
// Then boundary. Both use a stateful bob.ModFunc that counts calls.

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// TestOwnSectionColumnsMatchesChildColsOrder: renders
// ownSectionColumns's actual output and asserts it produces the same
// column expressions, in the same ORDER, as edge.childCols() names an
// own-then-carried expected layout for. Checking only len(got) would not
// catch a reordering between own columns and carried keys, a missing/extra
// .As(), or route 3 (hasKids) reading a carried key back under the wrong
// name — all things a broken ownSectionColumns could do while still
// returning the right column COUNT.
func TestOwnSectionColumnsMatchesChildColsOrder(t *testing.T) {
	t.Run("single_hop", func(t *testing.T) {
		// No carried keys (single-hop edge): every column is the child's
		// own, read from srcAlias by name, in childCols() order.
		edge := thenKeyRootMidEdge()
		n := flatNode{set: 1, parentSet: 0, edge: edge, hasKids: false}
		got := renderColumns(t, ownSectionColumns(n, nil))

		want := make([]string, 0, len(edge.childCols()))
		for _, c := range edge.childCols() {
			want = append(want, fmt.Sprintf("%q.%q", srcAlias, c.Name))
		}
		if err := columnOrderMismatch(want, got); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("multi_hop", func(t *testing.T) {
		// A multi-hop edge (route 2, not yet materialized into a CTE):
		// the child's own scan columns come first (read from srcAlias),
		// THEN the carried key(s) (read from the join alias edgeJoinChain
		// assigned, aliased to their SQL-layout "_sl_k<i>" name) — the
		// same order childCols() concatenates them in.
		edge := multiHopTwoHopEdge()
		_, carried, _ := edgeJoinChain(edge)
		n := flatNode{set: 1, parentSet: 0, edge: edge, hasKids: false}
		got := renderColumns(t, ownSectionColumns(n, carried))

		want := make([]string, 0, len(edge.childCols()))
		for _, c := range edge.childScanCols() {
			want = append(want, fmt.Sprintf("%q.%q", srcAlias, c.Name))
		}
		for _, c := range carried {
			want = append(want, fmt.Sprintf("%q.%q AS %q", c.alias, c.column, c.as))
		}
		if err := columnOrderMismatch(want, got); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("has_kids", func(t *testing.T) {
		// Route 3 (hasKids): every column, carried keys included, is
		// already materialized in this node's own CTE under its final
		// childCols() name — read back from srcAlias by that name, no
		// .As(), and no reference to the join alias at all (the `carried`
		// argument is unused on this path).
		edge := multiHopTwoHopEdge()
		n := flatNode{set: 1, parentSet: 0, edge: edge, hasKids: true}
		got := renderColumns(t, ownSectionColumns(n, nil))

		want := make([]string, 0, len(edge.childCols()))
		for _, c := range edge.childCols() {
			want = append(want, fmt.Sprintf("%q.%q", srcAlias, c.Name))
		}
		if err := columnOrderMismatch(want, got); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("negative_reordering_is_detected", func(t *testing.T) {
		// The comparison above must actually be able to fail: feed
		// columnOrderMismatch a deliberately reordered column list (the
		// multi_hop case's own last two entries swapped) and confirm it
		// reports a mismatch — without touching ownSectionColumns itself
		// (verify the check itself, not just that it currently passes).
		want := []string{`"_sl_t"."id"`, `"_sl_t"."label"`, `"_sl_h1"."parent_id" AS "_sl_k0"`}
		broken := []string{`"_sl_t"."id"`, `"_sl_h1"."parent_id" AS "_sl_k0"`, `"_sl_t"."label"`}
		if err := columnOrderMismatch(want, broken); err == nil {
			t.Fatal("columnOrderMismatch did not detect a reordered column list; the comparison itself is broken")
		}
	})
}

// renderColumns renders cols (the same []any ownSectionColumns/
// unionBranchProjection hand to sm.Columns) into "SELECT <col1>,
// <col2>, ..." and splits the result back into one string per column, in
// order — so a test can compare the actual rendered SQL text of each
// column against an expected list, not just a count.
func renderColumns(t *testing.T, cols []any) []string {
	t.Helper()
	q := psql.Select(sm.Columns(cols...))
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	body := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(sql), "SELECT"))
	parts := strings.Split(body, ",")
	out := make([]string, len(parts))
	for i, p := range parts {
		out[i] = strings.TrimSpace(p)
	}
	return out
}

// columnOrderMismatch reports the first index where got diverges from
// want, or nil when they match exactly (used by
// TestOwnSectionColumnsMatchesChildColsOrder's three positive cases and by
// its own negative_reordering_is_detected subtest, which checks that this
// comparison itself can fail).
func columnOrderMismatch(want, got []string) error {
	if len(want) != len(got) {
		return fmt.Errorf("column count mismatch: got %d, want %d\ngot:  %v\nwant: %v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			return fmt.Errorf("column %d mismatch: got %q, want %q\ngot:  %v\nwant: %v", i, got[i], want[i], got, want)
		}
	}
	return nil
}

// TestUnionBranchIsSingleStage composes an island with one to-many edge via
// composeIsland and checks that the rendered SQL contains no derived table
// (the `) AS "_sl_t"` marker of an inner SELECT wrapped by an outer one).
func TestUnionBranchIsSingleStage(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM tk_root"),
		multiHopBaselineRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if strings.Contains(built.SQL, `) AS "_sl_t"`) {
		t.Fatalf("expected single-stage SQL (no derived table), but found `) AS \"_sl_t\"` in:\n%s", built.SQL)
	}
}

// TestUnionBranchExtractsOptionsOnce: unionBranch
// (route 2, an edge with its own SelectOptions and an ORDER BY, so
// collapseOrderBy applies) must call extractOptionsFragment exactly once
// per branch, and hand that single result to both unionBranchOrdColumn
// (the _sl_ord ordinal) and buildEdgeSelect (the applied WHERE/ORDER BY),
// via edgeSelectSpec.fragment.
//
// countingMod embeds its own call count into a fake column name
// ("_ss_marker<n>") used by BOTH a WHERE clause and the ORDER BY that
// collapseOrderBy turns into _sl_ord's row_number() OVER (...) window. If
// extractOptionsFragment runs once, both places show the same marker. If
// it runs twice (the bug this test guards against — unionBranchOrdColumn
// and buildEdgeSelect each re-extracting the opts independently), the
// ordinal's ORDER BY column and the applied WHERE column diverge, because
// each extraction bumps the counter and gets a different marker.
func TestUnionBranchExtractsOptionsOnce(t *testing.T) {
	var calls int
	countingMod := bob.ModFunc[*pdialect.SelectQuery](func(q *pdialect.SelectQuery) {
		calls++
		marker := fmt.Sprintf("_ss_marker%d", calls)
		sm.Where(psql.Quote(srcAlias, marker).EQ(psql.Arg(calls))).Apply(q)
		sm.OrderBy(marker).Apply(q)
	})

	edge := thenKeyRootMidEdge()
	edge.Options = Mods(countingMod)

	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM tk_root"),
		multiHopBaselineRootParts(),
		NewNode(edge),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if calls != 1 {
		t.Fatalf("extractOptionsFragment ran the opts mod %d times for one branch; want exactly 1:\n%s", calls, built.SQL)
	}

	markers := regexp.MustCompile(`_ss_marker\d+`).FindAllString(built.SQL, -1)
	if len(markers) == 0 {
		t.Fatalf("expected the counting mod's marker column to appear in the generated SQL; got:\n%s", built.SQL)
	}
	for _, m := range markers {
		if m != markers[0] {
			t.Fatalf("the _sl_ord ordinal and the applied WHERE/ORDER BY used different marker columns %v — extractOptionsFragment ran more than once for one branch:\n%s", markers, built.SQL)
		}
	}

	// The checks above only require SOME marker to appear, identically
	// wherever it appears. They would still pass if buildEdgeSelect stopped
	// calling frag.applyTo(q.Expression), since the ordinal column (built
	// from the same fragment by unionBranchOrdColumn, which never touches
	// applyTo) would still carry the one surviving marker. The assertions
	// below pin each of the fragment's two destinations independently: the
	// row_number() ORDER BY expression and the WHERE clause must each show
	// marker1, and the bound argument (from the WHERE, via psql.Arg(calls))
	// must be present and equal to 1. Removing frag.applyTo's call would make
	// the WHERE and Args assertions fail while the ORDER BY one still passes.
	// sm.OrderBy(name) (the countingMod's own ORDER BY mod, same rendering
	// as parent_orderby_test.go's sm.OrderBy("id")) writes the column
	// name bare, with no table qualifier — this is unionBranchOrdColumn's
	// own row_number() OVER (ORDER BY ...) window, built directly from
	// frag.orderBy.Expressions.
	wantOrder := `ORDER BY _ss_marker1`
	if !strings.Contains(built.SQL, wantOrder) {
		t.Fatalf("missing the ordinal's own ORDER BY expression %q:\n%s", wantOrder, built.SQL)
	}
	// psql.Quote(srcAlias, marker) (the countingMod's own WHERE mod)
	// qualifies the column with "_sl_t". This is buildEdgeSelect's
	// frag.applyTo(q.Expression) call merging the fragment's WHERE into the
	// real query. Removing that call would make this assertion fail while
	// the ORDER BY one above still passes (the ordinal never goes through
	// applyTo).
	wantWhere := `"_sl_t"."_ss_marker1"`
	if !strings.Contains(built.SQL, wantWhere) {
		t.Fatalf("missing the applied WHERE's own qualified marker column %q:\n%s", wantWhere, built.SQL)
	}
	if len(built.Args) != 1 || built.Args[0] != 1 {
		t.Fatalf("built.Args = %v, want exactly [1] (the WHERE's psql.Arg(calls) with calls==1):\n%s", built.Args, built.SQL)
	}
}

// TestComposeIslandHasKidsExtractsOptionsOnce (route 3 half): a hasKids node's CTE body (route 1, built by composeIsland)
// must call extractOptionsFragment exactly once, and unionBranch's route 3
// (this node's own UNION ALL branch, reading the CTE back) must use THAT
// same extraction's ordered flag (hasKidsOrdered) rather than an
// independent second call. Before this fix, route 3 called
// hasOrderBy(n.edge.options()) — a second, independent extraction of the
// same opts, which for a stateful bob.ModFunc could disagree with the
// first (the CTE body materializing a real _sl_row_ord ordinal while
// route 3 still read it back as NULL, or the reverse).
//
// Fixture: builderCapParent -[Child, countingMod]-> builderCapChild (hasKids,
// because it has its own child) -[Grandchild, no options]-> hasKidsGrandchild
// — the same tree shape as TestComposeIsland_HasKidsOrderBy_AppliedOnceAtCTE
// (options_haskids_test.go), reusing its fixtures (hasKidsParentBuf,
// hasKidsGrandchildEdge).
func TestComposeIslandHasKidsExtractsOptionsOnce(t *testing.T) {
	var calls int
	countingMod := bob.ModFunc[*pdialect.SelectQuery](func(q *pdialect.SelectQuery) {
		calls++
		marker := fmt.Sprintf("_ss_haskids_marker%d", calls)
		sm.Where(psql.Quote(srcAlias, marker).EQ(psql.Arg(calls))).Apply(q)
		sm.OrderBy(marker).Apply(q)
	})

	orderedChild := builderCapEdgeWith(Mods(countingMod))
	tree := NewNode(orderedChild, NewNode(hasKidsGrandchildEdge()))

	root := partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema:  "",
		Table:   "bld_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[builderCapParent] { return &hasKidsParentBuf{} },
	}}
	plan, err := planIsland([]Node{tree})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	// DFS order: orderedChild (set 1, this test's hasKids node) comes
	// before its own child hasKidsGrandchildEdge (set 2, hasKids=false) —
	// same ordering TestComposeIsland_HasKidsOrderBy_AppliedOnceAtCTE
	// (options_haskids_test.go) relies on for the same tree shape.
	if len(plan.sections) != 2 || !plan.sections[0].hasKids || plan.sections[1].hasKids {
		t.Fatalf("fixture must produce [hasKids=true, hasKids=false]; got %+v", plan.sections)
	}

	parent := psql.RawQuery("SELECT id FROM bld_parents")
	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}

	if calls != 1 {
		t.Fatalf("extractOptionsFragment ran the hasKids node's opts mod %d times; want exactly 1:\n%s", calls, sql)
	}

	// The CTE body's own ordinal (route 1, collapseOrderBy's AppendSelect)
	// and route 3's read-back of that SAME ordinal must both be present —
	// derived from the one extraction above. If hasKidsOrdered disagreed
	// with the CTE body's actual ORDER BY (the pre-fix bug, reachable with
	// a call-parity-dependent mod), one of these two would be missing
	// while the other still rendered.
	wantCTEOrdinal := `row_number()OVER ( ORDER BY _ss_haskids_marker1) AS "_sl_row_ord"`
	if !strings.Contains(sql, wantCTEOrdinal) {
		t.Fatalf("missing the CTE body's own ordinal %q:\n%s", wantCTEOrdinal, sql)
	}
	wantRoute3Ordinal := `"_sl_t"."_sl_row_ord" AS "_sl_ord"`
	if !strings.Contains(sql, wantRoute3Ordinal) {
		t.Fatalf("missing route 3's read-back of the CTE's own ordinal %q — hasKidsOrdered must agree with the CTE body actually having an ORDER BY:\n%s", wantRoute3Ordinal, sql)
	}
}

// singleStageThenFakeExecutor is a bob.Executor that captures every QueryContext
// call's rendered SQL text (unlike thenKeyFakeExecutor, which ignores it), so
// TestRunThenExtractsOptionsOnce can inspect the SECOND call — the Then
// boundary's own composed query, built by runThen — without a real
// database. No ExecContext call is expected: neither the main island nor
// a Then boundary ever issues one.
type singleStageThenFakeExecutor struct {
	calls   int
	queries []string
	rows1   scan.Rows
	rows2   scan.Rows
}

func (e *singleStageThenFakeExecutor) QueryContext(_ context.Context, query string, _ ...any) (scan.Rows, error) {
	e.calls++
	e.queries = append(e.queries, query)
	switch e.calls {
	case 1:
		return e.rows1, nil
	case 2:
		return e.rows2, nil
	default:
		return nil, fmt.Errorf("singleStageThenFakeExecutor: unexpected QueryContext call #%d: %s", e.calls, query)
	}
}

func (e *singleStageThenFakeExecutor) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	return nil, fmt.Errorf("singleStageThenFakeExecutor: ExecContext not implemented (not needed by runThen)")
}

var _ bob.Executor = (*singleStageThenFakeExecutor)(nil)

// TestRunThenExtractsOptionsOnce (Then boundary half): runThen must call extractOptionsFragment exactly once
// per Then boundary, and share that single result between building the
// boundary's own parent query (route 4, via edgeSelectSpec.fragment),
// deciding parentOrdCol, and deciding whether the post-execution reorder
// runs. Before this fix, hooks.go called hasOrderBy(e.options())
// independently at the latter two decisions — three extractions total
// for one Then boundary with an ORDER BY option (buildEdgeSelect's own
// internal one, plus the two hasOrderBy calls).
//
// Fixture: a root-level Then(NewNode(edge)) boundary only (no ordinary
// section), thenKeyRoot -[Mid, countingMod]-> thenKeyMid — reusing
// thenkey_test.go's fixtures (thenKeyRoot/thenKeyRootBuf/thenKeyMid/thenKeyMidBuf/
// thenKeyRootMidEdge, thenKeyFakeRows/thenKeyColsRows). The main island returns one
// parent row (id=100); the Then boundary returns zero rows, which is
// enough for runThen's length checks (0 == 0) to pass without needing
// real thenKeyMid row data — this test's concern is the SQL text and the
// extraction count, not the stitched result.
func TestRunThenExtractsOptionsOnce(t *testing.T) {
	var calls int
	countingMod := bob.ModFunc[*pdialect.SelectQuery](func(q *pdialect.SelectQuery) {
		calls++
		marker := fmt.Sprintf("_ss_then_marker%d", calls)
		sm.Where(psql.Quote(srcAlias, marker).EQ(psql.Arg(calls))).Apply(q)
		sm.OrderBy(marker).Apply(q)
	})

	edge := thenKeyRootMidEdge()
	edge.Options = Mods(countingMod)

	root := TableParts[thenKeyRoot]{
		Table:   "tk_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[thenKeyRoot] { return &thenKeyRootBuf{} },
	}

	tree := Then(NewNode(edge))
	plan, err := planIsland([]Node{tree})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.thens) != 1 {
		t.Fatalf("expected exactly one Then boundary, got %d", len(plan.thens))
	}

	mainCols := expectedColumns(partsAdapter[thenKeyRoot]{root}, plan)
	mainRows := &thenKeyColsRows{
		cols: mainCols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			// thenKeyFakeRows.Scan (thenkey_test.go) wraps a raw int64 into
			// sql.NullInt64 itself for a *sql.NullInt64 destination — the
			// stored row value here must be int64(1), NOT sql.NullInt64{...}.
			{int64(0), int64(1), 100},
		}},
	}

	// Zero rows for the Then boundary's own query: runThen's length
	// checks (len(childKeys) == len(children), both 0) pass trivially, so
	// this test needs no real thenKeyMid row data.
	childCols := expectedColumns(edgeRootParts{edge.child(), edge}, islandPlan{})
	thenRows := &thenKeyColsRows{
		cols:            childCols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{}},
	}

	exec := &singleStageThenFakeExecutor{rows1: mainRows, rows2: thenRows}
	parentQuery := psql.RawQuery(`SELECT id FROM tk_root`)

	if _, err := All[thenKeyRoot](context.Background(), exec, parentQuery, root, tree); err != nil {
		t.Fatalf("All: %v", err)
	}

	if exec.calls != 2 {
		t.Fatalf("QueryContext calls: got %d, want 2 (main island + Then boundary):\n%v", exec.calls, exec.queries)
	}
	if calls != 1 {
		t.Fatalf("extractOptionsFragment ran the Then edge's opts mod %d times; want exactly 1:\n%s", calls, exec.queries[1])
	}

	thenSQL := exec.queries[1]
	// unionBranchOrdColumn-equivalent: route 4's own collapseOrderBy
	// builds this ordinal directly from frag.orderBy.Expressions.
	wantOrdinal := `row_number()OVER ( ORDER BY _ss_then_marker1) AS "_sl_row_ord"`
	if !strings.Contains(thenSQL, wantOrdinal) {
		t.Fatalf("missing route-4 parentQuery's own ordinal %q:\n%s", wantOrdinal, thenSQL)
	}
	// frag.applyTo's own WHERE, qualified with srcAlias.
	wantWhere := `"_sl_t"."_ss_then_marker1"`
	if !strings.Contains(thenSQL, wantWhere) {
		t.Fatalf("missing the applied WHERE marker column in the Then boundary's own SELECT %q:\n%s", wantWhere, thenSQL)
	}
	// composeIsland's own _sl_src ordinal, following parentOrdCol — only
	// rendered this way (instead of the no-ORDER-BY row_number() OVER ()
	// fallback) when parentOrdCol == ordCol, i.e. when `ordered` (this
	// SAME extraction's result) was true.
	wantSrcOrd := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(thenSQL, wantSrcOrd) {
		t.Fatalf("missing composeIsland's own _sl_src ordinal following parentOrdCol %q — parentOrdCol and the parentQuery's real ORDER BY must come from the SAME extraction:\n%s", wantSrcOrd, thenSQL)
	}
}
