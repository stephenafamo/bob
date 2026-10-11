package sectionload

// P-01: 子が無い/親キーが nil の to-one 関連も「ロード済み」にする。
//
// stitchInto は SQL を一切扱わない純粋関数(parents/children は既に
// メモリ上にある any スライス)なので、このテストは
// OPUS-EXECUTION-GUIDE.md §2 の「純粋関数の単体テスト」フォールバックを
// 使う。tools/bob-sectionload モジュールには DB ドライバ/testdb 接続の
// インフラが元から無い(go.mod に sql/pgx 系の直接依存が無く、
// pkg/dbsections だけが openDB を持つ)ため、GUIDE §4 の
// 「testdb の transaction 内で CREATE TEMP TABLE」経路は使わず、
// この経路(stitchInto を直接呼ぶ)で否定側・回帰を両方記録する。

import (
	"fmt"
	"testing"
)

// p01Loaded / p01Rels / p01Parent / p01Child は、生成モデルの
// R / R.Loaded / typed-nil Preload パターン(pkg/dbmodels/tasks.bob.go の
// Preload 実装を参照)を模した test 専用 model。

type p01Loaded struct {
	Owner bool
	Kids  bool
}

type p01Rels struct {
	Owner  *p01Child
	Kids   []*p01Child
	Loaded p01Loaded
}

type p01Parent struct {
	ID      int
	OwnerID *int // nil == nullable FK 未設定
	R       p01Rels
}

type p01Child struct {
	ID int
	// RefID is a separate FK-like field so the to-many edge's ChildKey can
	// match on something other than the child's own PK (which the to-one
	// edge's ChildKey uses). Without this, both edges would group by the
	// same field and the to-many "multiple children" fixture below would
	// never match its parent.
	RefID int
}

// p01PreloadOwner は any の型アサーション境界だけを模す。
// 生成テンプレートおよび実生成 Preload の検証は生成物テスト
// (pkg/dbsections/sectionload_p01_generated_test.go)で行う。
func p01PreloadOwner(p *p01Parent, retrieved any) error {
	rel, ok := retrieved.(*p01Child)
	if !ok {
		return fmt.Errorf("p01Parent cannot load %T as %q", retrieved, "Owner")
	}
	p.R.Owner = rel
	p.R.Loaded.Owner = true
	return nil
}

// p01ToOneEdge は、修正後の to-one Stitch(templates/table/01_edges.go.tpl)
// と同じ形: 子が無ければ typed nil を any 境界(p01PreloadOwner)経由で渡し、
// R.Loaded は常に true。
func p01ToOneEdge() Edge[*p01Parent, *p01Child] {
	return Edge[*p01Parent, *p01Child]{
		RelName: "Owner",
		ToMany:  false,
		ParentKey: func(p *p01Parent) []any {
			if p.OwnerID == nil {
				return nil
			}
			return []any{*p.OwnerID}
		},
		ChildKey: func(c *p01Child) []any { return []any{c.ID} },
		Stitch: func(p *p01Parent, children []*p01Child) error {
			var related *p01Child
			if len(children) > 0 {
				related = children[0]
			}
			var retrieved any = related // typed nil, exactly like the generated Stitch
			return p01PreloadOwner(p, retrieved)
		},
	}
}

// p01ToManyEdge は既存の to-many Stitch と同じ形(変更なし): 0 件でも
// 空スライスを渡して R.Loaded を立てる。
func p01ToManyEdge() Edge[*p01Parent, *p01Child] {
	return Edge[*p01Parent, *p01Child]{
		RelName: "Kids",
		ToMany:  true,
		ParentKey: func(p *p01Parent) []any {
			if p.OwnerID == nil {
				return nil
			}
			return []any{*p.OwnerID}
		},
		ChildKey: func(c *p01Child) []any { return []any{c.RefID} },
		Stitch: func(p *p01Parent, children []*p01Child) error {
			p.R.Kids = children
			p.R.Loaded.Kids = true
			return nil
		},
	}
}

// --- 否定側(P-01 Important-2): untyped nil は any 境界で拒否される ---
//
// p01PreloadOwner (= 実際の生成 Preload の to-one 分岐と同じ型アサーション)
// に untyped nil を直接渡すと、typed nil (*p01Child)(nil) とは違い
// ok=false でエラーになることを確認する。これは「生成テンプレートが誤って
// var related *p01Child を経由せず nil を直接渡すよう壊れた」という
// regression class を検出できることの直接的な証拠(P-01 Important-2、
// Codex review 2026-08-09)。
func TestP01PreloadOwner_UntypedNilIsRejected(t *testing.T) {
	p := &p01Parent{ID: 1}
	err := p01PreloadOwner(p, nil) // untyped nil, not (*p01Child)(nil)
	if err == nil {
		t.Fatalf("p01PreloadOwner(p, nil) = nil error, want an error (untyped nil must fail the type assertion, exactly like the real generated Preload)")
	}
	if p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should stay false when the any-boundary rejects the value: got true")
	}
}

func TestP01PreloadOwner_TypedNilIsAccepted(t *testing.T) {
	p := &p01Parent{ID: 1}
	var related *p01Child // typed nil — exactly what the generated Stitch passes
	if err := p01PreloadOwner(p, related); err != nil {
		t.Fatalf("p01PreloadOwner(p, (*p01Child)(nil)) = %v, want nil error (typed nil must pass the type assertion)", err)
	}
	if p.R.Owner != nil {
		t.Fatalf("R.Owner = %#v, want nil", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner should be true once a typed nil is accepted: got false")
	}
}

// --- 否定側(a): 親キーが nil(nullable FK 未設定)の to-one ---

func TestStitchIntoToOneNullKey(t *testing.T) {
	e := p01ToOneEdge()
	var calls int
	origStitch := e.Stitch
	e.Stitch = func(p *p01Parent, children []*p01Child) error {
		calls++
		return origStitch(p, children)
	}
	parents := []any{&p01Parent{ID: 1, OwnerID: nil}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if p.R.Owner != nil {
		t.Fatalf("R.Owner は nil のままであるべき: got %#v", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner は true であるべき(P-01): got false")
	}
	if calls != 1 {
		t.Fatalf("Stitch should be called exactly once for to-one with nil key: got %d", calls)
	}
}

// --- 否定側(b): 親キーは非nilだが対応する子が0件の to-one ---

func TestStitchIntoToOneNonNullKeyNoChild(t *testing.T) {
	e := p01ToOneEdge()
	ownerID := 99 // どの child にも存在しないキー
	parents := []any{&p01Parent{ID: 1, OwnerID: &ownerID}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if p.R.Owner != nil {
		t.Fatalf("R.Owner は nil のままであるべき: got %#v", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner は true であるべき(P-01): got false")
	}
}

// --- 回帰: to-one で子が見つかった場合(既存挙動を変えない) ---

func TestStitchIntoToOneWithChild(t *testing.T) {
	e := p01ToOneEdge()
	ownerID := 5
	parents := []any{&p01Parent{ID: 1, OwnerID: &ownerID}}
	children := []any{&p01Child{ID: 5}}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if p.R.Owner == nil || p.R.Owner.ID != 5 {
		t.Fatalf("R.Owner は id=5 の子であるべき: got %#v", p.R.Owner)
	}
	if !p.R.Loaded.Owner {
		t.Fatalf("R.Loaded.Owner は true であるべき: got false")
	}
}

// --- 回帰: to-many で子が0件(既存から動いていた経路) ---

func TestStitchIntoToManyZeroChildren(t *testing.T) {
	e := p01ToManyEdge()
	ownerID := 1
	parents := []any{&p01Parent{ID: 1, OwnerID: &ownerID}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if len(p.R.Kids) != 0 {
		t.Fatalf("R.Kids は空であるべき: got %#v", p.R.Kids)
	}
	if !p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids は true であるべき: got false")
	}
}

// --- 回帰: to-many で複数の子(既存から動いていた経路) ---

func TestStitchIntoToManyMultipleChildren(t *testing.T) {
	e := p01ToManyEdge()
	ownerID := 1
	parents := []any{&p01Parent{ID: 1, OwnerID: &ownerID}}
	children := []any{&p01Child{ID: 10, RefID: 1}, &p01Child{ID: 11, RefID: 1}}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if len(p.R.Kids) != 2 {
		t.Fatalf("R.Kids は2件であるべき: got %d", len(p.R.Kids))
	}
	if !p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids は true であるべき: got false")
	}
}

// --- 回帰: 親キーが nil の to-many は Stitch を一切呼ばない(旧挙動維持) ---
//
// P-01 Important-1(Codex review, 2026-08-09)で訂正: このテストは元々
// 「nil キーでも to-many は Loaded=true になる」ことを検証していたが、
// これは issue 自身の §5「to-manyは従来どおり空スライスを渡すため、
// 意味を変えません」と矛盾する誤った期待値だった(§10.3 に「to-one/
// to-manyとも全親で1回呼ばれる」という別の記述もあり、issue内で記述が
// 割れていた)。修正後の stitchInto は nil 親キーで to-many なら Stitch
// 自体を呼ばない(pre-P-01 と同じ)ため、R.Kids/R.Loaded.Kids はゼロ値の
// ままになる。現行スキーマに ToMany:true かつ ParentKey が nil を返し
// うる実エッジは0件(機械確認済み)のため、この経路は今のところ理論上の
// ものだが、runtime の一般契約としてはこちらが正しい。
func TestStitchIntoToManyNullKeySkipsStitchEntirely(t *testing.T) {
	e := p01ToManyEdge()
	var calls int
	origStitch := e.Stitch
	e.Stitch = func(p *p01Parent, children []*p01Child) error {
		calls++
		return origStitch(p, children)
	}
	parents := []any{&p01Parent{ID: 1, OwnerID: nil}}

	if _, err := stitchInto(e, parents, nil, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if p.R.Kids != nil {
		t.Fatalf("R.Kids はゼロ値(nil)のままであるべき(Stitch自体が呼ばれない): got %#v", p.R.Kids)
	}
	if p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids は false のままであるべき(nilキーのto-manyはStitchを呼ばない、pre-P-01と同じ): got true")
	}
	if calls != 0 {
		t.Fatalf("Stitch should not be called for to-many with nil key: got %d", calls)
	}
}

// --- 複合キーのマッチング確認 ---
//
// mapKey は複合キーを [N]any 配列にパックするため、全要素が一致したときだけ
// マッチする。部分一致(キーの一部が合っている)では別のグループになる。
// Codex review 2026-08-09 Minor 1: 複合キー経路の回帰テストを追加。

// p01ToManyCompositeKeyEdge は複合キー(2要素) の to-many Edge。
// 親キーと子キーともに 2 要素のスライスを返し、mapKey で [2]any に
// パックされる。全要素が一致したときだけ同じグループになる。
func p01ToManyCompositeKeyEdge() Edge[*p01Parent, *p01Child] {
	return Edge[*p01Parent, *p01Child]{
		RelName: "Kids",
		ToMany:  true,
		ParentKey: func(p *p01Parent) []any {
			if p.OwnerID == nil {
				return nil
			}
			// 複合キー: [親ID, 親の OwnerID]
			return []any{p.ID, *p.OwnerID}
		},
		ChildKey: func(c *p01Child) []any {
			// 複合キー: [子ID, 子RefID]
			return []any{c.ID, c.RefID}
		},
		Stitch: func(p *p01Parent, children []*p01Child) error {
			p.R.Kids = children
			p.R.Loaded.Kids = true
			return nil
		},
	}
}

// TestStitchIntoToManyCompositeKeyNoMatch は、複合キーの部分一致は
// マッチしないことを確認する。親キー[1,7]と子キー[2,7]は最初の要素が
// 異なるため異なるグループになり、子が届かない。
func TestStitchIntoToManyCompositeKeyNoMatch(t *testing.T) {
	e := p01ToManyCompositeKeyEdge()
	var calls int
	origStitch := e.Stitch
	e.Stitch = func(p *p01Parent, children []*p01Child) error {
		calls++
		return origStitch(p, children)
	}

	parentID := 7
	parents := []any{&p01Parent{ID: 1, OwnerID: &parentID}}
	children := []any{&p01Child{ID: 2, RefID: 7}}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p := parents[0].(*p01Parent)
	if len(p.R.Kids) != 0 {
		t.Fatalf("R.Kids は空であるべき(部分一致のキーは来ない): got %#v", p.R.Kids)
	}
	if !p.R.Loaded.Kids {
		t.Fatalf("R.Loaded.Kids は true であるべき(to-many は常に Stitch が呼ばれる): got false")
	}
	if calls != 1 {
		t.Fatalf("Stitch should be called exactly once: got %d", calls)
	}
}

// TestStitchIntoCompositeKeyMatchesAllComponents は、複合キーが全要素で
// 一致したときだけマッチ、部分一致は別のグループになることを確認する。
// 親1[1,7] は子1[1,7] に届き、子3[1,8] には届かない。
// 親2[2,7] は子2[2,7] に届く。
func TestStitchIntoCompositeKeyMatchesAllComponents(t *testing.T) {
	e := p01ToManyCompositeKeyEdge()

	parent1ID := 7
	parent2ID := 7
	parents := []any{
		&p01Parent{ID: 1, OwnerID: &parent1ID},
		&p01Parent{ID: 2, OwnerID: &parent2ID},
	}
	children := []any{
		&p01Child{ID: 1, RefID: 7}, // 親1[1,7] にマッチ
		&p01Child{ID: 2, RefID: 7}, // 親2[2,7] にマッチ
		&p01Child{ID: 1, RefID: 8}, // [1,8] — 親1[1,7] と部分一致だが別グループ
	}

	if _, err := stitchInto(e, parents, children, nil, nil, nil); err != nil {
		t.Fatalf("stitchInto: %v", err)
	}

	p1 := parents[0].(*p01Parent)
	if len(p1.R.Kids) != 1 {
		t.Fatalf("parent1[1,7] should have 1 child [1,7]: got %d children %v", len(p1.R.Kids), p1.R.Kids)
	}
	if p1.R.Kids[0].ID != 1 || p1.R.Kids[0].RefID != 7 {
		t.Fatalf("parent1[1,7] should match child[1,7]: got %#v", p1.R.Kids[0])
	}
	if !p1.R.Loaded.Kids {
		t.Fatalf("parent1 R.Loaded.Kids should be true: got false")
	}

	p2 := parents[1].(*p01Parent)
	if len(p2.R.Kids) != 1 {
		t.Fatalf("parent2[2,7] should have 1 child [2,7]: got %d children %v", len(p2.R.Kids), p2.R.Kids)
	}
	if p2.R.Kids[0].ID != 2 || p2.R.Kids[0].RefID != 7 {
		t.Fatalf("parent2[2,7] should match child[2,7]: got %#v", p2.R.Kids[0])
	}
	if !p2.R.Loaded.Kids {
		t.Fatalf("parent2 R.Loaded.Kids should be true: got false")
	}
}
