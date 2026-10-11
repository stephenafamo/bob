// このテストは P-24(.claude/tmp/07-企画書-対応必要機能の全件洗い出し.md の問題台帳)の
// 否定側検証である。sectionload.All は Then(...) 境界([6]、このブランチではP-17で
// ステップ番号が1つ繰り上がっている)を親のAfterQueryHook([7])より前に実行し終えて
// いなければならない。さもないと、
// 親モデルに登録された AfterSelectHook が Then(...) でロードされるはずの
// 関連フィールド(R)がまだ縫い合わされていない不完全なグラフを見てしまう
// (bob@v0.49.0 exec.go:178-190 のローダー→フックの順)。
//
// DB には接続しない。exec.QueryContext を横取りしてハードコードした行データを
// 返す最小のフェイク Executor を使う(bob.Executor は
// github.com/stephenafamo/bob@v0.49.0 exec.go:29 で
// scan.Queryer(QueryContext) + ExecContext と定義されている)。
package sectionload_test

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/scan"

	sectionload "github.com/stephenafamo/bob/dialect/psql/sectionload"
)

// --- テスト専用の最小モデル -------------------------------------------------
//
// pkg/dbsections の生成コード(例: tasks.bob.go の TaskParts、
// task_comments.bob.go の Task() エッジ定義)と同じ形を、DB スキーマなしで
// 手書みしたもの。実物は models.Task のような bob 生成モデルだが、ここでは
// R フィールドと AfterQueryHook だけを持つ最小構造体で足りる。

type fakeChild struct {
	ID int64
}

type fakeParent struct {
	ID      int64
	ChildID int64
	R       fakeParentRels
}

type fakeParentRels struct {
	Child *fakeChild
}

// fakeParentSlice implements bob.HookableType, mirroring how generated
// code's <Table>Slice types carry AfterQueryHook (models.TaskSlice 相当)。
type fakeParentSlice []*fakeParent

// hookObservations records, per element of the slice the hook was called
// with, whether R.Child was already stitched at hook-fire time. This is
// the thing Step 2 of the brief asks the test to check.
var hookObservations []bool

func (s fakeParentSlice) AfterQueryHook(_ context.Context, _ bob.Executor, _ bob.QueryType) error {
	hookObservations = make([]bool, len(s))
	for i, p := range s {
		hookObservations[i] = p.R.Child != nil
	}
	return nil
}

// --- TableParts / Edge (dbsections 生成コード相当) --------------------------

type fakeParentBuf struct {
	ID      int64
	ChildID sql.NullInt64
}

func (b *fakeParentBuf) TargetsFor(cols []sectionload.Column) []any { return []any{&b.ID, &b.ChildID} }

func (b *fakeParentBuf) BuildFor(modelCols []sectionload.Column) *fakeParent {
	p := &fakeParent{ID: b.ID}
	if b.ChildID.Valid {
		p.ChildID = b.ChildID.Int64
	}
	return p
}

type fakeChildBuf struct {
	ID int64
}

func (b *fakeChildBuf) TargetsFor(cols []sectionload.Column) []any { return []any{&b.ID} }
func (b *fakeChildBuf) BuildFor(modelCols []sectionload.Column) *fakeChild {
	return &fakeChild{ID: b.ID}
}

var fakeParentParts = sectionload.TableParts[*fakeParent]{
	Schema: "",
	Table:  "fake_parents",
	Columns: []sectionload.Column{
		{Name: "id", DBType: "bigint"},
		{Name: "child_id", DBType: "bigint"},
	},
	NewBuf:  func() sectionload.Buf[*fakeParent] { return &fakeParentBuf{} },
	AsSlice: func(v []*fakeParent) any { return fakeParentSlice(v) },
}

var fakeChildParts = sectionload.TableParts[*fakeChild]{
	Schema: "",
	Table:  "fake_children",
	Columns: []sectionload.Column{
		{Name: "id", DBType: "bigint"},
	},
	NewBuf: func() sectionload.Buf[*fakeChild] { return &fakeChildBuf{} },
}

// fakeParentChildEdge is a to-one edge, equivalent in shape to
// task_comments.bob.go's taskCommentSections.Task() edge.
var fakeParentChildEdge = sectionload.Edge[*fakeParent, *fakeChild]{
	RelName:  "Child",
	FromCols: []string{"child_id"},
	ToCols:   []string{"id"},
	ToMany:   false,
	Child:    fakeChildParts,
	ParentKey: func(p *fakeParent) []any {
		return []any{p.ChildID}
	},
	ChildKey: func(c *fakeChild) []any {
		return []any{c.ID}
	},
	Stitch: func(p *fakeParent, children []*fakeChild) error {
		if len(children) == 0 {
			return nil
		}
		p.R.Child = children[0]
		return nil
	},
}

// --- フェイク Executor -------------------------------------------------------
//
// bob.Executor (exec.go:29) は scan.Queryer(QueryContext(ctx, query string,
// args ...any) (scan.Rows, error)) と ExecContext を要求する。sectionload.All
// が実行するクエリは呼び出し順が決まっている:
//
//	1回目 — メイン島(親テーブルのみ、Then(...) はここに含まれない)
//	2回目 — runThen が発行する Then 境界のクエリ(子テーブル)
//
// フェイクは呼び出し回数でどちらのクエリかを判定し、対応する固定行を返す。
// 実際の SQL 文字列は検査しない(このテストの関心は実行順であって SQL 生成では
// ない)。
type fakeExecutor struct {
	calls   int
	queries []string
}

func (f *fakeExecutor) QueryContext(_ context.Context, query string, _ ...any) (scan.Rows, error) {
	f.calls++
	f.queries = append(f.queries, query)
	switch f.calls {
	case 1:
		// メイン島: fake_parents の1行(id=1, child_id=42)。
		return newFakeRows(
			[]string{"_sl_set", "_sl_ord", "id", "child_id"},
			[][]any{
				{int64(0), sql.NullInt64{Int64: 1, Valid: true}, int64(1), sql.NullInt64{Int64: 42, Valid: true}},
			},
		), nil
	case 2:
		// Then 境界: fake_children の1行(id=42)。
		return newFakeRows(
			[]string{"_sl_set", "_sl_ord", "id"},
			[][]any{
				{int64(0), sql.NullInt64{}, int64(42)},
			},
		), nil
	default:
		return nil, fmt.Errorf("fakeExecutor: unexpected QueryContext call #%d: %s", f.calls, query)
	}
}

func (f *fakeExecutor) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	return nil, fmt.Errorf("fakeExecutor: unexpected ExecContext call: %s", query)
}

// fakeRows is the minimal scan.Rows (interfaces.go, scan@v0.9.0) implementation:
// Scan/Columns/Next/Close/Err over a hardcoded [][]any row set.
type fakeRows struct {
	cols []string
	data [][]any
	pos  int
}

func newFakeRows(cols []string, data [][]any) *fakeRows {
	return &fakeRows{cols: cols, data: data, pos: -1}
}

func (r *fakeRows) Columns() ([]string, error) { return r.cols, nil }

func (r *fakeRows) Next() bool {
	r.pos++
	return r.pos < len(r.data)
}

func (r *fakeRows) Scan(dest ...any) error {
	row := r.data[r.pos]
	if len(dest) != len(row) {
		return fmt.Errorf("fakeRows: dest count %d != row width %d", len(dest), len(row))
	}
	for i, d := range dest {
		switch d := d.(type) {
		case *int64:
			v, ok := row[i].(int64)
			if !ok {
				return fmt.Errorf("fakeRows: col %d: want int64, got %T", i, row[i])
			}
			*d = v
		case *sql.NullInt64:
			v, ok := row[i].(sql.NullInt64)
			if !ok {
				return fmt.Errorf("fakeRows: col %d: want sql.NullInt64, got %T", i, row[i])
			}
			*d = v
		default:
			return fmt.Errorf("fakeRows: unsupported scan dest type %T at col %d", d, i)
		}
	}
	return nil
}

func (r *fakeRows) Close() error { return nil }
func (r *fakeRows) Err() error   { return nil }

// --- 本体 --------------------------------------------------------------------

// TestThenBoundaryBeforeParentHook asserts that by the time the parent's
// AfterQueryHook fires, a Then(...) boundary's relation has already been
// stitched into R. Before the P-24 fix, run.go's [6](parent hook) ran
// before [7](Then boundaries) — this branch's own numbering, one lower
// than P-24's original branch since P-17 removed a separate pre-Build
// step — so this failed: the hook saw R.Child == nil.
func TestThenBoundaryBeforeParentHook(t *testing.T) {
	hookObservations = nil
	ctx := context.Background()
	exec := &fakeExecutor{}

	parentQuery := psql.RawQuery(`SELECT id, child_id FROM fake_parents`)

	got, err := sectionload.All(
		ctx,
		exec,
		parentQuery,
		fakeParentParts,
		sectionload.Then(sectionload.NewNode(fakeParentChildEdge)),
	)
	if err != nil {
		t.Fatalf("sectionload.All: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("親の件数が違う: got %d want 1", len(got))
	}
	if exec.calls != 2 {
		t.Fatalf("QueryContext の呼び出し回数が違う: got %d want 2 (メイン島 + Then境界)", exec.calls)
	}
	if len(hookObservations) != 1 {
		t.Fatalf("AfterQueryHook が呼ばれた要素数が違う: got %d want 1 (hook自体が発火していない可能性)", len(hookObservations))
	}
	if !hookObservations[0] {
		t.Fatalf("AfterQueryHook 実行時点で Then(...) 境界の R.Child が未設定だった: " +
			"[6](親hook)が[7](Then境界)より先に実行されている(P-24)")
	}
	if got[0].R.Child == nil || got[0].R.Child.ID != 42 {
		t.Fatalf("最終結果の R.Child が期待どおりに縫い合わされていない: %+v", got[0].R)
	}
}
