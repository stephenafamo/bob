package sectionload

// Caller-provided SelectOptions (WHERE/ORDER BY/LIMIT/OFFSET) applied
// to one relationship edge's own SELECT.
//
// These tests exercise buildEdgeSelect directly (no DB, no composeIsland
// scaffolding needed for most of them), the same DB-free white-box style
// of the other white-box tests: hand-rolled Edge/TableParts/Buf fixtures,
// bob.Build to inspect the rendered SQL text and args.
//
// builderCapParent/builderCapChild/builderCapEdge()/builderCapChildBuf are the shared
// fixtures from builder_route4_test.go — reused here rather than duplicated.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func builderCapEdgeWith(opts SelectOptions) Edge[builderCapParent, builderCapChild] {
	e := builderCapEdge()
	e.Options = opts
	return e
}

// leafSpec builds the same edgeSelectSpec unionBranch would build for a
// non-hasKids (leaf) node: route 2, semi-joined against a parent CTE, opts
// applied and ORDER BY collapsed to a row-ordinal column.
func leafSpec(e Edge[builderCapParent, builderCapChild]) edgeSelectSpec {
	child := e.child()
	return edgeSelectSpec{
		edge:            e,
		cols:            child.cols(),
		alias:           srcAlias,
		from:            tableExpr(child.schemaName(), child.tableName()),
		semiFromCTE:     cteName(1),
		opts:            e.options(),
		collapseOrderBy: true,
	}
}

func buildSQL(t *testing.T, spec edgeSelectSpec) (string, []any) {
	t.Helper()
	q, err := buildEdgeSelect(spec)
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}
	sql, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	return sql, args
}

func TestBuildEdgeSelect_NoOptions_Unchanged(t *testing.T) {
	// Byte-for-byte the same as before Options existed: an edge with a
	// zero-value SelectOptions must produce identical SQL to route 2's
	// own baseline (builder_route4_test.go covers route 4's baseline the same
	// way).
	e := builderCapEdge() // Options left zero
	sql, args := buildSQL(t, leafSpec(e))

	if strings.Contains(sql, "row_number") {
		t.Fatalf("no ORDER BY was requested; row_number() must not appear:\n%s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("no options were requested; want 0 args, got %v", args)
	}
	for _, frag := range []string{`"id"`, `"parent_id"`, "IN", `"status"`} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing baseline fragment %q:\n%s", frag, sql)
		}
	}
}

func TestBuildEdgeSelect_OptionsWhere(t *testing.T) {
	e := builderCapEdgeWith(Mods(
		sm.Where(psql.Quote(srcAlias, "id").GT(psql.Arg(5))),
	))
	sql, args := buildSQL(t, leafSpec(e))

	if !strings.Contains(sql, `"id" >`) && !strings.Contains(sql, `_sl_t"."id" >`) {
		t.Fatalf("options WHERE not applied; got:\n%s", sql)
	}
	if len(args) != 1 || args[0] != 5 {
		t.Fatalf("args: got %v, want [5]", args)
	}
	// The match condition sectionload itself depends on must still be
	// present — options are ANDed alongside it, never replacing it.
	for _, frag := range []string{"IN", `"status"`} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("match condition fragment %q lost after applying options; got:\n%s", frag, sql)
		}
	}
}

func TestBuildEdgeSelect_OptionsLimitOffset(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.Limit(10), sm.Offset(5)))
	sql, _ := buildSQL(t, leafSpec(e))

	// bob renders LIMIT/OFFSET counts as literal integers, not bound
	// params (same as any plain sm.Limit/sm.Offset use elsewhere in this
	// codebase) — so the count is asserted in the SQL text, not in args.
	if strings.Count(sql, "LIMIT") != 1 || !strings.Contains(sql, "LIMIT 10") {
		t.Fatalf("want exactly one \"LIMIT 10\"; got:\n%s", sql)
	}
	if strings.Count(sql, "OFFSET") != 1 || !strings.Contains(sql, "OFFSET 5") {
		t.Fatalf("want exactly one \"OFFSET 5\"; got:\n%s", sql)
	}
}

// TestBuildEdgeSelect_OptionsOrderBy_Collapsed covers routes 1/2
// (collapseOrderBy: true): a literal ORDER BY does not survive a UNION ALL
// arm, so it must be rewritten into a
// row_number() OVER (ORDER BY ...) ordinal column, and the literal
// top-level ORDER BY must be gone.
func TestBuildEdgeSelect_OptionsOrderBy_Collapsed(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.OrderBy("id").Desc()))
	sql, _ := buildSQL(t, leafSpec(e))

	if !strings.Contains(sql, "row_number()") || !strings.Contains(sql, "OVER") {
		t.Fatalf("expected a row_number() OVER (...) ordinal column; got:\n%s", sql)
	}
	if !strings.Contains(sql, ordCol) {
		t.Fatalf("expected the ordinal column alias %q; got:\n%s", ordCol, sql)
	}
	if strings.Count(sql, "ORDER BY") != 1 {
		t.Fatalf("want exactly one ORDER BY (inside the OVER(...) window); got:\n%s", sql)
	}
	overIdx := strings.Index(sql, "OVER")
	orderIdx := strings.Index(sql, "ORDER BY")
	if orderIdx < overIdx {
		t.Fatalf("ORDER BY must be inside the OVER(...) window, not a literal top-level clause; got:\n%s", sql)
	}
	if !strings.Contains(sql, "DESC") {
		t.Fatalf("direction (DESC) lost when collapsing to a window function; got:\n%s", sql)
	}
}

// TestBuildEdgeSelect_OptionsOrderBy_LeftLiteralWhenNotCollapsed covers
// buildEdgeSelect's collapseOrderBy: false path in isolation: a literal
// ORDER BY opt is left exactly as bob's own ThenLoad would leave it — no
// row_number()/ordinal rewriting at all. This is
// NOT route 4: hooks.go's runThen sets collapseOrderBy: true there, because a route 4 query
// becomes the new island's own root and needs the same row_number()-based
// order-restoring mechanism every plain root query relies on
// (composeIsland's _sl_src wrapping) — see compose.go's collapseOrderBy
// doc. TestComposeIsland_Route4OrderBy_UsesExplicitOrdinal is the actual
// route 4 regression test; this test only exercises the false branch that
// route 3 (a with-children node's own CTE-read-back branch) uses.
func TestBuildEdgeSelect_OptionsOrderBy_LeftLiteralWhenNotCollapsed(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.OrderBy("id").Desc()))
	child := e.child()
	spec := edgeSelectSpec{
		edge:  e,
		cols:  child.cols(),
		alias: srcAlias,
		from:  tableExpr(child.schemaName(), child.tableName()),
		keys:  [][]any{{1}, {2}},
		opts:  e.options(),
		// collapseOrderBy left false: this test's subject, not route 4's.
	}
	sql, _ := buildSQL(t, spec)

	if strings.Contains(sql, "row_number") {
		t.Fatalf("collapseOrderBy: false must leave ORDER BY literal, not collapse it to row_number(); got:\n%s", sql)
	}
	if !strings.Contains(sql, "ORDER BY") || !strings.Contains(sql, "DESC") {
		t.Fatalf("literal ORDER BY ... DESC expected; got:\n%s", sql)
	}
}

func TestBuildEdgeSelect_OptionsSiblingNonInterference(t *testing.T) {
	// An edge with no options must be unaffected by another edge's
	// options — options.mods live on the Edge value itself and are never
	// read except by the spec explicitly built for that edge's own route.
	plain := builderCapEdge()
	withOpts := builderCapEdgeWith(Mods(sm.Where(psql.Quote(srcAlias, "id").GT(psql.Arg(5)))))

	plainSQL, plainArgs := buildSQL(t, leafSpec(plain))
	_, _ = buildSQL(t, leafSpec(withOpts)) // exercise it; result checked above

	if len(plainArgs) != 0 {
		t.Fatalf("sibling's options must not leak into the plain edge; got args %v", plainArgs)
	}
	if strings.Contains(plainSQL, `"id" >`) {
		t.Fatalf("sibling's WHERE leaked into the plain edge's SQL:\n%s", plainSQL)
	}
}

// TestBuildEdgeSelect_StructureViolation_SelectColumns covers the
// pre-flight structure check: a mod that replaces the SELECT column list changes what
// sectionload's fixed section layout depends on, so it must be rejected
// before the SQL is ever built/executed, via a typed error — never by
// parsing the rendered SQL text.
func TestBuildEdgeSelect_StructureViolation_SelectColumns(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.Columns(psql.Quote(srcAlias, "id"))))
	_, err := buildEdgeSelect(leafSpec(e))
	if err == nil {
		t.Fatal("expected a structure-violation error, got nil")
	}
	var structErr *SelectOptionsStructureError
	if !errors.As(err, &structErr) {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
	if structErr.Field != "select columns" {
		t.Fatalf("Field: got %q, want %q", structErr.Field, "select columns")
	}
	if structErr.RelName != e.RelName {
		t.Fatalf("RelName: got %q, want %q", structErr.RelName, e.RelName)
	}
}

func TestBuildEdgeSelect_StructureViolation_Join(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.InnerJoin(psql.Quote("other_table"))))
	_, err := buildEdgeSelect(leafSpec(e))
	if err == nil {
		t.Fatal("expected a structure-violation error, got nil")
	}
	var structErr *SelectOptionsStructureError
	if !errors.As(err, &structErr) {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
	if structErr.Field != "FROM/JOIN" {
		t.Fatalf("Field: got %q, want %q", structErr.Field, "FROM/JOIN")
	}
}

// TestBuildEdgeSelect_ModCannotClearInternalJoinCondition is the negative
// test for the join condition: a
// caller mod that reaches into the query and clears its own WHERE must
// not be able to remove the match condition sectionload itself depends
// on. Before the fix, opts were applied directly to the real query and
// only SelectList/TableRef/With/Combines were compared before vs. after —
// Where was never checked, so this exact mod deleted the join condition
// and buildEdgeSelect returned it without error. After the fix, opts are
// applied to an isolated, empty query and only their own WHERE is merged
// in (appended, not substituted) — a mod has no reference to the real
// query's Where field at all, so there is nothing for it to clear.
func TestBuildEdgeSelect_ModCannotClearInternalJoinCondition(t *testing.T) {
	e := builderCapEdgeWith(Mods(
		bob.ModFunc[*pdialect.SelectQuery](func(q *pdialect.SelectQuery) {
			q.Where.Conditions = nil
		}),
	))
	sql, _ := buildSQL(t, leafSpec(e))

	for _, frag := range []string{"IN", `"status"`} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("match condition fragment %q was removed by a caller mod; got:\n%s", frag, sql)
		}
	}
}

// TestBuildEdgeSelect_StructureViolation_Distinct and
// TestBuildEdgeSelect_StructureViolation_Fetch cover the rest of the
// structure check: a snapshot that only compared
// SelectList/TableRef/With/Combines, so a mod touching Distinct, Windows,
// Fetch, Locks, Load, EmbeddedHook, ContextualModdable, or any CombinedX
// field passed through unchecked. extractOptionsFragment now rejects all
// of them.
func TestBuildEdgeSelect_StructureViolation_Distinct(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.Distinct()))
	_, err := buildEdgeSelect(leafSpec(e))
	var structErr *SelectOptionsStructureError
	if !errors.As(err, &structErr) {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
	if structErr.Field != "DISTINCT" {
		t.Fatalf("Field: got %q, want %q", structErr.Field, "DISTINCT")
	}
}

func TestBuildEdgeSelect_StructureViolation_Fetch(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.Fetch(1, false)))
	_, err := buildEdgeSelect(leafSpec(e))
	var structErr *SelectOptionsStructureError
	if !errors.As(err, &structErr) {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
	if structErr.Field != "FETCH" {
		t.Fatalf("Field: got %q, want %q", structErr.Field, "FETCH")
	}
}

// TestComposeIsland_Route4OrderBy_UsesExplicitOrdinal covers ordering for
// route 4: a route 4 (Then boundary) parent query that requests an ORDER BY
// must not leave it as a literal clause and rely on composeIsland's _sl_src
// row_number() OVER () (no ORDER BY of its own) happening to preserve the
// planner's row order — an SQL property Postgres does not guarantee.
// buildEdgeSelect collapses
// that ORDER BY into an explicit ordinal column (same as routes 1/2), and
// composeIsland ORDERs BY it explicitly when told the column's name.
func TestComposeIsland_Route4OrderBy_UsesExplicitOrdinal(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.OrderBy("id").Desc()))
	childRoot := e.child()

	parentQuery, err := buildEdgeSelect(edgeSelectSpec{
		edge:            e,
		cols:            childRoot.cols(),
		alias:           srcAlias,
		from:            tableExpr(childRoot.schemaName(), childRoot.tableName()),
		keys:            [][]any{{1}, {2}, {3}},
		opts:            e.options(),
		collapseOrderBy: true,
	})
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}

	q, err := composeIsland(parentQuery, childRoot, islandPlan{}, ordCol)
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}

	if !strings.Contains(sql, `ORDER BY "q"."`+ordCol+`"`) {
		t.Fatalf("expected _sl_src's row_number() to ORDER BY %q explicitly; got:\n%s", ordCol, sql)
	}
	if strings.Contains(sql, "row_number() OVER ()") {
		t.Fatalf("must not fall back to an unordered row_number() OVER (); got:\n%s", sql)
	}
	// q.* would also expose ordCol itself in the outer projection (it is a
	// real SELECT column on parentQuery, not a window-only value) — the
	// root's own columns must be listed explicitly instead.
	if strings.Contains(sql, "q.*") {
		t.Fatalf("must not use q.* once an ordinal column is present (would leak %q); got:\n%s", ordCol, sql)
	}
}

// TestComposeIsland_Route4NoOrderBy_Unchanged is the companion negative
// case: when the edge requested no ORDER BY, passing an empty
// parentOrdCol must produce byte-identical SQL to before Important-2's
// fix — the plain, unordered row_number() OVER ().
func TestComposeIsland_Route4NoOrderBy_Unchanged(t *testing.T) {
	e := builderCapEdge() // no options
	childRoot := e.child()

	parentQuery, err := buildEdgeSelect(edgeSelectSpec{
		edge:            e,
		cols:            childRoot.cols(),
		alias:           srcAlias,
		from:            tableExpr(childRoot.schemaName(), childRoot.tableName()),
		keys:            [][]any{{1}, {2}, {3}},
		opts:            e.options(),
		collapseOrderBy: true,
	})
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}

	q, err := composeIsland(parentQuery, childRoot, islandPlan{}, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}

	if !strings.Contains(sql, "row_number() OVER ()") {
		t.Fatalf("expected the plain, unordered row_number() OVER () when no ordinal column is given; got:\n%s", sql)
	}
	if !strings.Contains(sql, "q.*") {
		t.Fatalf("expected q.* to still be used when no ordinal column is given; got:\n%s", sql)
	}
}

func TestHasOrderBy(t *testing.T) {
	if hasOrderBy(SelectOptions{}) {
		t.Fatal("zero-value SelectOptions must report no ORDER BY")
	}
	if hasOrderBy(Mods(sm.Where(psql.Quote("id").EQ(psql.Arg(1))))) {
		t.Fatal("a WHERE-only SelectOptions must report no ORDER BY")
	}
	if !hasOrderBy(Mods(sm.OrderBy("id"))) {
		t.Fatal("an ORDER BY SelectOptions must report ORDER BY present")
	}
}
