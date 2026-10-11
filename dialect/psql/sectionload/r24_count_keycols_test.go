package sectionload

// R-24 (P-34-multihop-count.md) Codex レビュー1回目 Important 1件対応:
// countKeyCols (node.go) が KeyCols の「列数」しか検査しておらず、列名・
// 順序が誤っていても同じ長さなら通ってしまう問題の否定側テスト。
//
// 例えば Sides[0].ToCols=["parent_id"] に対して KeyCols=[{Name:"leaf_id"}]
// のような取り違えがあっても、旧実装は長さ一致(1==1)だけを見て通してしま
// い、誤った列名でSQLを組み立てた結果、別の親への件数割り当てや全親ゼロ
// という誤集計が静かに成立し得た(Codexレビュー1回目、2026-08-20)。

import (
	"errors"
	"testing"
)

// p05TwoHopCompositeCountEdge: the CountEdge counterpart of
// p05TwoHopCompositeEdge (p05_sql_test.go) — p05_composite_parents(id1,id2)
// -> p05_mid -> p05_leaves, a 2-column composite key, for
// TestR24CountKeyCols_CompositeOrderMismatchIsRejected.
func p05TwoHopCompositeCountEdge() CountEdge[p05CompositeParent, p05Leaf] {
	return CountEdge[p05CompositeParent, p05Leaf]{
		RelName: "CompositeLeavesCount",
		Sides: []Side{
			{
				FromTable: "p05_composite_parents", FromCols: []string{"id1", "id2"},
				ToTable: "p05_mid", ToCols: []string{"parent_id1", "parent_id2"},
			},
			{FromTable: "p05_mid", FromCols: []string{"leaf_id"}, ToTable: "p05_leaves", ToCols: []string{"id"}},
		},
		Child: p05LeafParts(),
		KeyCols: []Column{
			{Name: "parent_id1", DBType: "int4"},
			{Name: "parent_id2", DBType: "int4"},
		},
		NewCarriedBuf: func() Buf[p05CompositeParent] { return &p05CompositeParentBuf{} },
		ParentKey:     func(p p05CompositeParent) []any { return []any{p.ID1, p.ID2} },
		ChildKey:      func(c p05Leaf) []any { return []any{c.ID} },
		SetCount:      func(p p05CompositeParent, n int64) error { return nil },
	}
}

// TestR24CountKeyCols_NameMismatchIsRejected (Codexレビュー1回目 Important):
// KeyCols[0].Name が Sides[0].ToCols[0] と食い違う(長さは一致)場合、
// countKeyCols (node.go、planIsland経由) が CountKeyColsMismatchError を
// 返すこと。
func TestR24CountKeyCols_NameMismatchIsRejected(t *testing.T) {
	edge := p05TwoHopCountEdge()
	if got, want := edge.Sides[0].ToCols[0], "parent_id"; got != want {
		t.Fatalf("fixture precondition failed: Sides[0].ToCols[0] = %q, want %q", got, want)
	}
	edge.KeyCols[0].Name = "leaf_id" // Sides[0].ToCols[0] is "parent_id" — a real column on p05_mid, but the WRONG one.

	_, err := planIsland([]Node{NewCountNode(edge)})
	var want *CountKeyColsMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountKeyColsMismatchError", err)
	}
	if want.RelName != "LeavesCount" {
		t.Fatalf("err = %+v, want RelName=LeavesCount", want)
	}
}

// TestR24CountKeyCols_CompositeOrderMismatchIsRejected (Codexレビュー1回目
// Important): 複合キー(2列)で、KeyCols の列数は一致するが順序が
// Sides[0].ToCols と逆(parent_id2, parent_id1)の場合も、同じ
// CountKeyColsMismatchError で拒否されること。列数だけの検査ではこの
// ケースを検出できない。
func TestR24CountKeyCols_CompositeOrderMismatchIsRejected(t *testing.T) {
	edge := p05TwoHopCompositeCountEdge()
	if got, want := edge.Sides[0].ToCols, ([]string{"parent_id1", "parent_id2"}); got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("fixture precondition failed: Sides[0].ToCols = %v, want %v", got, want)
	}
	edge.KeyCols[0], edge.KeyCols[1] = edge.KeyCols[1], edge.KeyCols[0] // reversed order, same length (2==2).

	_, err := planIsland([]Node{NewCountNode(edge)})
	var want *CountKeyColsMismatchError
	if !errors.As(err, &want) {
		t.Fatalf("err = %v, want *CountKeyColsMismatchError", err)
	}
	if want.RelName != "CompositeLeavesCount" {
		t.Fatalf("err = %+v, want RelName=CompositeLeavesCount", want)
	}
}
