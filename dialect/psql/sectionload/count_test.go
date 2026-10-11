package sectionload

// Count Node — the count-only section type. These tests are DB-free
// white-box tests: hand-rolled
// CountEdge/TableParts/Buf fixtures, bob.Build to inspect rendered SQL,
// and the thenkey_test.go fake executor for scan/stitch behavior.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// --- fixtures -------------------------------------------------------

// countParent stands in for a generated bob model: Count is the
// *int64 field the generated PreloadCount would set on C.<Rel>.
type countParent struct {
	ID    int
	Count *int64
}

type countParentBuf struct{ id int }

func (b *countParentBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *countParentBuf) BuildFor(modelCols []Column) *countParent {
	return &countParent{ID: b.id}
}

type countChild struct {
	ID       int
	ParentID int
}

type countChildBuf struct{ id, parentID int }

func (b *countChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.parentID}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "parent_id":
			out[i] = &b.parentID
		default:
			panic("countChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *countChildBuf) BuildFor(modelCols []Column) *countChild {
	return &countChild{ID: b.id, ParentID: b.parentID}
}

func countEdge() CountEdge[*countParent, *countChild] {
	return CountEdge[*countParent, *countChild]{
		RelName:  "Children",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*countChild]{
			Table: "count_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*countChild] { return &countChildBuf{} },
		},
		ParentKey: func(p *countParent) []any { return []any{p.ID} },
		ChildKey:  func(c *countChild) []any { return []any{c.ParentID} },
		SetCount: func(p *countParent, n int64) error {
			p.Count = &n
			return nil
		},
	}
}

// countParts is the plain TableParts (the form the public API All/Build
// take); countRoot wraps it as the partsSpec view internal helpers take.
func countParts() TableParts[*countParent] {
	return TableParts[*countParent]{
		Table:   "count_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[*countParent] { return &countParentBuf{} },
	}
}

func countRoot() partsAdapter[*countParent] {
	return partsAdapter[*countParent]{countParts()}
}

// --- planning -------------------------------------------------------

// TestPlanIsland_CountNodeIsNotASection: a count node must land in
// plan.counts, never in plan.sections — the whole point of a separate node
// kind is that the row-building path (unionBranch/stitchInto) never sees
// it.
func TestPlanIsland_CountNodeIsNotASection(t *testing.T) {
	plan, err := planIsland([]Node{NewCountNode(countEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 0 {
		t.Fatalf("a count node must not produce a row section; got %+v", plan.sections)
	}
	if len(plan.counts) != 1 {
		t.Fatalf("counts: got %d, want 1", len(plan.counts))
	}
	c := plan.counts[0]
	if c.set != 1 || c.parentSet != 0 {
		t.Fatalf("count set/parentSet: got %d/%d, want 1/0", c.set, c.parentSet)
	}
	if c.countCol.Name != "_sl_cnt1" || c.countCol.DBType != "bigint" {
		t.Fatalf("countCol: got %+v, want {_sl_cnt1 bigint}", c.countCol)
	}
	wantKey := []Column{{Name: "parent_id", DBType: "int4"}}
	if len(c.keyCols) != 1 || c.keyCols[0] != wantKey[0] {
		t.Fatalf("keyCols: got %+v, want %+v", c.keyCols, wantKey)
	}
}

// TestPlanIsland_CountSetsDoNotCollideWithSections: count sets are drawn
// from the same counter as section sets, so _sl_cnt<set> can never name
// the same number as an existing section's _sl_n<set> CTE.
func TestPlanIsland_CountSetsDoNotCollideWithSections(t *testing.T) {
	tree := []Node{
		NewNode(builderCapEdge(), NewCountNode(countEdge())),
		NewCountNode(countEdge()),
	}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	seen := map[int]string{}
	for _, s := range plan.sections {
		if prev, dup := seen[s.set]; dup {
			t.Fatalf("set %d used twice (%s and section)", s.set, prev)
		}
		seen[s.set] = "section"
	}
	for _, c := range plan.counts {
		if prev, dup := seen[c.set]; dup {
			t.Fatalf("set %d used twice (%s and count)", c.set, prev)
		}
		seen[c.set] = "count"
	}
	if len(plan.sections) != 1 || len(plan.counts) != 2 {
		t.Fatalf("sections/counts: got %d/%d, want 1/2", len(plan.sections), len(plan.counts))
	}
	// A count child must still mark its parent section hasKids: the count
	// branch semi-joins against that section's own CTE (_sl_n<set>), which
	// composeIsland only materializes for hasKids nodes.
	if !plan.sections[0].hasKids {
		t.Fatal("a section with a count child must be hasKids (its CTE is the count branch's semi-join source)")
	}
}

// TestPlanIsland_ThenOnCountNodeIsRejected: Then(...) marks a query
// boundary, and a count node can never be one (it has no subtree to load).
// Silently ignoring the marker would leave the caller believing an extra
// round trip was requested; planning it as a Then boundary would hand
// runThen a nil edge. Reject it explicitly instead.
func TestPlanIsland_ThenOnCountNodeIsRejected(t *testing.T) {
	_, err := planIsland([]Node{Then(NewCountNode(countEdge()))})
	if err == nil {
		t.Fatal("expected an error for Then(NewCountNode(...)), got nil")
	}
	var cerr *CountNodeThenError
	if !errors.As(err, &cerr) {
		t.Fatalf("expected *CountNodeThenError, got %T: %v", err, err)
	}
	if cerr.RelName != "Children" {
		t.Fatalf("RelName: got %q, want %q", cerr.RelName, "Children")
	}
}

// TestPlanIsland_CountAndThenBoundarySetsDoNotCollide: the next-counter
// planIsland uses for section sets, count sets, and thenPlan ids is shared
// across all three, so mixing a count node and a Then(...) boundary in the
// same plan must still produce three distinct, non-colliding identifiers.
func TestPlanIsland_CountAndThenBoundarySetsDoNotCollide(t *testing.T) {
	tree := []Node{
		NewNode(builderCapEdge(), Then(NewNode(thenKeyMidLeafEdge()))),
		NewCountNode(countEdge()),
	}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 || len(plan.counts) != 1 || len(plan.thens) != 1 {
		t.Fatalf("sections/counts/thens: got %d/%d/%d, want 1/1/1", len(plan.sections), len(plan.counts), len(plan.thens))
	}
	seen := map[int]string{}
	for _, s := range plan.sections {
		seen[s.set] = "section"
	}
	for _, c := range plan.counts {
		if prev, dup := seen[c.set]; dup {
			t.Fatalf("count set %d collides with %s", c.set, prev)
		}
		seen[c.set] = "count"
	}
	for _, tp := range plan.thens {
		if prev, dup := seen[tp.id]; dup {
			t.Fatalf("thenPlan id %d collides with %s", tp.id, prev)
		}
		seen[tp.id] = "then"
	}
}

// TestPlanIsland_CountUnknownToColumnIsError: countKeyCols resolves
// CountEdge.ToCols against the child table's declared columns the same
// way NewColumnPlan resolves its own keyCols; a name that matches neither
// is a generator bug, reported the same way.
func TestPlanIsland_CountUnknownToColumnIsError(t *testing.T) {
	e := countEdge()
	e.ToCols = []string{"nonexistent_column"}
	_, err := planIsland([]Node{NewCountNode(e)})
	if err == nil {
		t.Fatal("expected an error for an unknown ToCols entry, got nil")
	}
	var uerr *UnknownKeyColumnError
	if !errors.As(err, &uerr) {
		t.Fatalf("expected *UnknownKeyColumnError, got %T: %v", err, err)
	}
	if uerr.Column != "nonexistent_column" {
		t.Fatalf("Column: got %q, want %q", uerr.Column, "nonexistent_column")
	}
}

// --- SQL composition -------------------------------------------------

// TestComposeIsland_CountBranchShape: the count section must be one more
// UNION ALL arm of the SAME island (A-04-1), selecting nothing but its
// join key and count(*), grouped by that key, and semi-joined against the
// parent CTE exactly like a row section is.
func TestComposeIsland_CountBranchShape(t *testing.T) {
	plan, err := planIsland([]Node{NewCountNode(countEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM count_parents")

	q, err := composeIsland(parent, countRoot(), plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s", sql)

	for _, frag := range []string{
		"count(*)",       // the aggregate itself
		"_sl_cnt1",       // the layout's count column (A-04-2)
		"GROUP BY",       // one row per parent key
		"UNION ALL",      // same island, not a separate query (A-04-1)
		`"parent_id"`,    // the grouped join key
		"count_children", // read straight from the child table
		"NULL::bigint",   // padding of _sl_cnt1 in the other branches
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("SQL missing expected fragment %q:\n%s", frag, sql)
		}
	}
	if n := strings.Count(sql, "UNION ALL"); n != 1 {
		t.Fatalf("want exactly one UNION ALL arm (one count section); got %d:\n%s", n, sql)
	}
	// A count section builds no models, so its child table's non-key
	// columns must never be selected.
	if strings.Contains(sql, `"_sl_t"."id"`) {
		t.Fatalf("a count section must select only its join key and count(*):\n%s", sql)
	}
}

// TestExpectedColumns_CountSlotsComeLast pins the fixed layout: counts
// occupy len(keyCols)+1 columns after every row section. execIsland validates the result columns positionally against this,
// so section0List and expectedColumns must agree exactly.
func TestExpectedColumns_CountSlotsComeLast(t *testing.T) {
	plan, err := planIsland([]Node{
		NewNode(builderCapEdge()),
		NewCountNode(countEdge()),
	})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	got := expectedColumns(countRoot(), plan)
	want := []string{
		"_sl_set", "_sl_ord",
		"id",              // root
		"id", "parent_id", // builderCapEdge's child section
		"parent_id", "_sl_cnt2", // the count section
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expectedColumns = %v, want %v", got, want)
	}

	list := section0List(countRoot(), plan)
	for _, frag := range []string{`NULL::int4 AS "parent_id"`, `NULL::bigint AS "_sl_cnt2"`} {
		if !strings.Contains(list, frag) {
			t.Fatalf("section0List must NULL-pad the count slots; missing %q in:\n%s", frag, list)
		}
	}
}

// TestComposeIsland_CountOrderByLimitWindow: ORDER BY and LIMIT pass
// through into the count's own aggregate subquery verbatim, the same way
// bob's LoadCount<Rel> appends caller mods to its aggregate SELECT. Alone
// an ORDER BY cannot change any count, but combined with LIMIT it decides
// WHICH parent-key groups are counted, so rejecting it would forbid the
// deterministic-window idiom bob supports. The existing collapseOrderBy
// machinery (ORDER BY → row_number() ordinal) must NOT kick in here: no
// row order needs to survive into stitching for a count.
func TestComposeIsland_CountOrderByLimitWindow(t *testing.T) {
	e := countEdge()
	e.Options = Mods(sm.OrderBy("parent_id"), sm.Limit(2))
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM count_parents")

	q, err := composeIsland(parent, countRoot(), plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	for _, frag := range []string{"ORDER BY", "LIMIT", "GROUP BY"} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("SQL missing %q — count options must pass through verbatim:\n%s", frag, sql)
		}
	}
	if strings.Contains(sql, ordCol) {
		t.Fatalf("a count branch must not grow a %s ordinal column (collapseOrderBy is a row-section mechanism):\n%s", ordCol, sql)
	}
}

// TestComposeIsland_CountWhereOptionApplied: a WHERE mod narrows the rows
// counted and goes through the same structure check every row section's
// options do (buildEdgeSelect), so the semi-join sectionload itself built
// must survive alongside it.
func TestComposeIsland_CountWhereOptionApplied(t *testing.T) {
	e := countEdge()
	e.Options = Mods(sm.Where(psql.Quote(srcAlias, "id").GT(psql.Arg(5))))
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM count_parents")

	q, err := composeIsland(parent, countRoot(), plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	if len(args) != 1 || args[0] != 5 {
		t.Fatalf("args: got %v, want [5]", args)
	}
	if !strings.Contains(sql, "GROUP BY") || !strings.Contains(sql, "IN") {
		t.Fatalf("the WHERE option must be ANDed alongside the semi-join, not replace it:\n%s", sql)
	}
}

// TestComposeIsland_CountStructureViolationRejected: the count section is
// built by buildEdgeSelect, so a mod that rewrites the SELECT list is
// caught by the same pre-flight structure check row sections get — a count layout slot is exactly as position-sensitive.
func TestComposeIsland_CountStructureViolationRejected(t *testing.T) {
	e := countEdge()
	e.Options = Mods(sm.Columns(psql.Quote(srcAlias, "id")))
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM count_parents")

	_, err = composeIsland(parent, countRoot(), plan, "")
	var structErr *SelectOptionsStructureError
	if !errors.As(err, &structErr) {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
}

// TestComposeIsland_MultipleCountLayoutStaysAligned pins the ORDER of the
// count slots across every function that emits them: with two count
// sections in the same island, section0List, unionBranch, countBranch,
// composeIsland and checkWidth must all place them in plan.counts order,
// after every row section. A regression in any single one of them would
// misalign execIsland's positional layout check.
func TestComposeIsland_MultipleCountLayoutStaysAligned(t *testing.T) {
	first := countEdge()
	first.RelName = "FirstCount"
	second := countEdge()
	second.RelName = "SecondCount"

	plan, err := planIsland([]Node{
		NewNode(builderCapEdge()),
		NewCountNode(first),
		NewCountNode(second),
	})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	root := countRoot()
	wantColumns := []string{
		colSet, colOrd,
		"id",
		"id", "parent_id",
		"parent_id", "_sl_cnt2",
		"parent_id", "_sl_cnt3",
	}
	if got := expectedColumns(root, plan); !reflect.DeepEqual(got, wantColumns) {
		t.Fatalf("expectedColumns = %v, want %v", got, wantColumns)
	}

	assertCountSlotOrder := func(name, sql string) {
		t.Helper()
		cnt2 := strings.Index(sql, `AS "_sl_cnt2"`)
		cnt3 := strings.Index(sql, `AS "_sl_cnt3"`)
		if cnt2 < 0 || cnt3 < 0 || cnt2 >= cnt3 {
			t.Fatalf("%s count slots are not in plan.counts order:\n%s", name, sql)
		}
	}

	buildSQL := func(name string, q bob.Query) string {
		t.Helper()
		sql, _, err := bob.Build(context.Background(), q)
		if err != nil {
			t.Fatalf("%s: bob.Build: %v", name, err)
		}
		return sql
	}

	assertCountSlotOrder("section0List", section0List(root, plan))

	rowBranch, err := unionBranch(root, plan, plan.sections[0], false)
	if err != nil {
		t.Fatalf("unionBranch: %v", err)
	}
	assertCountSlotOrder("unionBranch", buildSQL("unionBranch", rowBranch))

	for _, cp := range plan.counts {
		branch, err := countBranch(root, plan, cp)
		if err != nil {
			t.Fatalf("countBranch(set=%d): %v", cp.set, err)
		}
		assertCountSlotOrder(
			fmt.Sprintf("countBranch(set=%d)", cp.set),
			buildSQL("countBranch", branch),
		)
	}

	parent := psql.RawQuery("SELECT id FROM count_parents")
	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	fullSQL := buildSQL("composeIsland", q)
	set1 := strings.Index(fullSQL, `1 AS "_sl_set"`)
	set2 := strings.Index(fullSQL, `2 AS "_sl_set"`)
	set3 := strings.Index(fullSQL, `3 AS "_sl_set"`)
	if set1 < 0 || set2 < 0 || set3 < 0 || set1 >= set2 || set2 >= set3 {
		t.Fatalf("UNION ALL branches are not in section/count order:\n%s", fullSQL)
	}

	// Base 3 + row section 2 + first count 2 = 7.
	// The second count crosses the limit at 9.
	err = checkWidth(root, plan, 7)
	var wide *TooWideError
	if !errors.As(err, &wide) {
		t.Fatalf("checkWidth: got %T: %v, want *TooWideError", err, err)
	}
	if wide.Total != 9 || wide.Max != 7 || wide.AtRel != "SecondCount" {
		t.Fatalf("TooWideError = %+v, want Total=9 Max=7 AtRel=SecondCount", wide)
	}
}

// TestReservedName_CountColumn closes the gap found by a
// source-tree verification:
// _sl_cnt<set> is a real result column of the composed island (unlike
// _sl_row_ord, which only exists inside an edge's own subquery), so a
// parent query using that identifier must be refused like _sl_set is.
func TestReservedName_CountColumn(t *testing.T) {
	if got := reservedNameRe.FindString(`SELECT 1 AS _sl_cnt1 FROM t`); got != "_sl_cnt1" {
		t.Fatalf("reservedNameRe must match _sl_cnt1; got %q", got)
	}
	if got := reservedNameRe.FindString(`SELECT count_1 AS counted FROM t`); got != "" {
		t.Fatalf("reservedNameRe must not match unrelated identifiers; got %q", got)
	}
}

// --- scanning --------------------------------------------------------

// countRows builds the canned result set of a one-count island:
//
//	layout: _sl_set, _sl_ord, root.id, count.parent_id, _sl_cnt1
//
// three parents (10, 20, 30) and count rows for only two of them, so the
// third exercises the "no row came back" path stitchCounts must fill.
func countRows(cols []string) *thenKeyColsRows {
	return &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			{int64(0), int64(1), 10, 0, nil},
			{int64(0), int64(2), 20, 0, nil},
			{int64(0), int64(3), 30, 0, nil},
			{int64(1), nil, 0, 10, int64(2)},
			{int64(1), nil, 0, 20, int64(1)},
		}},
	}
}

// TestExecIsland_CountRowsLandInTheirOwnLane: count rows must never reach
// items (no model is built for them) and must be keyed by the value the
// edge's own ChildKey extracts, so stitchCounts can look each parent up.
func TestExecIsland_CountRowsLandInTheirOwnLane(t *testing.T) {
	plan, err := planIsland([]Node{NewCountNode(countEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := countRoot()
	exec := thenKeyFakeExecutor{rows: countRows(expectedColumns(root, plan))}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}

	if len(res.items[0]) != 3 {
		t.Fatalf("root items: got %d, want 3", len(res.items[0]))
	}
	set := plan.counts[0].set
	got := res.counts[set]
	if len(got) != 2 {
		t.Fatalf("counts[%d]: got %v, want 2 entries (parents 10 and 20)", set, got)
	}
	if got[10] != 2 || got[20] != 1 {
		t.Fatalf("counts[%d] = %v, want map[10:2 20:1]", set, got)
	}
	if _, present := got[30]; present {
		t.Fatalf("counts[%d] must hold only the keys the query returned; 0-filling is stitchCounts's job: %v", set, got)
	}

	// The parent keys the count edge extracts must be collected for EVERY
	// parent row, aligned with items[0] — that is what lets stitchCounts
	// give parent 30 a zero.
	keys := res.parentKeys[set]
	if len(keys) != 3 {
		t.Fatalf("parentKeys[%d]: got %d entries, want 3", set, len(keys))
	}
	for i, want := range []int{10, 20, 30} {
		kv, ok := keys[i].([]any)
		if !ok || len(kv) != 1 || kv[0] != want {
			t.Fatalf("parentKeys[%d][%d] = %#v, want []any{%d}", set, i, keys[i], want)
		}
	}
}

// rowChild/rowChildBuf/rowEdge: a row-loading (non-count) edge
// fixture for the sparse-set-numbers regression test below. It cannot
// reuse builderCapEdge — that fixture's Parent type is the unrelated
// builderCapParent struct, whereas the root used here is countRoot
// (Parent *countParent). execIsland extracts a row section's parent
// key straight from the ISLAND ROOT's own scan buffer
// (child.edge.parentKeyFromBufAny(rootBuf) for children of set 0), and
// Edge[P, C].parentKeyFromBufAny type-asserts that buffer to
// bufAdapter[P] — mixing an unrelated Parent type panics on the type
// assertion rather than exercising the sparse-numbering bug this test is
// for, so the row edge here shares countParent as its Parent type.
type rowChild struct {
	ID       int
	ParentID int
}

type rowChildBuf struct{ id, parentID int }

func (b *rowChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.parentID}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "parent_id":
			out[i] = &b.parentID
		default:
			panic("rowChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *rowChildBuf) BuildFor(modelCols []Column) *rowChild {
	return &rowChild{ID: b.id, ParentID: b.parentID}
}

func rowEdge() Edge[*countParent, *rowChild] {
	return Edge[*countParent, *rowChild]{
		RelName:  "Rows",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Child: TableParts[*rowChild]{
			Table: "row_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*rowChild] { return &rowChildBuf{} },
		},
		ParentKey: func(p *countParent) []any { return []any{p.ID} },
		ChildKey:  func(c *rowChild) []any { return []any{c.ParentID} },
		Stitch:    func(p *countParent, cs []*rowChild) error { return nil },
	}
}

// TestExecIsland_SparseSetNumbersDoNotPanic: planIsland's next-counter is
// shared across row sections and count sections, interleaved in
// declaration order — declaring a count node before a row node gives the
// count section set=1 and the row section set=2. res.items/res.secOrds
// must be sized off the HIGHEST set number in use (via resultSetSlots),
// not off len(plan.sections)+1, or indexing res.items[2] panics with index
// out of range (found in review).
func TestExecIsland_SparseSetNumbersDoNotPanic(t *testing.T) {
	rowEdge := rowEdge()
	countEdge := countEdge()

	// Declaration order matters: count first gives it set=1, pushing the
	// row section to set=2.
	plan, err := planIsland([]Node{
		NewCountNode(countEdge),
		NewNode(rowEdge),
	})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.counts) != 1 || plan.counts[0].set != 1 {
		t.Fatalf("count set: got %+v, want set=1", plan.counts)
	}
	if len(plan.sections) != 1 || plan.sections[0].set != 2 {
		t.Fatalf("row section set: got %+v, want set=2", plan.sections)
	}

	root := countRoot()
	cols := expectedColumns(root, plan)
	exec := thenKeyFakeExecutor{rows: &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id, rowSection.id, rowSection.parent_id, count.parent_id, _sl_cnt1
			{int64(0), int64(1), 10, 0, 0, 0, nil},
			{int64(2), nil, 0, 100, 10, 0, nil},
			{int64(1), nil, 0, 0, 0, 10, int64(2)},
		}},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[0]) != 1 {
		t.Fatalf("root items: got %d, want 1", len(res.items[0]))
	}
	if len(res.items[2]) != 1 {
		t.Fatalf("row section items (set 2): got %d, want 1", len(res.items[2]))
	}
	if got := res.counts[1][10]; got != 2 {
		t.Fatalf("count (set 1) for key 10: got %d, want 2", got)
	}
}

// --- stitching -------------------------------------------------------

// TestStitchCounts_EveryParentGetsANonNilValue pins the heart of the count
// contract: bob's own LoadCount<Rel> assigns &count to EVERY parent, so a
// parent with no matching children ends up with a pointer to 0, never
// nil. nil means "not loaded"; &0 means "loaded, zero children".
func TestStitchCounts_EveryParentGetsANonNilValue(t *testing.T) {
	plan, err := planIsland([]Node{NewCountNode(countEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := countRoot()
	exec := thenKeyFakeExecutor{rows: countRows(expectedColumns(root, plan))}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if err := stitchIsland(plan, res); err != nil {
		t.Fatalf("stitchIsland: %v", err)
	}

	want := map[int]int64{10: 2, 20: 1, 30: 0}
	for _, it := range res.items[0] {
		p := it.(*countParent)
		if p.Count == nil {
			t.Fatalf("parent %d: Count is nil; every parent must get a non-nil value (bob's &0 contract)", p.ID)
		}
		if *p.Count != want[p.ID] {
			t.Fatalf("parent %d: Count = %d, want %d", p.ID, *p.Count, want[p.ID])
		}
	}
}

// TestStitchCounts_NullParentKeyStillGetsZero: a parent whose nullable
// join key is not set matches no count row. stitchInto skips such a parent
// for ROWS (leaving R untouched), but bob's LoadCount<Rel> assigns to every
// parent unconditionally, so counts must not skip it.
func TestStitchCounts_NullParentKeyStillGetsZero(t *testing.T) {
	e := countEdge()
	e.ParentKey = func(p *countParent) []any { return nil } // key never set
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := countRoot()
	exec := thenKeyFakeExecutor{rows: countRows(expectedColumns(root, plan))}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if err := stitchIsland(plan, res); err != nil {
		t.Fatalf("stitchIsland: %v", err)
	}
	for _, it := range res.items[0] {
		p := it.(*countParent)
		if p.Count == nil || *p.Count != 0 {
			t.Fatalf("parent %d with an unset join key: Count = %v, want a non-nil 0", p.ID, p.Count)
		}
	}
}

// TestAll_SetCountErrorIsWrapped: the generated SetCount reports a
// stale generated contract (A-04-4) by returning an error; it must reach
// the caller of the public All() entry point, not be swallowed anywhere
// along the way (execIsland's scan, stitchIsland's stitching, or All's own
// plumbing).
func TestAll_SetCountErrorIsWrapped(t *testing.T) {
	e := countEdge()
	e.SetCount = func(p *countParent, n int64) error {
		return &StaleCountContractError{RelName: "Children"}
	}
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := countRoot()
	exec := thenKeyFakeExecutor{rows: countRows(expectedColumns(root, plan))}

	_, err = All(
		context.Background(),
		exec,
		psql.RawQuery("SELECT id FROM count_parents"),
		countParts(),
		NewCountNode(e),
	)
	if err == nil {
		t.Fatal("expected SetCount's error to propagate, got nil")
	}
	var stale *StaleCountContractError
	if !errors.As(err, &stale) {
		t.Fatalf("expected a wrapped *StaleCountContractError, got %T: %v", err, err)
	}
}

// --- public API ------------------------------------------------------

// TestAll_CountNodeEndToEnd drives the whole path through the public API
// (hooks → reserved-name check → plan → compose → exec → stitch → order
// restore) with the canned executor, so nothing but the database is faked.
func TestAll_CountNodeEndToEnd(t *testing.T) {
	e := countEdge()
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	exec := &queueExecutor{queue: []scan.Rows{
		countRows(expectedColumns(countRoot(), plan)),
	}}

	out, err := All(
		context.Background(), exec,
		psql.RawQuery("SELECT id FROM count_parents"),
		countParts(),
		NewCountNode(e),
	)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(out) != 3 {
		t.Fatalf("parents: got %d, want 3", len(out))
	}
	if exec.i != 1 {
		t.Fatalf("queries issued: got %d, want 1", exec.i)
	}
	requireCountSQL(t, exec.queries[0])
	want := map[int]int64{10: 2, 20: 1, 30: 0}
	for _, p := range out {
		if p.Count == nil {
			t.Fatalf("parent %d: Count is nil", p.ID)
		}
		if *p.Count != want[p.ID] {
			t.Fatalf("parent %d: Count = %d, want %d", p.ID, *p.Count, want[p.ID])
		}
	}
	// Parent order must still come from the parent query's ordinality.
	for i, wantID := range []int{10, 20, 30} {
		if out[i].ID != wantID {
			t.Fatalf("parent order: got %d at index %d, want %d", out[i].ID, i, wantID)
		}
	}
}

// TestAll_CountBeforeRowSection_ResultSetSlotsSufficient drives the same
// declaration order as TestExecIsland_SparseSetNumbersDoNotPanic (a count
// node declared before a row section, so the count gets set=1 and the row
// section is pushed to set=2) through the full public All() entry point
// instead of stopping at execIsland/stitchIsland. All() adds its own
// layers on top (validatePlan, parent-query hooks, Build/composeIsland's
// own wiring) that the lower-level test never exercises, so a regression
// in resultSetSlots's max-based sizing could in principle
// resurface there even with the lower-level test passing.
func TestAll_CountBeforeRowSection_ResultSetSlotsSufficient(t *testing.T) {
	countEdge := countEdge()
	rowEdge := rowEdge()

	tree := []Node{
		NewCountNode(countEdge), // set=1 (count)
		NewNode(rowEdge),        // set=2 (row section)
	}
	plan, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.counts) != 1 || plan.counts[0].set != 1 {
		t.Fatalf("count set: got %+v, want set=1", plan.counts)
	}
	if len(plan.sections) != 1 || plan.sections[0].set != 2 {
		t.Fatalf("row section set: got %+v, want set=2", plan.sections)
	}
	if slots, want := resultSetSlots(plan), plan.sections[0].set+1; slots < want {
		t.Fatalf("resultSetSlots: got %d, want >= %d (must size off the highest set, not len(plan.sections)+1)", slots, want)
	}

	root := countRoot()
	exec := &queueExecutor{queue: []scan.Rows{&thenKeyColsRows{
		cols: expectedColumns(root, plan),
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id, rowSection.id, rowSection.parent_id, count.parent_id, _sl_cnt1
			{int64(0), int64(1), 10, 0, 0, 0, nil},
			{int64(2), nil, 0, 100, 10, 0, nil},
			{int64(1), nil, 0, 0, 0, 10, int64(2)},
		}},
	}}}

	// All() must reach stitchIsland without an index-out-of-range panic —
	// the failure mode resultSetSlots exists to prevent.
	out, err := All(
		context.Background(), exec,
		psql.RawQuery("SELECT id FROM count_parents"),
		countParts(),
		tree...,
	)
	if err != nil {
		t.Fatalf("All (count-before-section): %v", err)
	}
	if len(out) != 1 || out[0].ID != 10 {
		t.Fatalf("parents: got %+v, want one parent with ID=10", out)
	}
	if out[0].Count == nil || *out[0].Count != 2 {
		t.Fatalf("parent 10's Count: got %v, want *2", out[0].Count)
	}
}

// TestBuild_CountNodeSQL: Build() renders the first island without running
// it, so the count column must already be visible there — this is the
// observability entry point a caller uses to check the single-query claim.
func TestBuild_CountNodeSQL(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM count_parents"),
		countParts(),
		NewCountNode(countEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	for _, frag := range []string{"count(*)", "_sl_cnt1", "UNION ALL"} {
		if !strings.Contains(built.SQL, frag) {
			t.Fatalf("Build SQL missing %q:\n%s", frag, built.SQL)
		}
	}
}

// requireCountSQL asserts that query is a count island's rendered SQL:
// the aggregate itself, the fixed layout's count column, and the UNION ALL
// arm that carries it. Shared by tests that need to check the SQL actually
// issued to a fake executor, not just the SQL Build() renders in isolation.
func requireCountSQL(t *testing.T, query string) {
	t.Helper()
	for _, frag := range []string{"count(*)", "_sl_cnt1", "UNION ALL"} {
		if !strings.Contains(query, frag) {
			t.Fatalf("count SQL missing %q:\n%s", frag, query)
		}
	}
}

// TestAll_CountInsideThenIslandRuns: a count declared inside a Then(...)
// subtree belongs to the SECOND island, and runThen must plan/compose/
// exec/stitch it there just like the top-level island does. This drives
// the whole thing through the public API with a two-query fake executor
// (query 1 = the main island, query 2 = the Then island) and asserts the
// generated-style SetCount callback actually received the right value —
// if runThen ever stops passing childPlan/res through to stitchIsland's
// count lane, `recorded` stays empty and this fails.
func TestAll_CountInsideThenIslandRuns(t *testing.T) {
	recorded := map[int]int64{}
	midEdge := thenKeyRootMidEdge()
	countEdge := midLeafCountEdge(recorded)

	rootParts := TableParts[thenKeyRoot]{
		Table:   "tk_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[thenKeyRoot] { return &thenKeyRootBuf{} },
	}

	// Island 1: the count lives inside the Then subtree, so the FIRST
	// island must plan no count of its own.
	tree := []Node{Then(NewNode(midEdge, NewCountNode(countEdge)))}
	plan1, err := planIsland(tree)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan1.thens) != 1 || len(plan1.counts) != 0 || len(plan1.sections) != 0 {
		t.Fatalf("the count must belong to the Then island, not this one: %+v", plan1)
	}
	cols1 := expectedColumns(partsAdapter[thenKeyRoot]{rootParts}, plan1)

	// Island 2 (the Then island): mid becomes the root, the count is a
	// section of THAT island. Recomputing the plan here is deterministic —
	// runThen will build the identical one from plan1.thens[0].children.
	subPlan, err := planIsland(plan1.thens[0].children)
	if err != nil {
		t.Fatalf("planIsland(then children): %v", err)
	}
	if len(subPlan.counts) != 1 || subPlan.counts[0].parentSet != 0 {
		t.Fatalf("the Then island must plan the count against its own root: %+v", subPlan.counts)
	}
	cols2 := expectedColumns(edgeRootParts{midEdge.child(), midEdge}, subPlan)

	exec := &queueExecutor{queue: []scan.Rows{
		&thenKeyColsRows{cols: cols1, thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id
			{int64(0), int64(1), 100},
		}}},
		&thenKeyColsRows{cols: cols2, thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, mid.id, mid.root_id, mid.other_key,
			// count key (other_key), _sl_cnt1
			{int64(0), int64(1), 200, 100, 999, 0, nil},
			{int64(1), nil, 0, 0, 0, 999, int64(3)},
		}}},
	}}

	out, err := All(
		context.Background(), exec,
		psql.RawQuery("SELECT id FROM tk_root"),
		rootParts,
		Then(NewNode(midEdge, NewCountNode(countEdge))),
	)
	if err != nil {
		t.Fatalf("All: %v", err)
	}
	if len(out) != 1 || out[0].ID != 100 {
		t.Fatalf("parents: got %+v, want exactly one root with ID 100", out)
	}
	if exec.i != 2 {
		t.Fatalf("queries issued: got %d, want 2 (main island + Then island)", exec.i)
	}
	if strings.Contains(exec.queries[0], "count(*)") ||
		strings.Contains(exec.queries[0], "_sl_cnt") {
		t.Fatalf(
			"the main island must not contain the Then island's count:\n%s",
			exec.queries[0],
		)
	}
	requireCountSQL(t, exec.queries[1])
	if len(recorded) != 1 || recorded[200] != 3 {
		t.Fatalf("recorded counts = %v, want map[200:3] (mid 200 counted 3 leaves)", recorded)
	}
}

// queueExecutor hands out one canned Rows per query, in order. An All()
// call may issue several queries (the main island, then one per Then
// boundary), each with its own column layout, so a single-rows fake
// (thenKeyFakeExecutor) cannot serve them all.
type queueExecutor struct {
	queue   []scan.Rows
	queries []string
	i       int
}

func (e *queueExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	if e.i >= len(e.queue) {
		return nil, fmt.Errorf("queueExecutor: unexpected query #%d: %s", e.i+1, query)
	}
	r := e.queue[e.i]
	e.queries = append(e.queries, query)
	e.i++
	return r, nil
}

func (e *queueExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("queueExecutor: ExecContext not implemented (not needed by execIsland)")
}

var _ bob.Executor = (*queueExecutor)(nil)

// leafCountBuf scans exactly the columns it is asked for. A count
// section's buffer is only ever asked for the edge's key columns, so a
// fixture that ignores cols (like thenKeyLeafBuf, which always returns both
// of its targets) would register one scan target too many and misalign
// the whole row.
type leafCountBuf struct{ id, otherKey int }

func (b *leafCountBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.otherKey}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "other_key":
			out[i] = &b.otherKey
		default:
			panic("leafCountBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *leafCountBuf) BuildFor(modelCols []Column) thenKeyLeaf {
	return thenKeyLeaf{ID: b.id, OtherKey: b.otherKey}
}

// midLeafCountEdge counts thenKeyLeaf rows per thenKeyMid, reusing the thenKey
// model types so a Then boundary can be built without new table types.
// SetCount records into the given map so tests can observe stitching.
func midLeafCountEdge(record map[int]int64) CountEdge[thenKeyMid, thenKeyLeaf] {
	return CountEdge[thenKeyMid, thenKeyLeaf]{
		RelName:  "LeafCount",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		Child: TableParts[thenKeyLeaf]{
			Table: "tk_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[thenKeyLeaf] { return &leafCountBuf{} },
		},
		ParentKey: func(p thenKeyMid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c thenKeyLeaf) []any { return []any{c.OtherKey} },
		SetCount: func(p thenKeyMid, n int64) error {
			record[p.ID] = n
			return nil
		},
	}
}

// --- validatePlan × count node ---
//
// validatePlan's check of a ColumnPlan-restricted edge's children
// unconditionally read c.edge, which is nil for a count node child —
// before this fix, any tree nesting a count node under a restricted row
// edge panicked (nil pointer dereference) instead of returning the same
// static errors a restricted row child gets.

// restrictedMidEdge returns thenKeyRootMidEdge with a ColumnPlan that
// fetches only id (Model) + root_id and the given extra key columns
// (Internal), plus the ChildKeyFromBuf override validatePlan requires of
// any restricting edge.
func restrictedMidEdge(t *testing.T, extraKeyCols ...string) Edge[thenKeyRoot, thenKeyMid] {
	t.Helper()
	e := thenKeyRootMidEdge()
	all := e.Child.Columns
	keyCols := append([]string{"root_id"}, extraKeyCols...)
	plan, err := NewColumnPlan(all, []string{"id"}, nil, keyCols)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	e.ColumnPlan = plan
	e.ChildKeyFromBuf = func(b Buf[thenKeyMid]) []any {
		return []any{b.(*thenKeyMidBuf).rootID}
	}
	return e
}

// TestValidatePlan_CountChildUnderRestrictedEdge_RequiresParentKeyFromBuf:
// before this fix, validatePlan dereferenced c.edge (nil for a count
// node) and panicked; now it must return the same MissingKeyFromBufError
// a row child gets when the restricted parent's buffer would be read
// through the BuildFor(nil) fallback.
func TestValidatePlan_CountChildUnderRestrictedEdge_RequiresParentKeyFromBuf(t *testing.T) {
	parent := restrictedMidEdge(t, "other_key")
	cnt := midLeafCountEdge(map[int]int64{}) // ParentKeyFromBuf deliberately nil

	err := validatePlan([]Node{NewNode(parent, NewCountNode(cnt))}, nil)
	var want *MissingKeyFromBufError
	if !errors.As(err, &want) {
		t.Fatalf("expected MissingKeyFromBufError, got %v", err)
	}
	if want.RelName != "LeafCount" || want.Field != "ParentKeyFromBuf" || want.RestrictedRelName != "Mid" {
		t.Fatalf("error fields = %+v, want RelName=LeafCount Field=ParentKeyFromBuf RestrictedRelName=Mid", want)
	}
}

// TestValidatePlan_CountChildFromColsMustBeFetched: the count child's
// FromCols must be inside the restricted parent's fetched column set,
// exactly like a row child's.
func TestValidatePlan_CountChildFromColsMustBeFetched(t *testing.T) {
	parent := restrictedMidEdge(t) // other_key NOT fetched
	cnt := midLeafCountEdge(map[int]int64{})
	cnt.ParentKeyFromBuf = func(b Buf[thenKeyMid]) []any {
		return []any{b.(*thenKeyMidBuf).otherKey}
	}

	err := validatePlan([]Node{NewNode(parent, NewCountNode(cnt))}, nil)
	var want *MissingRequiredColumnError
	if !errors.As(err, &want) {
		t.Fatalf("expected MissingRequiredColumnError, got %v", err)
	}
	if want.RelName != "Mid" || want.ChildRelName != "LeafCount" || want.Column != "other_key" {
		t.Fatalf("error fields = %+v, want RelName=Mid ChildRelName=LeafCount Column=other_key", want)
	}
}

// TestValidatePlan_CountChildUnderRestrictedEdge_OK: with the override
// set and the key column fetched, the same tree passes.
func TestValidatePlan_CountChildUnderRestrictedEdge_OK(t *testing.T) {
	parent := restrictedMidEdge(t, "other_key")
	cnt := midLeafCountEdge(map[int]int64{})
	cnt.ParentKeyFromBuf = func(b Buf[thenKeyMid]) []any {
		return []any{b.(*thenKeyMidBuf).otherKey}
	}

	if err := validatePlan([]Node{NewNode(parent, NewCountNode(cnt))}, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
}
