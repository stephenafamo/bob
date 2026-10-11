package sectionload

// Regression test for a bug found in independent review of the P-02×P-03
// branch merge (2026-08-09): unionBranch and composeIsland's hasKids CTE
// body built their physical SELECT column list from n.edge.child().cols()
// (the table's full, unrestricted column list) instead of
// n.edge.childCols() (the ColumnPlan-aware list section0List/checkWidth
// already used correctly) — silently defeating P-03's whole column-subset
// feature the moment any edge actually used a non-zero ColumnPlan. No
// generated Edge sets ColumnPlan yet (the public Only/Except API is still
// deferred post-P-02), so nothing caught this via the existing suite.
//
// Covers both routes the bug touched: route 2 (a leaf node's semi-join
// UNION branch) and route 1/3 (a hasKids node's CTE body + its own
// UNION-branch read-back).

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type p04ColPlanChild struct {
	ID       int
	ParentID int
}

type p04ColPlanChildBuf struct {
	id, parentID int
	secret       string // Internal-adjacent: scanned only if named, never here
}

func (b *p04ColPlanChildBuf) TargetsFor(cols []Column) []any {
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
			panic("p04ColPlanChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p04ColPlanChildBuf) BuildFor(modelCols []Column) p04ColPlanChild {
	if len(modelCols) == 0 {
		return p04ColPlanChild{ID: b.id, ParentID: b.parentID}
	}
	m := p04ColPlanChild{}
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

func p04ColPlanEdge(t *testing.T) Edge[p11CapParent, p04ColPlanChild] {
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
	return Edge[p11CapParent, p04ColPlanChild]{
		RelName:    "ColPlanChild",
		FromCols:   []string{"id"},
		ToCols:     []string{"parent_id"},
		ToMany:     true,
		ColumnPlan: plan,
		Child: TableParts[p04ColPlanChild]{
			Schema:  "",
			Table:   "p04_colplan_children",
			Columns: all,
			NewBuf:  func() Buf[p04ColPlanChild] { return &p04ColPlanChildBuf{} },
		},
		ParentKey: func(p p11CapParent) []any { return []any{p.ID} },
		ChildKey:  func(c p04ColPlanChild) []any { return []any{c.ParentID} },
		Stitch:    func(p p11CapParent, cs []p04ColPlanChild) error { return nil },
	}
}

// TestUnionBranch_ColumnPlanRestrictsLeafSelectList covers route 2 (a leaf
// node's semi-join UNION branch, unionBranch's non-hasKids path).
func TestUnionBranch_ColumnPlanRestrictsLeafSelectList(t *testing.T) {
	e := p04ColPlanEdge(t)
	tree := NewNode(e)

	root := partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema:  "",
		Table:   "p11_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p11CapParent] { return &p02ParentBuf{} },
	}}
	plan, err := planIsland([]Node{tree})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p11_parents")

	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
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
	e := p04ColPlanEdge(t)
	// A grandchild under e makes e's node hasKids (route 1/3), exercising
	// the composeIsland CTE-body call site the bug also affected.
	tree := NewNode(e, NewNode(p02GrandchildEdge()))

	root := partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema:  "",
		Table:   "p11_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p11CapParent] { return &p02ParentBuf{} },
	}}
	plan, err := planIsland([]Node{tree})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p11_parents")

	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
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

// --- P-33 §8 テスト6・7・10(d) --------------------------------------
//
// p33ColPlanEdgeViaOptions is p04ColPlanEdge's P-33 counterpart: instead of
// hand-setting ColumnPlan directly (P-03's original test style, before the
// public Only/Except API existed), it sets Options.Except("secret") and
// lets resolveColumnPlans (columnselect.go) compute ColumnPlan the same
// way a generated RelWith(sectionload.Except(...), ...) call would. It
// also sets ChildKeyFromBuf, simulating what 01_edges.go.tpl generates for
// EVERY edge post-P-33 (7.4節) — required by validatePlan for any
// single-hop restricted edge.

func p33ColPlanEdgeViaOptions(opts SelectOptions) Edge[p11CapParent, p04ColPlanChild] {
	return Edge[p11CapParent, p04ColPlanChild]{
		RelName:  "ColPlanChildViaOptions",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Options:  opts,
		Child: TableParts[p04ColPlanChild]{
			Table: "p04_colplan_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "secret", DBType: "text"},
			},
			NewBuf: func() Buf[p04ColPlanChild] { return &p04ColPlanChildBuf{} },
		},
		ParentKey: func(p p11CapParent) []any { return []any{p.ID} },
		ChildKey:  func(c p04ColPlanChild) []any { return []any{c.ParentID} },
		ChildKeyFromBuf: func(b Buf[p04ColPlanChild]) []any {
			return []any{b.(*p04ColPlanChildBuf).parentID}
		},
		Stitch: func(p p11CapParent, cs []p04ColPlanChild) error { return nil },
	}
}

func p33ColPlanRoot() partsAdapter[p11CapParent] {
	return partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema:  "",
		Table:   "p11_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p11CapParent] { return &p02ParentBuf{} },
	}}
}

// TestValidatePlan_ExceptRestrictedEdgeWithGeneratedStyleChildKeyFromBufPasses
// is P-33 §8 テスト6: Except を指定したエッジで木を組んだとき、
// ChildKeyFromBuf が(テンプレート拡張後は)常に設定されているため
// validatePlan を通過することを確認する(発見Aが解消されたことの直接
// 確認)。
func TestValidatePlan_ExceptRestrictedEdgeWithGeneratedStyleChildKeyFromBufPasses(t *testing.T) {
	e := p33ColPlanEdgeViaOptions(Except("secret"))
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v (発見A: ChildKeyFromBufが常に生成されるようになったので通るはず)", err)
	}
}

// TestSelectOptions_ModsThenExceptComposesWhereAndColumnRestriction is
// P-33 §8 テスト7: sectionload.Mods(sm.Where(...)).Except("col") のチェーン
// を実際に組み立て、(a) 解決後の ColumnPlan が正しく col を除外している
// こと、(b) 生成 SQL に WHERE 条件と絞られた SELECT 列の両方が反映されて
// いることを compose レベルのテストで確認する(判定1、6.9節)。逆方向
// (Only/Except起点からModsを後付けする入口が無いこと)はコンパイル
// エラーになる性質上テストではなくdocコメントに一本化する
// (I6-3、options.goのExceptのdocコメント参照)。
func TestSelectOptions_ModsThenExceptComposesWhereAndColumnRestriction(t *testing.T) {
	opts := Mods(sm.Where(psql.Quote("status").EQ(psql.Arg("active")))).Except("secret")
	e := p33ColPlanEdgeViaOptions(opts)
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	// (a) 解決後の ColumnPlan が正しく secret を除外している。
	plan := resolved[0].edge.(Edge[p11CapParent, p04ColPlanChild]).ColumnPlan
	for _, c := range plan.All() {
		if c.Name == "secret" {
			t.Fatalf("ColumnPlan.All() = %v, must not contain secret", plan.All())
		}
	}
	if len(plan.All()) == 0 {
		t.Fatal("ColumnPlan.All() is empty, want id and parent_id at least")
	}

	root := p33ColPlanRoot()
	islandPlan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p11_parents")

	q, err := composeIsland(parent, root, islandPlan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s\nargs: %v", sql, args)

	// (b) WHERE 条件と絞られた SELECT 列の両方が反映されていること。
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

// TestBuildEdgeSelect_ModsColumnsPlusExceptStillRejected is P-33 §8
// テスト10(d): sectionload.Mods(sm.Columns(...)).Except("col") のチェーンは
// resolveColumnPlans を通過した後、compose 時(buildEdgeSelect 内の
// extractOptionsFragment)で *SelectOptionsStructureError{Field:"select
// columns"} により拒否されることを確認する(I4、Opus review 2026-08-18:
// 発見Cの全称主張「mod経由の生の列指定は従来どおり拒否」に対する否定側
// テスト。P-33が初めて可能にする Only/Except との組み合わせであり、新しい
// 抜け道が無いことの確認)。
func TestBuildEdgeSelect_ModsColumnsPlusExceptStillRejected(t *testing.T) {
	opts := Mods(sm.Columns(psql.Quote("id"))).Except("secret")
	e := p33ColPlanEdgeViaOptions(opts)
	tree := NewNode(e)

	// resolveColumnPlans は Options.only/except しか見ないので、mods の
	// 中身(sm.Columns)には一切関知せず通過する。
	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v (mods自体はresolveColumnPlansの検査対象外のはず)", err)
	}

	root := p33ColPlanRoot()
	islandPlan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p11_parents")

	_, err = composeIsland(parent, root, islandPlan, DefaultMaxColumns, "")
	want, ok := err.(*SelectOptionsStructureError)
	if !ok {
		t.Fatalf("expected *SelectOptionsStructureError, got %T: %v", err, err)
	}
	if want.Field != "select columns" {
		t.Fatalf("err.Field = %q, want \"select columns\"", want.Field)
	}
}
