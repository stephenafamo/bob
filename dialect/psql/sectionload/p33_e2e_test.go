package sectionload

// P-33 §8 テスト8・9・10(c)・12・13・14: DB不要の end-to-end / 統合テスト。
//
// ハーネスは p26_buffer_aliasing_test.go/p28_thenkey_test.go の
// p28FakeExecutor + p28FakeRows/p28ColsRows(いずれもこのパッケージの他
// テストファイルで定義済み)を流用する。フィクスチャ自体は p28Root/p28Mid
// などの既存フィクスチャを流用せず、この issue 専用に新設する
// (p28MidBuf.TargetsFor は cols 引数を無視して常に3列返す実装のため、
// ColumnPlan で列を絞る P-33 のテストにはそのまま使えない — 生成コードの
// switch-per-column 方式を模した専用の p33*Buf を用意する)。
//
// テスト10(c)(多段エッジへの Only/Except は明示エラー)は
// columnselect_test.go の TestResolveColumnPlans_MultiHopEdgeIsError
// (テスト3(i))と全く同じ検証(p05TwoHopEdge + resolveColumnPlans →
// *MultiHopColumnRestrictionUnsupportedError)であるため、ここでは重複
//実装しない。テスト10(a)(b)も同様に既存テストで満たされている
// (10(a): validate_test.go の
// TestValidatePlan_RestrictedEdgeMissingChildKeyFromBufIsError。10(b):
// columnplan_test.go の TestNewColumnPlan_ExceptExcludingKeyIsError と
// columnselect_test.go の TestResolveColumnPlans_ExceptExcludingKeyIsError)。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

// --- fixtures: p33Root -> p33Mid -> p33Leaf (すべて単一hop) -----------

type (
	p33Root    struct{ ID int }
	p33RootBuf struct{ id int }
)

func (b *p33RootBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *p33RootBuf) BuildFor(modelCols []Column) p33Root {
	return p33Root{ID: b.id}
}

type (
	p33Mid    struct{ ID, RootID, OtherKey int }
	p33MidBuf struct{ id, rootID, otherKey int }
)

func (b *p33MidBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.rootID, &b.otherKey}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "root_id":
			out[i] = &b.rootID
		case "other_key":
			out[i] = &b.otherKey
		default:
			panic("p33MidBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p33MidBuf) BuildFor(modelCols []Column) p33Mid {
	if modelCols == nil {
		return p33Mid{ID: b.id, RootID: b.rootID, OtherKey: b.otherKey}
	}
	m := p33Mid{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "root_id":
			m.RootID = b.rootID
		case "other_key":
			m.OtherKey = b.otherKey
		}
	}
	return m
}

type (
	p33Leaf    struct{ ID, OtherKey int }
	p33LeafBuf struct{ id, otherKey int }
)

func (b *p33LeafBuf) TargetsFor(cols []Column) []any {
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
			panic("p33LeafBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p33LeafBuf) BuildFor(modelCols []Column) p33Leaf {
	return p33Leaf{ID: b.id, OtherKey: b.otherKey}
}

func p33RootTableParts() TableParts[p33Root] {
	return TableParts[p33Root]{
		Table:   "p33_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p33Root] { return &p33RootBuf{} },
	}
}

func p33RootParts() partsAdapter[p33Root] {
	return partsAdapter[p33Root]{p33RootTableParts()}
}

// p33RootMidEdge is the "親エッジ"(祖父母→親): root -> mid, restricted via
// Options (Only/Except), with ChildKeyFromBuf set to simulate what
// 01_edges.go.tpl generates for EVERY edge post-P-33 (7.4節).
func p33RootMidEdge(opts SelectOptions) Edge[p33Root, p33Mid] {
	return Edge[p33Root, p33Mid]{
		RelName:  "Mid",
		FromCols: []string{"id"},
		ToCols:   []string{"root_id"},
		ToMany:   false,
		Options:  opts,
		Child: TableParts[p33Mid]{
			Table: "p33_mid",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "root_id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[p33Mid] { return &p33MidBuf{} },
		},
		ParentKey: func(p p33Root) []any { return []any{p.ID} },
		ChildKey:  func(c p33Mid) []any { return []any{c.RootID} },
		ChildKeyFromBuf: func(b Buf[p33Mid]) []any {
			return []any{b.(*p33MidBuf).rootID}
		},
		Stitch: func(p p33Root, cs []p33Mid) error { return nil },
	}
}

// p33MidLeafEdge is the "子エッジ"(親→子): mid -> leaf, unrestricted.
// FromCols=["other_key"] is the column I3/6.10節 describes as NOT
// auto-protected by p33RootMidEdge's own NewColumnPlan (which only knows
// about "root_id", its own toCols()).
func p33MidLeafEdge() Edge[p33Mid, p33Leaf] {
	return Edge[p33Mid, p33Leaf]{
		RelName:  "Leaf",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		ToMany:   true,
		Child: TableParts[p33Leaf]{
			Table: "p33_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[p33Leaf] { return &p33LeafBuf{} },
		},
		ParentKey: func(p p33Mid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c p33Leaf) []any { return []any{c.OtherKey} },
		ParentKeyFromBuf: func(b Buf[p33Mid]) []any {
			return []any{b.(*p33MidBuf).otherKey}
		},
		Stitch: func(p p33Mid, cs []p33Leaf) error { return nil },
	}
}

// p33MidLeafCountEdge is p33MidLeafEdge's count-node counterpart, for
// テスト9.
func p33MidLeafCountEdge() CountEdge[p33Mid, p33Leaf] {
	return CountEdge[p33Mid, p33Leaf]{
		RelName:  "LeafCount",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		Child: TableParts[p33Leaf]{
			Table: "p33_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[p33Leaf] { return &p33LeafBuf{} },
		},
		ParentKey: func(p p33Mid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c p33Leaf) []any { return []any{c.OtherKey} },
		ParentKeyFromBuf: func(b Buf[p33Mid]) []any {
			return []any{b.(*p33MidBuf).otherKey}
		},
		// ChildKeyFromBuf (Codexレビュー1回目・指摘2 関連テスト、
		// 2026-08-19、Codexレビュー2回目・軽微指摘で明示列BuildForへ修正):
		// I5(7.4節)で02_counts.go.tplが生成するようになったChildKeyFromBuf
		// を、フィクスチャ側にも反映する。無ければ childKeyFromBufAny
		// (node.go)がChildKey(b.BuildFor(nil))へフォールバックし、count
		// 専用のkeyCols限定スキャンバッファに対してBuildFor(nil)(=全列
		// 構築)を呼んでしまう経路が生成後の実コードに残ることを、この
		// フィクスチャでは検出できなくなる。生成コード
		// (02_counts.go.tplのChildKeyFromBuf)と同じく、型アサーションで
		// 直接フィールドを読むのではなく、明示的な列リストで
		// b.BuildFor(...)を呼ぶ形にする(型アサーション直読みだと、
		// 生成コードが実際にBuildForを経由する経路の検証にならない)。
		ChildKeyFromBuf: func(b Buf[p33Leaf]) []any {
			o := b.BuildFor([]Column{
				{Name: "other_key", DBType: "int4"},
			})
			return []any{o.OtherKey}
		},
		SetCount: func(p p33Mid, n int64) error { return nil },
	}
}

// p33PanicOnFullBuildLeafBuf panics on BuildFor(nil) — the same sentinel
// technique columnplan_test.go's panicIfFullBuildBuf uses, applied to
// p33Leaf. Used by TestCountEdge_ChildKeyFromBufAvoidsFullBuild to prove
// childKeyFromBufAny actually calls the ChildKeyFromBuf override instead
// of falling back to BuildFor(nil) (Codexレビュー1回目・指摘2 関連テスト)。
type p33PanicOnFullBuildLeafBuf struct{ p33LeafBuf }

// p33FullBuildPanicMessage is the sentinel panic value BuildFor(nil) raises
// (Codexレビュー2回目・軽微指摘、2026-08-19): comparing the RECOVERED value
// against this exact string, on both the success and failure sides of
// TestCountEdge_ChildKeyFromBufAvoidsFullBuild, tells a genuine
// BuildFor(nil) call apart from an unrelated type-assertion panic that
// would otherwise also satisfy a bare `recover() != nil` check.
const p33FullBuildPanicMessage = "BuildFor(nil) must not be called when ChildKeyFromBuf is set"

func (b *p33PanicOnFullBuildLeafBuf) BuildFor(modelCols []Column) p33Leaf {
	if modelCols == nil {
		panic(p33FullBuildPanicMessage)
	}
	return b.p33LeafBuf.BuildFor(modelCols)
}

// TestCountEdge_ChildKeyFromBufAvoidsFullBuild is Codexレビュー1回目・
// 指摘2の関連テスト(2026-08-19、Codexレビュー2回目・軽微指摘で強化):
// p33MidLeafCountEdgeにChildKeyFromBufを設定した状態でchildKeyFromBufAny
// を直接呼び、(a) ChildKeyFromBufのオーバーライドが正しい値を返し、かつ
// BuildFor(nil)を一切呼ばないこと、(b) このオーバーライドを外すと
// childKeyFromBufAnyはBuildFor(nil)(全列構築)へ実際にフォールバック
// すること、の両方をDB無しで確認する。
//
// 2回目レビュー前は(a)に通常のp33LeafBuf(BuildFor(nil)でもパニックしない
// バッファ)を渡していたため、仮にランタイムが誤って
// 「オーバーライド呼び出しの前後にBuildFor(nil)も呼ぶ」ような壊れ方をしても
// 検出できなかった。(a)にも全列構築時パニックバッファを渡すことで、
// オーバーライドが「一度もBuildFor(nil)を経由しない」ことまで保証する。
// (b)も、recoverした値がp33FullBuildPanicMessageと一致することまで確認し、
// 型アサーション失敗等の無関係なpanicと取り違えないようにする。
func TestCountEdge_ChildKeyFromBufAvoidsFullBuild(t *testing.T) {
	edge := p33MidLeafCountEdge()

	// (a): オーバーライドが設定されていれば、全列構築時にパニックする
	// バッファを渡してもBuildFor(nil)を経由せず正しい値を返す。
	panicBuf := &p33PanicOnFullBuildLeafBuf{p33LeafBuf{id: 10, otherKey: 999}}
	got := edge.childKeyFromBufAny(bufAdapter[p33Leaf]{panicBuf})
	if len(got) != 1 || got[0].(int) != 999 {
		t.Fatalf("childKeyFromBufAny = %v, want [999]", got)
	}

	// (b): オーバーライドを外すと、BuildFor(nil)へ実際にフォールバック
	// することをpanicで確認する(count専用バッファに対する全列構築の
	// 危険性そのものの再現)。recoverした値がp33FullBuildPanicMessageと
	// 一致することまで確認し、無関係な型アサーションpanic等と混同しない。
	edge.ChildKeyFromBuf = nil
	panicBuf2 := &p33PanicOnFullBuildLeafBuf{p33LeafBuf{id: 10, otherKey: 999}}
	func() {
		defer func() {
			r := recover()
			if r != p33FullBuildPanicMessage {
				t.Fatalf("panic = %v, want %q", r, p33FullBuildPanicMessage)
			}
		}()
		edge.childKeyFromBufAny(bufAdapter[p33Leaf]{panicBuf2})
	}()
}

// --- テスト8: ネスト(孫)での制限 ---------------------------------

// TestExecIsland_RestrictedGrandparentAllowsUnrestrictedGrandchild is
// P-33 §8 テスト8: 祖父母(root)→親(mid、Except("id")で制限)→子(leaf、
// 無制限)の3段木で、execIsland の戻り値まで検証する。
func TestExecIsland_RestrictedGrandparentAllowsUnrestrictedGrandchild(t *testing.T) {
	midEdge := p33RootMidEdge(Except("id"))
	leafEdge := p33MidLeafEdge()
	tree := NewNode(midEdge, NewNode(leafEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 2 {
		t.Fatalf("sections: got %d, want 2 (mid, leaf)", len(plan.sections))
	}

	root := p33RootParts()
	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, mid.root_id, mid.other_key,
	//         leaf.id, leaf.other_key
	// (mid の childCols() は ColumnPlan.All()=[root_id, other_key] —
	// Except("id") で "id" だけが除外され、"other_key" は除外していない
	// ので自動的に Model へ残る。leaf は無制限なので table 順の
	// [id, other_key]。)
	exec := p28FakeExecutor{rows: &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0, 0, 0},
			{int64(1), nil, 0, 100, 999, 0, 0},
			{int64(2), nil, 0, 0, 0, 300, 999},
		}},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}

	if len(res.items[0]) != 1 || res.items[0][0].(p33Root).ID != 100 {
		t.Fatalf("root items = %v, want [{ID:100}]", res.items[0])
	}
	midSet := plan.sections[0].set
	if len(res.items[midSet]) != 1 {
		t.Fatalf("mid items: got %d, want 1", len(res.items[midSet]))
	}
	mid := res.items[midSet][0].(p33Mid)
	if mid.RootID != 100 || mid.OtherKey != 999 {
		t.Fatalf("mid = %+v, want RootID=100 OtherKey=999", mid)
	}
	if mid.ID != 0 {
		t.Fatalf("mid.ID = %d, want 0 (excluded by Except(\"id\"), never scanned into the model)", mid.ID)
	}
	leafSet := plan.sections[1].set
	if len(res.items[leafSet]) != 1 {
		t.Fatalf("leaf items: got %d, want 1", len(res.items[leafSet]))
	}
	leaf := res.items[leafSet][0].(p33Leaf)
	if leaf.ID != 300 || leaf.OtherKey != 999 {
		t.Fatalf("leaf = %+v, want ID=300 OtherKey=999", leaf)
	}
}

// --- テスト9: 制限親+count子の組み合わせ -----------------------------

// TestExecIsland_RestrictedParentWithCountChild is P-33 §8 テスト9: 親
// エッジに Except を指定し、その子に RelCount(CountEdge)を配置した木が
// validatePlan を通過し、execIsland が正しい件数を返すことを確認する
// (6.8節)。
func TestExecIsland_RestrictedParentWithCountChild(t *testing.T) {
	midEdge := p33RootMidEdge(Except("id"))
	countEdge := p33MidLeafCountEdge()
	tree := NewNode(midEdge, NewCountNode(countEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 || len(plan.counts) != 1 {
		t.Fatalf("plan = %+v, want 1 section (mid) + 1 count", plan)
	}

	root := p33RootParts()
	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, mid.root_id, mid.other_key,
	//         count.other_key(keyCol), _sl_cnt<set>
	exec := p28FakeExecutor{rows: &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0, 0, nil},
			{int64(1), nil, 0, 100, 999, 0, nil},
			{int64(2), nil, 0, 0, 0, 999, int64(3)},
		}},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}

	countSet := plan.counts[0].set
	got := res.counts[countSet]
	if len(got) != 1 || got[999] != 3 {
		t.Fatalf("counts[%d] = %v, want map[999:3]", countSet, got)
	}
}

// --- テスト13・14: 子エッジ fromCols の自動保護なし ------------------

// TestValidatePlan_ChildFromColsNamedInOnlyPasses is P-33 §8 テスト13
// (I3、6.10節通過側)。Codexレビュー1回目・指摘1(2026-08-19)で
// Except→Onlyへ差し替え: 設計書の意図(Onlyで子エッジのfromColsを明示する
// 経路)を、Exceptでは一度も踏んでいなかった(SelectOptions.Onlyが
// resolveEdgeColumnPlanへ正しく配線されている経路の正常系検証が0件
// だった)。other_keyを明示的にOnlyへ含め、root_id(自エッジのtoCols)は
// 自動的にInternalへ追加されることも確認する。
func TestValidatePlan_ChildFromColsNamedInOnlyPasses(t *testing.T) {
	// other_keyは子エッジのfromColsとして明示する。
	// root_idは自エッジのtoColsなのでInternalへ自動追加される。
	midEdge := p33RootMidEdge(Only("other_key"))
	leafEdge := p33MidLeafEdge()
	tree := NewNode(midEdge, NewNode(leafEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	columnPlan := resolved[0].edge.(Edge[p33Root, p33Mid]).ColumnPlan
	if len(columnPlan.Model) != 1 || columnPlan.Model[0].Name != "other_key" {
		t.Fatalf("Model = %v, want [other_key]", columnPlan.Model)
	}
	if len(columnPlan.Internal) != 1 || columnPlan.Internal[0].Name != "root_id" {
		t.Fatalf("Internal = %v, want [root_id]", columnPlan.Internal)
	}

	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}

	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := p33RootParts()
	parent := psql.RawQuery("SELECT id FROM p33_root")
	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	if !strings.Contains(sql, `"other_key"`) {
		t.Fatalf("other_key (子エッジの fromCols) がSQLに含まれていない:\n%s", sql)
	}
	if !strings.Contains(sql, `"root_id"`) {
		t.Fatalf("root_id (自エッジのtoCols、Internalへ自動追加) がSQLに含まれていない:\n%s", sql)
	}
}

// TestValidatePlan_ChildFromColsOmittedFromOnlyIsError is P-33 §8 テスト14
// (I3、6.10節拒否側)。Codexレビュー1回目・指摘1でExcept→Onlyへ差し替え
// (テスト13と同じ理由)。自エッジの結合キー(root_id)だけをOnlyに明示し、
// 子エッジが必要とするother_keyを意図的に省略した場合、
// *MissingRequiredColumnError になることを確認する。関数名は
// Codexレビュー2回目・注記2(2026-08-19)により、現在の検証内容(Onlyから
// の省略)に合わせて ExcludedIsError から OmittedFromOnlyIsError へ改名。
func TestValidatePlan_ChildFromColsOmittedFromOnlyIsError(t *testing.T) {
	// 自エッジの結合キーだけを明示し、子エッジが必要とする
	// other_keyを意図的に省略する。
	midEdge := p33RootMidEdge(Only("root_id"))
	leafEdge := p33MidLeafEdge()
	tree := NewNode(midEdge, NewNode(leafEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	err = validatePlan(resolved, nil)
	var want *MissingRequiredColumnError
	if !errors.As(err, &want) {
		t.Fatalf("validatePlan err = %v, want *MissingRequiredColumnError", err)
	}
	if want.RelName != "Mid" || want.ChildRelName != "Leaf" || want.Column != "other_key" {
		t.Fatalf("err fields = %+v, want RelName=Mid ChildRelName=Leaf Column=other_key", want)
	}
}

// --- テスト12: 発見Bのend-to-end ------------------------------------

type (
	p33SecretChild    struct{ ID, ParentID int }
	p33SecretChildBuf struct {
		id, parentID int
		secret       string
	}
)

func (b *p33SecretChildBuf) TargetsFor(cols []Column) []any {
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
			panic("p33SecretChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p33SecretChildBuf) BuildFor(modelCols []Column) p33SecretChild {
	if modelCols == nil {
		return p33SecretChild{ID: b.id, ParentID: b.parentID}
	}
	m := p33SecretChild{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "parent_id":
			m.ParentID = b.parentID
			// secret には対応するフィールドが無い: Except("secret") が
			// 効いていれば modelCols に secret は現れない。
		}
	}
	return m
}

// p33SecretEdge takes opts explicitly (Codexレビュー1回目・指摘1関連テスト、
// 2026-08-19) so callers can exercise both Only and Except through the same
// fixture — TestExecIsland_ExceptUnmentionedKeyReachesModel(テスト12)は
// Except("secret")を渡し、TestBuild_ResolvesOnlyExceptThroughProductionWiring
// の表形式版はOnly/Except両方を渡す。
func p33SecretEdge(opts SelectOptions) Edge[p33Root, p33SecretChild] {
	return Edge[p33Root, p33SecretChild]{
		RelName:  "SecretChild",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Options:  opts,
		Child: TableParts[p33SecretChild]{
			Table: "p33_secret_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "secret", DBType: "text"},
			},
			NewBuf: func() Buf[p33SecretChild] { return &p33SecretChildBuf{} },
		},
		ParentKey: func(p p33Root) []any { return []any{p.ID} },
		ChildKey:  func(c p33SecretChild) []any { return []any{c.ParentID} },
		ChildKeyFromBuf: func(b Buf[p33SecretChild]) []any {
			return []any{b.(*p33SecretChildBuf).parentID}
		},
		Stitch: func(p p33Root, cs []p33SecretChild) error { return nil },
	}
}

// TestExecIsland_ExceptUnmentionedKeyReachesModel is P-33 §8 テスト12
// (発見Bのend-to-end、C3): Except("secret") を指定した単一hopエッジを
// 実行し、除外していない結合キー列("parent_id")が (a) SQL に出る、
// (b) スキャンされる、(c) 子モデルのフィールドに値が入っている、の3点を
// 確認する。
func TestExecIsland_ExceptUnmentionedKeyReachesModel(t *testing.T) {
	e := p33SecretEdge(Except("secret"))
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	root := p33RootParts()
	parent := psql.RawQuery("SELECT id FROM p33_root")
	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	// (a) SQL に出る。
	if !strings.Contains(sql, `"parent_id"`) {
		t.Fatalf("(a) parent_id がSQLに含まれていない:\n%s", sql)
	}
	if strings.Contains(sql, `"secret"`) {
		t.Fatalf("Except(\"secret\") で除外したはずのsecretがSQLに含まれている:\n%s", sql)
	}

	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, child.id, child.parent_id
	// (ColumnPlan.Model=[id, parent_id]: secret 以外は自動的に Model へ)
	exec := p28FakeExecutor{rows: &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0},
			{int64(1), nil, 0, 500, 100},
		}},
	}}
	res, err := execIsland(context.Background(), exec, q, root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	set := plan.sections[0].set
	if len(res.items[set]) != 1 {
		t.Fatalf("child items: got %d, want 1", len(res.items[set]))
	}
	child := res.items[set][0].(p33SecretChild)
	// (b)(c): parent_id がスキャンされ、かつモデルのフィールドへ入っている。
	if child.ParentID != 100 {
		t.Fatalf("child.ParentID = %d, want 100 (parent_id must be scanned AND copied into the model)", child.ParentID)
	}
	if child.ID != 500 {
		t.Fatalf("child.ID = %d, want 500", child.ID)
	}
}

// TestExecIsland_OnlyRestrictsModelToNamedColumns is テスト12のOnly版
// (Copilot CLIレビュー・Nit1、2026-08-19、ユーザー承認済み): Only("id")を
// 指定した単一hopエッジを実行し、(a) SQLにidが出てsecretが出ない、
// (b) 指定したidがスキャンされモデルへ入る、(c) 指定していない
// parent_id(結合キー、Internalへ自動追加)はスキャンされるがモデルには
// 入らずゼロ値のままである、の3点をexecIslandレベルで確認する。
// ExceptとOnlyはNewColumnPlanへの経路が異なる(exceptモード/onlyモード
// 分岐)ため、Exceptだけでなくonlyモード固有の「指定外の列がゼロ値に
// なる」動作もexecIslandまで通して確認する。
func TestExecIsland_OnlyRestrictsModelToNamedColumns(t *testing.T) {
	e := p33SecretEdge(Only("id"))
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	root := p33RootParts()
	parent := psql.RawQuery("SELECT id FROM p33_root")
	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	// (a) SQL確認: idが出てsecretは出ない。
	if !strings.Contains(sql, `."id"`) {
		t.Fatalf("(a) idがSQLに含まれていない:\n%s", sql)
	}
	if strings.Contains(sql, `"secret"`) {
		t.Fatalf("Only(\"id\")で除外したはずのsecretがSQLに含まれている:\n%s", sql)
	}

	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, child.id, child.parent_id
	// (ColumnPlan.Model=[id], Internal=[parent_id]: exceptモードのテスト12
	// と同じ2列がスキャンされるが、Model/Internalの割り当てが逆になる)
	exec := p28FakeExecutor{rows: &p28ColsRows{
		cols: cols,
		p28FakeRows: &p28FakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0},
			{int64(1), nil, 0, 500, 100},
		}},
	}}
	res, err := execIsland(context.Background(), exec, q, root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	set := plan.sections[0].set
	if len(res.items[set]) != 1 {
		t.Fatalf("child items: got %d, want 1", len(res.items[set]))
	}
	child := res.items[set][0].(p33SecretChild)
	// (b): Only("id")で指定したidはスキャンされモデルへ入る。
	if child.ID != 500 {
		t.Fatalf("child.ID = %d, want 500 (Only(\"id\")で指定したので値が入るはず)", child.ID)
	}
	// (c): 指定していないparent_id(Internal)はスキャンされるがモデルには
	// 入らずゼロ値のまま。
	if child.ParentID != 0 {
		t.Fatalf("child.ParentID = %d, want 0 (Internal列なのでモデルには入らない)", child.ParentID)
	}
}

// --- run.go の配線そのものを通す検証(advisor指摘) ---------------
//
// 上記のテストは全て resolveColumnPlans を手で呼んでから
// planIsland/composeIsland/execIsland に直接渡しており、run.go の
// All/Build が実際に resolveColumnPlans を呼び出す配線そのものは
// 一度も踏んでいなかった(advisorレビューで指摘)。run.go の該当2箇所
// (All の [0.5]、Build の先頭)を削除しても上記のテストは全て緑のまま
// 通ってしまう false green だったため、Build を直接呼ぶテストを追加する。

// TestBuild_ResolvesOnlyExceptThroughProductionWiring is run.go の Build
// が resolveColumnPlans を実際に呼び出すことを検証する: 手で
// resolveColumnPlans を呼ばず、Build(ctx, parentQuery, parts, nodes...)
// だけを呼び出し、Options が SQL に反映されていることを確認する。
// DB接続・fake executor 不要(Build は SQL を組むだけで実行しない)。
//
// Codexレビュー1回目・指摘1(関連テスト、2026-08-19)により表形式化:
// 元はExceptケースのみだったため、OnlyがresolveColumnPlansへ正しく
// 配線されている経路をBuild経由では一度も検証していなかった。
func TestBuild_ResolvesOnlyExceptThroughProductionWiring(t *testing.T) {
	tests := []struct {
		name string
		opts SelectOptions
	}{
		{name: "Only", opts: Only("id")},
		{name: "Except", opts: Except("secret")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := p33SecretEdge(tc.opts)
			parent := psql.RawQuery("SELECT id FROM p33_root")

			built, err := Build(context.Background(), parent, p33RootTableParts(), NewNode(e))
			if err != nil {
				t.Fatalf("Build: %v (run.goのBuildがresolveColumnPlansを呼んでいなければ、Optionsが一切効かずcomposeIslandはColumnPlanゼロ値のまま無制限SQLを組んで成功してしまう可能性もあるため、エラーにならないこと自体は無制限フォールバックとの区別にならない — 以下のSQL内容チェックが本体)", err)
			}
			// Copilot CLIレビュー・Minor1(2026-08-19、ユーザー承認済み)対応:
			// `"id"`単体での照合は`"parent_id"`にも部分一致してしまい
			// (strings.Containsは部分文字列一致のため)、id自体がSELECTに
			// 出ていなくてもparent_idの存在だけで通過してしまう偽陰性が
			// あった。テーブル修飾形式`."id"`(例: `"p33_root"."id"`や
			// `_sl_t"."id"`)で照合し、`."parent_id"`とは区別する。
			if !strings.Contains(built.SQL, `."id"`) {
				t.Fatalf("Build().SQL に table-qualified な \"id\" が含まれていない — resolveColumnPlansがBuildの中で呼ばれていない可能性:\n%s", built.SQL)
			}
			if !strings.Contains(built.SQL, `"parent_id"`) {
				t.Fatalf("Build().SQL に parent_id が含まれていない — resolveColumnPlansがBuildの中で呼ばれていない可能性:\n%s", built.SQL)
			}
			if strings.Contains(built.SQL, `"secret"`) {
				t.Fatalf("Only/Exceptが効いていれば secret は絶対にSQLに出ないはず(resolveColumnPlansがBuildの中で呼ばれていない場合、ColumnPlanがゼロ値のままとなり無制限SQL=secretを含むSQLになる):\n%s", built.SQL)
			}
		})
	}
}

// TestAll_RejectsMultiHopColumnRestrictionThroughProductionWiring is
// run.go の All が resolveColumnPlans を実際に呼び出すことを、エラー
// 経路側から検証する: p05TwoHopEdge(多段エッジ)に Only を指定した木を
// 手で resolveColumnPlans を呼ばずに直接 All(...) へ渡し、
// *MultiHopColumnRestrictionUnsupportedError が返ることを確認する。
// All の [0.5] が削除されていれば、この多段エッジは(ColumnPlanが
// ゼロ値のまま)validatePlan もエラーにせず素通りしてしまう
// (validateMultiHopEdgeはColumnPlanの状態に関係なく通る木のため)。
func TestAll_RejectsMultiHopColumnRestrictionThroughProductionWiring(t *testing.T) {
	e := p05TwoHopEdge()
	e.Options = Only("label")
	parent := psql.RawQuery("SELECT id FROM p05_parents")

	_, err := All[p05Parent](context.Background(), p28FakeExecutor{rows: &p28FakeRows{}}, parent, p05ParentParts(), NewNode(e))
	var want *MultiHopColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("All(...) err = %v, want *MultiHopColumnRestrictionUnsupportedError (Allの中でresolveColumnPlansが呼ばれていない可能性)", err)
	}
}
