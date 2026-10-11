package sectionload

// P-04: count Node — the count-only section type. These tests are DB-free
// white-box tests in the style established by P-02/P-09/P-11: hand-rolled
// CountEdge/TableParts/Buf fixtures, bob.Build to inspect rendered SQL,
// and the p28_thenkey_test.go fake executor for scan/stitch behavior.

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

// p04CountParent stands in for a generated bob model: Count is the
// *int64 field the generated PreloadCount would set on C.<Rel>.
type p04CountParent struct {
	ID    int
	Count *int64
}

type p04CountParentBuf struct{ id int }

func (b *p04CountParentBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *p04CountParentBuf) BuildFor(modelCols []Column) *p04CountParent {
	return &p04CountParent{ID: b.id}
}

type p04CountChild struct {
	ID       int
	ParentID int
}

type p04CountChildBuf struct{ id, parentID int }

func (b *p04CountChildBuf) TargetsFor(cols []Column) []any {
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
			panic("p04CountChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p04CountChildBuf) BuildFor(modelCols []Column) *p04CountChild {
	return &p04CountChild{ID: b.id, ParentID: b.parentID}
}

func p04CountEdge() CountEdge[*p04CountParent, *p04CountChild] {
	return CountEdge[*p04CountParent, *p04CountChild]{
		RelName:  "Children",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*p04CountChild]{
			Table: "p04_count_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*p04CountChild] { return &p04CountChildBuf{} },
		},
		ParentKey: func(p *p04CountParent) []any { return []any{p.ID} },
		ChildKey:  func(c *p04CountChild) []any { return []any{c.ParentID} },
		SetCount: func(p *p04CountParent, n int64) error {
			p.Count = &n
			return nil
		},
	}
}

// p04CountParts is the plain TableParts (the form the public API All/Build
// take); p04CountRoot wraps it as the partsSpec view internal helpers take.
func p04CountParts() TableParts[*p04CountParent] {
	return TableParts[*p04CountParent]{
		Table:   "p04_count_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[*p04CountParent] { return &p04CountParentBuf{} },
	}
}

func p04CountRoot() partsAdapter[*p04CountParent] {
	return partsAdapter[*p04CountParent]{p04CountParts()}
}

// --- planning -------------------------------------------------------

// TestPlanIsland_CountNodeIsNotASection: a count node must land in
// plan.counts, never in plan.sections — the whole point of a separate node
// kind is that the row-building path (unionBranch/stitchInto) never sees
// it (issue P-04 §10.6).
func TestPlanIsland_CountNodeIsNotASection(t *testing.T) {
	plan, err := planIsland([]Node{NewCountNode(p04CountEdge())})
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
		NewNode(p11CapEdge(), NewCountNode(p04CountEdge())),
		NewCountNode(p04CountEdge()),
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
	_, err := planIsland([]Node{Then(NewCountNode(p04CountEdge()))})
	if err == nil {
		t.Fatal("expected an error for Then(NewCountNode(...)), got nil")
	}
	cerr, ok := err.(*CountNodeThenError)
	if !ok {
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
		NewNode(p11CapEdge(), Then(NewNode(p28MidLeafEdge()))),
		NewCountNode(p04CountEdge()),
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
	e := p04CountEdge()
	e.ToCols = []string{"nonexistent_column"}
	_, err := planIsland([]Node{NewCountNode(e)})
	if err == nil {
		t.Fatal("expected an error for an unknown ToCols entry, got nil")
	}
	uerr, ok := err.(*UnknownKeyColumnError)
	if !ok {
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
	plan, err := planIsland([]Node{NewCountNode(p04CountEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p04_count_parents")

	q, err := composeIsland(parent, p04CountRoot(), plan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s", sql)

	for _, frag := range []string{
		"count(*)",           // the aggregate itself
		"_sl_cnt1",           // the layout's count column (A-04-2)
		"GROUP BY",           // one row per parent key
		"UNION ALL",          // same island, not a separate query (A-04-1)
		`"parent_id"`,        // the grouped join key
		"p04_count_children", // read straight from the child table
		"NULL::bigint",       // padding of _sl_cnt1 in the other branches
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
// occupy len(keyCols)+1 columns after every row section (issue P-04 §10.2,
// option A). execIsland validates the result columns positionally against this,
// so section0List and expectedColumns must agree exactly.
func TestExpectedColumns_CountSlotsComeLast(t *testing.T) {
	plan, err := planIsland([]Node{
		NewNode(p11CapEdge()),
		NewCountNode(p04CountEdge()),
	})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	got := expectedColumns(p04CountRoot(), plan)
	want := []string{
		"_sl_set", "_sl_ord",
		"id",              // root
		"id", "parent_id", // p11CapEdge's child section
		"parent_id", "_sl_cnt2", // the count section
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expectedColumns = %v, want %v", got, want)
	}

	list := section0List(p04CountRoot(), plan)
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
	e := p04CountEdge()
	e.Options = Mods(sm.OrderBy("parent_id"), sm.Limit(2))
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p04_count_parents")

	q, err := composeIsland(parent, p04CountRoot(), plan, DefaultMaxColumns, "")
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
	e := p04CountEdge()
	e.Options = Mods(sm.Where(psql.Quote(srcAlias, "id").GT(psql.Arg(5))))
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p04_count_parents")

	q, err := composeIsland(parent, p04CountRoot(), plan, DefaultMaxColumns, "")
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
// caught by the same pre-flight structure check row sections get (P-02
// §9 steps 7-8) — a count layout slot is exactly as position-sensitive.
func TestComposeIsland_CountStructureViolationRejected(t *testing.T) {
	e := p04CountEdge()
	e.Options = Mods(sm.Columns(psql.Quote(srcAlias, "id")))
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p04_count_parents")

	_, err = composeIsland(parent, p04CountRoot(), plan, DefaultMaxColumns, "")
	if _, ok := err.(*SelectOptionsStructureError); !ok {
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
	first := p04CountEdge()
	first.RelName = "FirstCount"
	second := p04CountEdge()
	second.RelName = "SecondCount"

	plan, err := planIsland([]Node{
		NewNode(p11CapEdge()),
		NewCountNode(first),
		NewCountNode(second),
	})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	root := p04CountRoot()
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

	parent := psql.RawQuery("SELECT id FROM p04_count_parents")
	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	fullSQL := buildSQL("composeIsland", q)
	set1 := strings.Index(fullSQL, `1 AS "_sl_set"`)
	set2 := strings.Index(fullSQL, `2 AS "_sl_set"`)
	set3 := strings.Index(fullSQL, `3 AS "_sl_set"`)
	if set1 < 0 || set2 < 0 || set3 < 0 || !(set1 < set2 && set2 < set3) {
		t.Fatalf("UNION ALL branches are not in section/count order:\n%s", fullSQL)
	}

	// Base 3 + row section 2 + first count 2 = 7.
	// The second count crosses the limit at 9.
	err = checkWidth(root, plan, 7)
	wide, ok := err.(*TooWideError)
	if !ok {
		t.Fatalf("checkWidth: got %T: %v, want *TooWideError", err, err)
	}
	if wide.Total != 9 || wide.Max != 7 || wide.AtRel != "SecondCount" {
		t.Fatalf("TooWideError = %+v, want Total=9 Max=7 AtRel=SecondCount", wide)
	}
}

// TestReservedName_CountColumn closes the gap found by the 2026-08-08
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

// p04CountRows builds the canned result set of a one-count island:
//
//	layout: _sl_set, _sl_ord, root.id, count.parent_id, _sl_cnt1
//
// three parents (10, 20, 30) and count rows for only two of them, so the
// third exercises the "no row came back" path stitchCounts must fill.
func p04CountRows(cols []string) *p28ColsRows {
	return &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
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
	plan, err := planIsland([]Node{NewCountNode(p04CountEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := p04CountRoot()
	exec := p28FakeExecutor{rows: p04CountRows(expectedColumns(root, plan))}

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

// p04RowChild/p04RowChildBuf/p04RowEdge: a row-loading (non-count) edge
// fixture for the sparse-set-numbers regression test below. It cannot
// reuse p11CapEdge — that fixture's Parent type is the unrelated
// p11CapParent struct, whereas the root used here is p04CountRoot
// (Parent *p04CountParent). execIsland extracts a row section's parent
// key straight from the ISLAND ROOT's own scan buffer
// (child.edge.parentKeyFromBufAny(rootBuf) for children of set 0), and
// Edge[P, C].parentKeyFromBufAny type-asserts that buffer to
// bufAdapter[P] — mixing an unrelated Parent type panics on the type
// assertion rather than exercising the sparse-numbering bug this test is
// for, so the row edge here shares p04CountParent as its Parent type.
type p04RowChild struct {
	ID       int
	ParentID int
}

type p04RowChildBuf struct{ id, parentID int }

func (b *p04RowChildBuf) TargetsFor(cols []Column) []any {
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
			panic("p04RowChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p04RowChildBuf) BuildFor(modelCols []Column) *p04RowChild {
	return &p04RowChild{ID: b.id, ParentID: b.parentID}
}

func p04RowEdge() Edge[*p04CountParent, *p04RowChild] {
	return Edge[*p04CountParent, *p04RowChild]{
		RelName:  "Rows",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Child: TableParts[*p04RowChild]{
			Table: "p04_row_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*p04RowChild] { return &p04RowChildBuf{} },
		},
		ParentKey: func(p *p04CountParent) []any { return []any{p.ID} },
		ChildKey:  func(c *p04RowChild) []any { return []any{c.ParentID} },
		Stitch:    func(p *p04CountParent, cs []*p04RowChild) error { return nil },
	}
}

// TestExecIsland_SparseSetNumbersDoNotPanic: planIsland's next-counter is
// shared across row sections and count sections, interleaved in
// declaration order — declaring a count node before a row node gives the
// count section set=1 and the row section set=2. res.items/res.secOrds
// must be sized off the HIGHEST set number in use (via resultSetSlots),
// not off len(plan.sections)+1, or indexing res.items[2] panics with index
// out of range (found during Task 3's review, 2026-08-13).
func TestExecIsland_SparseSetNumbersDoNotPanic(t *testing.T) {
	rowEdge := p04RowEdge()
	countEdge := p04CountEdge()

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

	root := p04CountRoot()
	cols := expectedColumns(root, plan)
	exec := p28FakeExecutor{rows: &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
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
	plan, err := planIsland([]Node{NewCountNode(p04CountEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := p04CountRoot()
	exec := p28FakeExecutor{rows: p04CountRows(expectedColumns(root, plan))}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if err := stitchIsland(plan, res); err != nil {
		t.Fatalf("stitchIsland: %v", err)
	}

	want := map[int]int64{10: 2, 20: 1, 30: 0}
	for _, it := range res.items[0] {
		p := it.(*p04CountParent)
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
	e := p04CountEdge()
	e.ParentKey = func(p *p04CountParent) []any { return nil } // key never set
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := p04CountRoot()
	exec := p28FakeExecutor{rows: p04CountRows(expectedColumns(root, plan))}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if err := stitchIsland(plan, res); err != nil {
		t.Fatalf("stitchIsland: %v", err)
	}
	for _, it := range res.items[0] {
		p := it.(*p04CountParent)
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
	e := p04CountEdge()
	e.SetCount = func(p *p04CountParent, n int64) error {
		return &StaleCountContractError{RelName: "Children"}
	}
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := p04CountRoot()
	exec := p28FakeExecutor{rows: p04CountRows(expectedColumns(root, plan))}

	_, err = All(
		context.Background(),
		exec,
		psql.RawQuery("SELECT id FROM p04_count_parents"),
		p04CountParts(),
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
	e := p04CountEdge()
	plan, err := planIsland([]Node{NewCountNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	exec := &p04QueueExecutor{queue: []scan.Rows{
		p04CountRows(expectedColumns(p04CountRoot(), plan)),
	}}

	out, err := All(
		context.Background(), exec,
		psql.RawQuery("SELECT id FROM p04_count_parents"),
		p04CountParts(),
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
	requireP04CountSQL(t, exec.queries[0])
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
// in resultSetSlots's max-based sizing (Task 3's review, 2026-08-13) could
// in principle resurface there even with the lower-level test passing
// (Copilot review, 2026-08-14, Minor-2).
func TestAll_CountBeforeRowSection_ResultSetSlotsSufficient(t *testing.T) {
	countEdge := p04CountEdge()
	rowEdge := p04RowEdge()

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

	root := p04CountRoot()
	exec := &p04QueueExecutor{queue: []scan.Rows{&p28ColsRows{
		cols: expectedColumns(root, plan),
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
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
		psql.RawQuery("SELECT id FROM p04_count_parents"),
		p04CountParts(),
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
		psql.RawQuery("SELECT id FROM p04_count_parents"),
		p04CountParts(),
		NewCountNode(p04CountEdge()),
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

// requireP04CountSQL asserts that query is a count island's rendered SQL:
// the aggregate itself, the fixed layout's count column, and the UNION ALL
// arm that carries it. Shared by tests that need to check the SQL actually
// issued to a fake executor, not just the SQL Build() renders in isolation.
func requireP04CountSQL(t *testing.T, query string) {
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
	midEdge := p28RootMidEdge()
	countEdge := p04MidLeafCountEdge(recorded)

	rootParts := TableParts[p28Root]{
		Table:   "p28_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
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
	cols1 := expectedColumns(partsAdapter[p28Root]{rootParts}, plan1)

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

	exec := &p04QueueExecutor{queue: []scan.Rows{
		&p28ColsRows{cols: cols1, p28FakeRows: &p28FakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, root.id
			{int64(0), int64(1), 100},
		}}},
		&p28ColsRows{cols: cols2, p28FakeRows: &p28FakeRows{rowsData: [][]any{
			// _sl_set, _sl_ord, mid.id, mid.root_id, mid.other_key,
			// count key (other_key), _sl_cnt1
			{int64(0), int64(1), 200, 100, 999, 0, nil},
			{int64(1), nil, 0, 0, 0, 999, int64(3)},
		}}},
	}}

	out, err := All(
		context.Background(), exec,
		psql.RawQuery("SELECT id FROM p28_root"),
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
	requireP04CountSQL(t, exec.queries[1])
	if len(recorded) != 1 || recorded[200] != 3 {
		t.Fatalf("recorded counts = %v, want map[200:3] (mid 200 counted 3 leaves)", recorded)
	}
}

// p04QueueExecutor hands out one canned Rows per query, in order. An All()
// call may issue several queries (the main island, then one per Then
// boundary), each with its own column layout, so a single-rows fake
// (p28FakeExecutor) cannot serve them all.
type p04QueueExecutor struct {
	queue   []scan.Rows
	queries []string
	i       int
}

func (e *p04QueueExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	if e.i >= len(e.queue) {
		return nil, fmt.Errorf("p04QueueExecutor: unexpected query #%d: %s", e.i+1, query)
	}
	r := e.queue[e.i]
	e.queries = append(e.queries, query)
	e.i++
	return r, nil
}

func (e *p04QueueExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("p04QueueExecutor: ExecContext not implemented (not needed by execIsland)")
}

var _ bob.Executor = (*p04QueueExecutor)(nil)

// p04LeafCountBuf scans exactly the columns it is asked for. A count
// section's buffer is only ever asked for the edge's key columns, so a
// fixture that ignores cols (like p28LeafBuf, which always returns both
// of its targets) would register one scan target too many and misalign
// the whole row.
type p04LeafCountBuf struct{ id, otherKey int }

func (b *p04LeafCountBuf) TargetsFor(cols []Column) []any {
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
			panic("p04LeafCountBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p04LeafCountBuf) BuildFor(modelCols []Column) p28Leaf {
	return p28Leaf{ID: b.id, OtherKey: b.otherKey}
}

// p04MidLeafCountEdge counts p28Leaf rows per p28Mid, reusing the P-28
// model types so a Then boundary can be built without new table types.
// SetCount records into the given map so tests can observe stitching.
func p04MidLeafCountEdge(record map[int]int64) CountEdge[p28Mid, p28Leaf] {
	return CountEdge[p28Mid, p28Leaf]{
		RelName:  "LeafCount",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		Child: TableParts[p28Leaf]{
			Table: "p28_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[p28Leaf] { return &p04LeafCountBuf{} },
		},
		ParentKey: func(p p28Mid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c p28Leaf) []any { return []any{c.OtherKey} },
		SetCount: func(p p28Mid, n int64) error {
			record[p.ID] = n
			return nil
		},
	}
}

// --- validatePlan × count node (P-04 merge self-review, 2026-08-14) ---
//
// validatePlan's check of a ColumnPlan-restricted edge's children
// unconditionally read c.edge, which is nil for a count node child —
// before this fix, any tree nesting a count node under a restricted row
// edge panicked (nil pointer dereference) instead of returning the same
// static errors a restricted row child gets.

// p04RestrictedMidEdge returns p28RootMidEdge with a ColumnPlan that
// fetches only id (Model) + root_id and the given extra key columns
// (Internal), plus the ChildKeyFromBuf override validatePlan requires of
// any restricting edge.
func p04RestrictedMidEdge(t *testing.T, extraKeyCols ...string) Edge[p28Root, p28Mid] {
	t.Helper()
	e := p28RootMidEdge()
	all := e.Child.Columns
	keyCols := append([]string{"root_id"}, extraKeyCols...)
	plan, err := NewColumnPlan(all, []string{"id"}, nil, keyCols)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	e.ColumnPlan = plan
	e.ChildKeyFromBuf = func(b Buf[p28Mid]) []any {
		return []any{b.(*p28MidBuf).rootID}
	}
	return e
}

// TestValidatePlan_CountChildUnderRestrictedEdge_RequiresParentKeyFromBuf:
// before this fix, validatePlan dereferenced c.edge (nil for a count
// node) and panicked; now it must return the same MissingKeyFromBufError
// a row child gets when the restricted parent's buffer would be read
// through the BuildFor(nil) fallback.
func TestValidatePlan_CountChildUnderRestrictedEdge_RequiresParentKeyFromBuf(t *testing.T) {
	parent := p04RestrictedMidEdge(t, "other_key")
	cnt := p04MidLeafCountEdge(map[int]int64{}) // ParentKeyFromBuf deliberately nil

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
// exactly like a row child's (P-03 Important の同型).
func TestValidatePlan_CountChildFromColsMustBeFetched(t *testing.T) {
	parent := p04RestrictedMidEdge(t) // other_key NOT fetched
	cnt := p04MidLeafCountEdge(map[int]int64{})
	cnt.ParentKeyFromBuf = func(b Buf[p28Mid]) []any {
		return []any{b.(*p28MidBuf).otherKey}
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
	parent := p04RestrictedMidEdge(t, "other_key")
	cnt := p04MidLeafCountEdge(map[int]int64{})
	cnt.ParentKeyFromBuf = func(b Buf[p28Mid]) []any {
		return []any{b.(*p28MidBuf).otherKey}
	}

	if err := validatePlan([]Node{NewNode(parent, NewCountNode(cnt))}, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
}
