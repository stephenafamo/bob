package sectionload

// P-33 §8 テスト2〜4: resolveColumnPlans/resolveEdgeColumnPlan の単体テスト
// (エラー系網羅とコピー非破壊性)。fakeParent/fakeChild/fakeChildBuf は
// columnplan_test.go で定義済みの共有フィクスチャを再利用する。

import (
	"errors"
	"testing"
)

// p33FakeEdge is a valid, single-hop Edge[*fakeParent, *fakeChild] fixture
// for resolveColumnPlans/resolveEdgeColumnPlan tests: it has everything
// those functions read (sides()/toCols()/child()/relName()/options()), but
// no ChildKey/Stitch — tests here never reach compose/exec, only the
// resolution step (validatePlan/composeIsland are exercised by the
// p33_e2e_test.go fixtures instead).
func p33FakeEdge() Edge[*fakeParent, *fakeChild] {
	return Edge[*fakeParent, *fakeChild]{
		RelName:  "FakeChild",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*fakeChild]{
			Table: "fake_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "secret", DBType: "text"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[*fakeChild] { return &fakeChildBuf{} },
		},
		ParentKey: func(p *fakeParent) []any { return []any{p.ID} },
		ChildKey:  func(c *fakeChild) []any { return []any{0} },
	}
}

// p33FakeCountEdge is the count-node counterpart of p33FakeEdge, for test
// 3(f) (CountColumnRestrictionUnsupportedError).
func p33FakeCountEdge() CountEdge[*fakeParent, *fakeChild] {
	return CountEdge[*fakeParent, *fakeChild]{
		RelName:  "FakeChildCount",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*fakeChild]{
			Table:   "fake_children",
			Columns: []Column{{Name: "id", DBType: "int4"}, {Name: "parent_id", DBType: "int4"}},
			NewBuf:  func() Buf[*fakeChild] { return &fakeChildBuf{} },
		},
		ParentKey: func(p *fakeParent) []any { return []any{p.ID} },
		ChildKey:  func(c *fakeChild) []any { return []any{0} },
		SetCount:  func(p *fakeParent, n int64) error { return nil },
	}
}

// --- Test 3: resolveColumnPlans エラー系 -----------------------------

// 3(a): Only + Except 両方指定 → *ColumnSelectionConflictError.
func TestResolveColumnPlans_BothOnlyAndExceptIsConflict(t *testing.T) {
	e := p33FakeEdge()
	e.Options = SelectOptions{}.Only("id").Except("secret")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *ColumnSelectionConflictError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ColumnSelectionConflictError", err)
	}
}

// 3(b): Only() を空リストで呼ぶ → *ColumnSelectionEmptyError{Field:"Only"}.
func TestResolveColumnPlans_EmptyOnlyIsError(t *testing.T) {
	e := p33FakeEdge()
	e.Options = Only()

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *ColumnSelectionEmptyError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ColumnSelectionEmptyError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Only" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Only", want)
	}
}

// 3(c): Except() を空リストで呼ぶ → Field:"Except".
func TestResolveColumnPlans_EmptyExceptIsError(t *testing.T) {
	e := p33FakeEdge()
	e.Options = Except()

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *ColumnSelectionEmptyError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *ColumnSelectionEmptyError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Except" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Except", want)
	}
}

// 3(d): 実在しない列名 → *UnknownColumnError.
func TestResolveColumnPlans_UnknownColumnIsError(t *testing.T) {
	e := p33FakeEdge()
	e.Options = Only("does_not_exist")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *UnknownColumnError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *UnknownColumnError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Only" || want.Column != "does_not_exist" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Only Column=does_not_exist", want)
	}
}

// 3(e): 結合キー列を Except に指定 → *KeyColumnExcludedError
// (NewColumnPlan 経由、変更なし)。
func TestResolveColumnPlans_ExceptExcludingKeyIsError(t *testing.T) {
	e := p33FakeEdge()
	e.Options = Except("parent_id")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *KeyColumnExcludedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *KeyColumnExcludedError", err)
	}
	if want.Column != "parent_id" {
		t.Fatalf("err.Column = %q, want parent_id", want.Column)
	}
}

// 3(f): count ノードに Only/Except → *CountColumnRestrictionUnsupportedError.
func TestResolveColumnPlans_CountNodeWithOnlyIsError(t *testing.T) {
	c := p33FakeCountEdge()
	c.Options = Only("id")

	_, err := resolveColumnPlans([]Node{NewCountNode(c)})
	var want *CountColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountColumnRestrictionUnsupportedError", err)
	}
	if want.RelName != "FakeChildCount" {
		t.Fatalf("err.RelName = %q, want FakeChildCount", want.RelName)
	}
}

// 3(g): 何も指定しない SelectOptions{} → ColumnPlan はゼロ値のまま
// (既存 SQL と byte-identical — SQL 生成そのものは compose レベルの
// テストが検証するので、ここでは ColumnPlan がゼロ値のまま解決される
// ことだけを確認する)。
//
// Codexレビュー1回目・指摘4(2026-08-19): 長さ(len==0)だけの確認だと、
// Model/Internalが「非nilの空スライス」(制限されているが公開列が0件、
// M6注記参照)でも成功してしまう。ColumnPlanではnilと非nilの空スライスが
// 異なる意味を持つ(nil=無制限、非nil空=制限されて0件)ため、nilそのものを
// 直接確認する。
func TestResolveColumnPlans_NoRestrictionLeavesColumnPlanZero(t *testing.T) {
	e := p33FakeEdge() // Options はゼロ値(Mods も Only/Except も呼んでいない)

	resolved, err := resolveColumnPlans([]Node{NewNode(e)})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	got := resolved[0].edge.(Edge[*fakeParent, *fakeChild]).ColumnPlan
	if got.Model != nil || got.Internal != nil {
		t.Fatalf("ColumnPlan = %+v, want exact zero value", got)
	}
}

// 3(h): 同じ側で列名を重複指定 → *DuplicateColumnError.
func TestResolveColumnPlans_DuplicateColumnNameIsError(t *testing.T) {
	e := p33FakeEdge()
	e.Options = Except("secret", "secret")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *DuplicateColumnError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *DuplicateColumnError", err)
	}
	if want.RelName != "FakeChild" || want.Field != "Except" || want.Column != "secret" {
		t.Fatalf("err fields = %+v, want RelName=FakeChild Field=Except Column=secret", want)
	}
}

// 3(i): 多段(multi-hop)エッジに Only/Except →
// *MultiHopColumnRestrictionUnsupportedError. p05TwoHopEdge (p05_sql_test.go)
// is a real 2-side (len(Sides)==2) fixture reused from P-05.
func TestResolveColumnPlans_MultiHopEdgeIsError(t *testing.T) {
	e := p05TwoHopEdge()
	e.Options = Only("label")

	_, err := resolveColumnPlans([]Node{NewNode(e)})
	var want *MultiHopColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *MultiHopColumnRestrictionUnsupportedError", err)
	}
	if want.RelName != "Leaves" {
		t.Fatalf("err.RelName = %q, want Leaves", want.RelName)
	}
}

// --- Test 4: resolveColumnPlans の非破壊性 ----------------------------

// TestResolveColumnPlans_DoesNotMutateOriginalEdge is P-33 §8 テスト4
// (6.4節): resolveColumnPlans/withColumnPlan はエッジのコピーにだけ
// ColumnPlan を載せる。呼び出し元が保持する元の Node/Edge 値の
// ColumnPlan は常にゼロ値のままでなければならない — 同じ Node 値を
// 複数回 All()/Build() に渡しても安全であるための前提。
//
// Node.edge は edgeSpec インターフェースで ColumnPlan の getter を持たない
// (columnPlanRestricts() の bool しか無い)ため、元値を検査するには
// 具体型で型アサーションする(I6-2、Opus review 2026-08-18)。
func TestResolveColumnPlans_DoesNotMutateOriginalEdge(t *testing.T) {
	e := p33FakeEdge()
	e.Options = Except("secret")
	original := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{original})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	// 元の Node 値(呼び出し元がまだ握っている値)の ColumnPlan はゼロ値
	// のまま。
	origPlan := original.edge.(Edge[*fakeParent, *fakeChild]).ColumnPlan
	if origPlan.Model != nil || origPlan.Internal != nil {
		t.Fatalf("original edge's ColumnPlan = %+v, want exact zero value (must not be mutated)", origPlan)
	}

	// 解決後の Node 値の ColumnPlan は非ゼロ(secret 以外が Model へ)。
	resolvedPlan := resolved[0].edge.(Edge[*fakeParent, *fakeChild]).ColumnPlan
	if len(resolvedPlan.Model) != 2 {
		t.Fatalf("resolved edge's ColumnPlan.Model = %v, want 2 columns (id, parent_id)", resolvedPlan.Model)
	}
}

// TestResolveColumnPlans_DoesNotMutateOriginalChildEdge is Codexレビュー
// 1回目・指摘4(2026-08-19)の追加テスト: 上のテストは最上位エッジしか
// 検査しておらず、columnselect.go の resolveNode が子配列
// (n.children)を複製してから書き換える処理(6.4節と同じ「コピーであって
// 破壊しない」原則の子ノード版)を検証していなかった。子配列の複製を
// 削除する退行が入ると、元の木の子エッジだけが書き換わってしまうが、
// 最上位エッジだけを見るテストでは検出できない。
func TestResolveColumnPlans_DoesNotMutateOriginalChildEdge(t *testing.T) {
	child := Edge[*fakeChild, *fakeChild]{
		RelName:  "NestedFakeChild",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Options:  Except("secret"),
		Child:    p33FakeEdge().Child,
		ParentKey: func(*fakeChild) []any {
			return []any{0}
		},
		ChildKey: func(*fakeChild) []any {
			return []any{0}
		},
	}

	root := p33FakeEdge()
	original := NewNode(root, NewNode(child))

	resolved, err := resolveColumnPlans([]Node{original})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	originalChildPlan := original.children[0].edge.(Edge[*fakeChild, *fakeChild]).ColumnPlan
	if originalChildPlan.Model != nil || originalChildPlan.Internal != nil {
		t.Fatalf(
			"original child plan = %+v, want exact zero value",
			originalChildPlan,
		)
	}

	resolvedChildPlan := resolved[0].children[0].edge.(Edge[*fakeChild, *fakeChild]).ColumnPlan
	if len(resolvedChildPlan.Model) == 0 {
		t.Fatalf("resolved child plan = %+v, want restricted plan", resolvedChildPlan)
	}
}
