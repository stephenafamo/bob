package sectionload

// P-26: execIsland は行ループの外側でスキャンバッファを1つ作り、全行を
// そこへ Scan していた。生成コードの BuildFor はバッファのフィールドを
// 値コピーしてモデルを作るが、json.RawMessage のようにスライスを持つ型は
// スライスヘッダしかコピーされない。生成物の jsonb スキャン先である
// null.Val[types.JSON[json.RawMessage]] は最終的に
// json.RawMessage.UnmarshalJSON、すなわち
//
//	*m = append((*m)[:0], data...)
//
// に落ちるので、容量が足りるかぎり「既存の裏配列へインプレース」で書き
// 込む。したがって次の行の Scan が直前の行のモデルの中身を書き換える。
//
// このファイルはその挙動を DB 無しで再現する。下の p26FakeRows.Scan は
// *json.RawMessage 宛てに append((*v)[:0], ...) を行い、標準ライブラリと
// 同じインプレース書き込みを再現する。
//
// 【重要】2つのペイロードは同じバイト長でなければならない。2本目が長いと
// append が新しい配列を確保してしまい、修正前のコードでもこのテストが
// 通ってしまう(= 検査自体が壊れていることに気づけない)。p26RequireSameLength
// がその前提を毎回確認する。

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// --- fixtures: jsonb 相当の列を持つ root と child ---

type p26Root struct {
	ID   int
	Blob json.RawMessage
}

type p26RootBuf struct {
	id   int
	blob json.RawMessage
}

func (b *p26RootBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.blob} }

// BuildFor は生成コードと同じ素の値コピー。Blob はスライスなのでヘッダ
// だけがコピーされ、裏配列はバッファと共有されたままになる。
func (b *p26RootBuf) BuildFor(modelCols []Column) p26Root {
	return p26Root{ID: b.id, Blob: b.blob}
}

type p26Child struct {
	ID       int
	ParentID int
	Blob     json.RawMessage
}

type p26ChildBuf struct {
	id, parentID int
	blob         json.RawMessage
}

func (b *p26ChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.parentID, &b.blob}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "parent_id":
			out[i] = &b.parentID
		case "blob":
			out[i] = &b.blob
		default:
			panic("p26ChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p26ChildBuf) BuildFor(modelCols []Column) p26Child {
	return p26Child{ID: b.id, ParentID: b.parentID, Blob: b.blob}
}

func p26RootParts() TableParts[p26Root] {
	return TableParts[p26Root]{
		Table: "p26_root",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "blob", DBType: "jsonb"},
		},
		NewBuf: func() Buf[p26Root] { return &p26RootBuf{} },
	}
}

func p26Edge() Edge[p26Root, p26Child] {
	return Edge[p26Root, p26Child]{
		RelName:  "Children",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Child: TableParts[p26Child]{
			Table: "p26_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "blob", DBType: "jsonb"},
			},
			NewBuf: func() Buf[p26Child] { return &p26ChildBuf{} },
		},
		ParentKey: func(p p26Root) []any { return []any{p.ID} },
		ChildKey:  func(c p26Child) []any { return []any{c.ParentID} },
		Stitch:    func(p p26Root, cs []p26Child) error { return nil },
	}
}

// --- fake rows: *json.RawMessage へインプレース書き込みする Scan ---
//
// p28FakeRows(p28_thenkey_test.go)と同じ役割だが、あちらは int/int64 系
// しか扱わない。ここで必要なのは「既存の裏配列へ追記する」スキャン先なので、
// 専用のものを用意する。

type p26FakeRows struct {
	cols     []string
	rowsData [][]any
	i        int
}

func (r *p26FakeRows) Columns() ([]string, error) { return r.cols, nil }
func (r *p26FakeRows) Next() bool                 { r.i++; return r.i <= len(r.rowsData) }
func (r *p26FakeRows) Err() error                 { return nil }
func (r *p26FakeRows) Close() error               { return nil }

func (r *p26FakeRows) Scan(dest ...any) error {
	row := r.rowsData[r.i-1]
	if len(dest) != len(row) {
		return fmt.Errorf("p26FakeRows.Scan: %d destinations, %d values", len(dest), len(row))
	}
	for i, d := range dest {
		switch v := d.(type) {
		case *int64:
			*v = row[i].(int64)
		case *sql.NullInt64:
			if row[i] == nil {
				*v = sql.NullInt64{}
			} else {
				*v = sql.NullInt64{Int64: row[i].(int64), Valid: true}
			}
		case *int:
			*v = row[i].(int)
		case *json.RawMessage:
			// 標準ライブラリの再現: json.RawMessage.UnmarshalJSON は
			// *m = append((*m)[:0], data...) で既存の裏配列へ書き込む。
			// row[i] が nil(他セクションの NULL パディング)のときは
			// 長さ 0 に切り詰めるだけ、という点も本物と同じ。
			b, _ := row[i].([]byte)
			*v = append((*v)[:0], b...)
		default:
			return fmt.Errorf("p26FakeRows.Scan: unsupported dest type %T at position %d", d, i)
		}
	}
	return nil
}

// p26PayloadA/B は相異なる、かつ同じバイト長の JSON。長さを揃えるのは
// 必須条件で、2本目が長いと append が再割り当てしてしまい、修正前の
// コードでもテストが通ってしまう。
var (
	p26PayloadA = []byte(`{"row":"a"}`)
	p26PayloadB = []byte(`{"row":"b"}`)
)

func p26RequireSameLength(t *testing.T) {
	t.Helper()
	if len(p26PayloadA) != len(p26PayloadB) {
		t.Fatalf(
			"p26PayloadA/B のバイト長が違う(%d vs %d)。長さが違うと append が新しい配列を確保するため、"+
				"バッファを使い回す修正前のコードでもこのテストが通ってしまい、検査として成立しない",
			len(p26PayloadA), len(p26PayloadB),
		)
	}
}

// --- tests ---

// TestExecIsland_SectionBufferNotAliasedAcrossRows: 同じセクションから2行
// 返ったとき、1行目の子モデルの JSON が2行目の値に化けてはならない。
func TestExecIsland_SectionBufferNotAliasedAcrossRows(t *testing.T) {
	p26RequireSameLength(t)

	plan, err := planIsland([]Node{NewNode(p26Edge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 {
		t.Fatalf("sections: got %d, want 1", len(plan.sections))
	}
	set := plan.sections[0].set

	root := partsAdapter[p26Root]{p26RootParts()}
	cols := expectedColumns(root, plan)

	// layout: _sl_set, _sl_ord, root.id, root.blob,
	//         child.id, child.parent_id, child.blob
	exec := p28FakeExecutor{rows: &p26FakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, []byte(`{}`), 0, 0, nil},
			{int64(1), nil, 0, nil, 10, 1, p26PayloadA},
			{int64(1), nil, 0, nil, 11, 1, p26PayloadB},
		},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[set]) != 2 {
		t.Fatalf("section items: got %d, want 2", len(res.items[set]))
	}

	first := res.items[set][0].(p26Child)
	second := res.items[set][1].(p26Child)
	if string(first.Blob) != string(p26PayloadA) {
		t.Fatalf(
			"1行目の子モデルの JSON = %q, want %q — セクションのバッファを行ごとに作り直していないため、"+
				"2行目の Scan が1行目のモデルの裏配列を書き換えた(P-26)",
			first.Blob, p26PayloadA,
		)
	}
	if string(second.Blob) != string(p26PayloadB) {
		t.Fatalf("2行目の子モデルの JSON = %q, want %q", second.Blob, p26PayloadB)
	}
}

// TestExecIsland_RootBufferNotAliasedAcrossRows: root(set 0)から2行返った
// ときも同じ。jsonb 列を持つテーブルが親になった場合の経路。
func TestExecIsland_RootBufferNotAliasedAcrossRows(t *testing.T) {
	p26RequireSameLength(t)

	plan, err := planIsland([]Node{NewNode(p26Edge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := partsAdapter[p26Root]{p26RootParts()}
	cols := expectedColumns(root, plan)

	exec := p28FakeExecutor{rows: &p26FakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, p26PayloadA, 0, 0, nil},
			{int64(0), int64(2), 2, p26PayloadB, 0, 0, nil},
		},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[0]) != 2 {
		t.Fatalf("root items: got %d, want 2", len(res.items[0]))
	}

	first := res.items[0][0].(p26Root)
	second := res.items[0][1].(p26Root)
	if string(first.Blob) != string(p26PayloadA) {
		t.Fatalf(
			"1行目の親モデルの JSON = %q, want %q — root のバッファを行ごとに作り直していないため、"+
				"2行目の Scan が1行目のモデルの裏配列を書き換えた(P-26)",
			first.Blob, p26PayloadA,
		)
	}
	if string(second.Blob) != string(p26PayloadB) {
		t.Fatalf("2行目の親モデルの JSON = %q, want %q", second.Blob, p26PayloadB)
	}
}

// p26EdgeColumnPlanRestricted is p26Edge with a ColumnPlan that keeps
// id/blob in Model and pushes parent_id (the join key) to Internal —
// childCols() then returns id, blob, parent_id, a DIFFERENT order than
// the table's declared id, parent_id, blob.
func p26EdgeColumnPlanRestricted(t *testing.T) Edge[p26Root, p26Child] {
	t.Helper()
	e := p26Edge()
	plan, err := NewColumnPlan(
		e.Child.Columns,
		[]string{"id", "blob"},
		nil,
		[]string{"parent_id"},
	)
	if err != nil {
		t.Fatalf("NewColumnPlan: %v", err)
	}
	e.ColumnPlan = plan
	return e
}

// TestExecIsland_SectionBufferNotAliasedAcrossRows_ColumnPlanRestricted is
// the ColumnPlan-restricted counterpart of
// TestExecIsland_SectionBufferNotAliasedAcrossRows (Copilotレビュー
// Minor 2、2026-08-14): refreshBuf's caller uses n.edge.childCols(), which
// under a ColumnPlan is Model++Internal — a different column order than
// n.edge.child().cols() (the table's declared order). This test exercises
// that reordered path instead of only the no-ColumnPlan case above.
func TestExecIsland_SectionBufferNotAliasedAcrossRows_ColumnPlanRestricted(t *testing.T) {
	p26RequireSameLength(t)

	edge := p26EdgeColumnPlanRestricted(t)
	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 {
		t.Fatalf("sections: got %d, want 1", len(plan.sections))
	}
	set := plan.sections[0].set

	root := partsAdapter[p26Root]{p26RootParts()}
	cols := expectedColumns(root, plan)

	// layout: _sl_set, _sl_ord, root.id, root.blob,
	//         child.id, child.blob, child.parent_id
	// (childCols() order under this ColumnPlan is Model[id,blob] then
	// Internal[parent_id] — NOT the table's declared id,parent_id,blob.)
	exec := p28FakeExecutor{rows: &p26FakeRows{
		cols: cols,
		rowsData: [][]any{
			{int64(0), int64(1), 1, []byte(`{}`), 0, nil, 0},
			{int64(1), nil, 0, nil, 10, p26PayloadA, 1},
			{int64(1), nil, 0, nil, 11, p26PayloadB, 1},
		},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	if len(res.items[set]) != 2 {
		t.Fatalf("section items: got %d, want 2", len(res.items[set]))
	}

	first := res.items[set][0].(p26Child)
	second := res.items[set][1].(p26Child)
	if string(first.Blob) != string(p26PayloadA) {
		t.Fatalf(
			"1行目の子モデルの JSON = %q, want %q — ColumnPlan制限下でセクションのバッファを行ごとに"+
				"作り直していないため、2行目の Scan が1行目のモデルの裏配列を書き換えた(P-26)",
			first.Blob, p26PayloadA,
		)
	}
	if string(second.Blob) != string(p26PayloadB) {
		t.Fatalf("2行目の子モデルの JSON = %q, want %q", second.Blob, p26PayloadB)
	}
}
