package sectionload

// Regression test: unionBranch and composeIsland's hasKids CTE body must
// build their physical SELECT column list from n.edge.childCols() (the
// ColumnPlan-aware list that section0List/checkWidth already use), not from
// n.edge.child().cols() (the table's full, unrestricted column list).
// Using the latter silently defeats the column-subset feature as soon as an
// edge has a non-zero ColumnPlan.
//
// Covers both routes: route 2 (a leaf node's semi-join UNION branch) and
// route 1/3 (a hasKids node's CTE body + its own UNION-branch read-back).

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type colPlanChild struct {
	ID       int
	ParentID int
}

type colPlanChildBuf struct {
	id, parentID int
	secret       string // Internal-adjacent: scanned only if named, never here
}

func (b *colPlanChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.parentID, &b.secret}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "parent_id":
			out[i] = &b.parentID
		case "secret":
			out[i] = &b.secret
		default:
			panic("colPlanChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *colPlanChildBuf) BuildFor(modelCols []Column) colPlanChild {
	if len(modelCols) == 0 {
		return colPlanChild{ID: b.id, ParentID: b.parentID}
	}
	m := colPlanChild{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "parent_id":
			m.ParentID = b.parentID
		}
	}
	return m
}

func colPlanEdge(t *testing.T) Edge[builderCapParent, colPlanChild] {
	t.Helper()
	all := []Column{
		{Name: "id", DBType: "int4"},
		{Name: "parent_id", DBType: "int4"},
		{Name: "secret", DBType: "text"},
	}
	plan, err := NewColumnPlan(all, []string{"id"}, nil, []string{"parent_id"})
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	if len(plan.Model) != 1 || len(plan.Internal) != 1 {
		t.Fatalf("test setup: plan = %+v, want Model=[id] Internal=[parent_id]", plan)
	}
	return Edge[builderCapParent, colPlanChild]{
		RelName:    "ColPlanChild",
		FromCols:   []string{"id"},
		ToCols:     []string{"parent_id"},
		ToMany:     true,
		ColumnPlan: plan,
		Child: TableParts[colPlanChild]{
			Schema:  "",
			Table:   "colplan_children",
			Columns: all,
			NewBuf:  func() Buf[colPlanChild] { return &colPlanChildBuf{} },
		},
		ParentKey: func(p builderCapParent) []any { return []any{p.ID} },
		ChildKey:  func(c colPlanChild) []any { return []any{c.ParentID} },
		Stitch:    func(p builderCapParent, cs []colPlanChild) error { return nil },
	}
}

// TestUnionBranch_ColumnPlanRestrictsLeafSelectList covers route 2 (a leaf
// node's semi-join UNION branch, unionBranch's non-hasKids path).
func TestUnionBranch_ColumnPlanRestrictsLeafSelectList(t *testing.T) {
	e := colPlanEdge(t)
	tree := NewNode(e)

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
	parent := psql.RawQuery("SELECT id FROM bld_parents")

	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s", sql)

	if strings.Contains(sql, "secret") {
		t.Fatalf("ColumnPlan excluded \"secret\", but it appears in the SQL:\n%s", sql)
	}
	if !strings.Contains(sql, `"parent_id"`) {
		t.Fatalf("ColumnPlan's Internal key column \"parent_id\" must still be fetched for stitching; got:\n%s", sql)
	}
}

// TestComposeIsland_ColumnPlanRestrictsHasKidsCTEAndReadback covers routes
// 1 and 3 together: a hasKids node's own CTE body (route 1) and its
// UNION-branch read-back from that CTE (route 3) must both use the
// restricted column list, not the table's full columns.
func TestComposeIsland_ColumnPlanRestrictsHasKidsCTEAndReadback(t *testing.T) {
	e := colPlanEdge(t)
	// A grandchild under e makes e's node hasKids (route 1/3), exercising
	// the composeIsland CTE-body call site the bug also affected.
	tree := NewNode(e, NewNode(hasKidsGrandchildEdge()))

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
	parent := psql.RawQuery("SELECT id FROM bld_parents")

	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s", sql)

	if strings.Contains(sql, "secret") {
		t.Fatalf("ColumnPlan excluded \"secret\" from the hasKids node's CTE body/read-back, but it appears in the SQL:\n%s", sql)
	}
	if !strings.Contains(sql, `"parent_id"`) {
		t.Fatalf("ColumnPlan's Internal key column \"parent_id\" must still be fetched for stitching; got:\n%s", sql)
	}
}

// --- Column restriction through SelectOptions --------------------------------
//
// colRestrictColPlanEdgeViaOptions is colPlanEdge's counterpart that goes through
// the public API: instead of hand-setting ColumnPlan directly, it sets
// Options.Except("secret") and lets resolveColumnPlans (columnselect.go)
// compute ColumnPlan the same way a generated RelWith(sectionload.Except(...),
// ...) call would. It also sets ChildKeyFromBuf, simulating what the
// generated edges template emits for every edge — required by validatePlan
// for any single-hop restricted edge.

func colRestrictColPlanEdgeViaOptions(opts SelectOptions) Edge[builderCapParent, colPlanChild] {
	return Edge[builderCapParent, colPlanChild]{
		RelName:  "ColPlanChildViaOptions",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Options:  opts,
		Child: TableParts[colPlanChild]{
			Table: "colplan_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "secret", DBType: "text"},
			},
			NewBuf: func() Buf[colPlanChild] { return &colPlanChildBuf{} },
		},
		ParentKey: func(p builderCapParent) []any { return []any{p.ID} },
		ChildKey:  func(c colPlanChild) []any { return []any{c.ParentID} },
		ChildKeyFromBuf: func(b Buf[colPlanChild]) []any {
			return []any{b.(*colPlanChildBuf).parentID}
		},
		Stitch: func(p builderCapParent, cs []colPlanChild) error { return nil },
	}
}

func colRestrictColPlanRoot() partsAdapter[builderCapParent] {
	return partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema:  "",
		Table:   "bld_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[builderCapParent] { return &hasKidsParentBuf{} },
	}}
}

// TestValidatePlan_ExceptRestrictedEdgeWithGeneratedStyleChildKeyFromBufPasses
// checks that a tree built from an Except-restricted edge passes
// validatePlan, because ChildKeyFromBuf is always set by the generated
// edges.
func TestValidatePlan_ExceptRestrictedEdgeWithGeneratedStyleChildKeyFromBufPasses(t *testing.T) {
	e := colRestrictColPlanEdgeViaOptions(Except("secret"))
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v (ChildKeyFromBuf is always generated, so this should pass)", err)
	}
}

// TestSelectOptions_ModsThenExceptComposesWhereAndColumnRestriction is
// builds a sectionload.Mods(sm.Where(...)).Except("col") chain and checks at
// the compose level that (a) the resolved ColumnPlan excludes col and (b) the
// generated SQL reflects both the WHERE condition and the narrowed SELECT
// columns. The reverse direction (no entry point to add Mods after an
// Only/Except) is a compile-time property, so it is documented on Except in
// options.go rather than tested.
func TestSelectOptions_ModsThenExceptComposesWhereAndColumnRestriction(t *testing.T) {
	opts := Mods(sm.Where(psql.Quote("status").EQ(psql.Arg("active")))).Except("secret")
	e := colRestrictColPlanEdgeViaOptions(opts)
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	// (a) The resolved ColumnPlan excludes secret.
	plan := resolved[0].edge.(Edge[builderCapParent, colPlanChild]).ColumnPlan
	for _, c := range plan.All() {
		if c.Name == "secret" {
			t.Fatalf("ColumnPlan.All() = %v, must not contain secret", plan.All())
		}
	}
	if len(plan.All()) == 0 {
		t.Fatal("ColumnPlan.All() is empty, want id and parent_id at least")
	}

	root := colRestrictColPlanRoot()
	islandPlan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM bld_parents")

	q, err := composeIsland(parent, root, islandPlan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s\nargs: %v", sql, args)

	// (b) Both the WHERE condition and the narrowed SELECT columns are reflected.
	if !strings.Contains(sql, `"status"`) {
		t.Fatalf("WHERE condition (status) is missing from the composed SQL:\n%s", sql)
	}
	if len(args) == 0 || args[0] != "active" {
		t.Fatalf("args = %v, want [\"active\", ...] from sm.Where", args)
	}
	if strings.Contains(sql, "secret") {
		t.Fatalf("Except(\"secret\") excluded \"secret\", but it appears in the SQL:\n%s", sql)
	}
	if !strings.Contains(sql, `"parent_id"`) {
		t.Fatalf("the join key column \"parent_id\" must still be fetched for stitching; got:\n%s", sql)
	}
}

// TestBuildEdgeSelect_ModsColumnsPlusExceptStillRejected checks that a
// sectionload.Mods(sm.Columns(...)).Except("col") chain, after passing
// resolveColumnPlans, is rejected at compose time (extractOptionsFragment in
// buildEdgeSelect) with *SelectOptionsStructureError{Field:"select columns"}.
// This is the negative test for "raw column selection through a mod is still
// rejected": Only/Except combined with such a mod is a new combination, and
// this confirms it opens no loophole.
func TestBuildEdgeSelect_ModsColumnsPlusExceptStillRejected(t *testing.T) {
	opts := Mods(sm.Columns(psql.Quote("id"))).Except("secret")
	e := colRestrictColPlanEdgeViaOptions(opts)
	tree := NewNode(e)

	// resolveColumnPlans only looks at Options.only/except, so it ignores
	// the contents of the mods (sm.Columns) and lets the edge through.
	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v (mods themselves are not inspected by resolveColumnPlans)", err)
	}

	root := colRestrictColPlanRoot()
	islandPlan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM bld_parents")

	_, err = composeIsland(parent, root, islandPlan, "")
	var want *SelectOptionsStructureError
	if !errors.As(err, &want) {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
	if want.Field != "select columns" {
		t.Fatalf("err.Field = %q, want \"select columns\"", want.Field)
	}
}
