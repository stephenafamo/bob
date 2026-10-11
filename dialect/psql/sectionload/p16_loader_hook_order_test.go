// このテストは M-01(P-16)のフォローアップとして、Codexレビューが指摘した
// 「未検証で残る点」(親クエリに直付けした loader と、登録済みの after-select
// hook を実際に組み合わせた順序検証がまだ無い)を埋める否定側検証である。
//
// sectionload.All は親クエリに直付けされた loader を、親の AfterQueryHook
// より前に実行し終えていなければならない(bob@v0.49.0 exec.go:178-190 の
// ローダー→フックの順)。さもないと、loader が書き込んだフィールドを hook が
// 読めず、loader のエラーが hook の実行を止められない。
//
// DB には接続しない。exec.QueryContext を横取りしてハードコードした行データを
// 返す最小のフェイク Executor を使う(bob.Executor は
// github.com/stephenafamo/bob@v0.49.0 exec.go:29 で
// scan.Queryer(QueryContext) + ExecContext と定義されている)。
package sectionload_test

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"

	sectionload "github.com/stephenafamo/bob/dialect/psql/sectionload"
)

// --- テスト専用の最小モデル -------------------------------------------------

type p16FakeParent struct {
	ID int64
	// LoadedAt is written by the parent-attached loader. If the hook can
	// read "loaded" here, the loader ran before the hook.
	LoadedAt string
}

// p16FakeParentSlice implements bob.HookableType, mirroring how generated
// code's <Table>Slice types carry AfterQueryHook.
type p16FakeParentSlice []*p16FakeParent

var (
	// p16AfterSelectHooks mirrors generated code's <Table>.AfterSelectHooks
	// registry (e.g. Tasks.AfterSelectHooks, tasks.bob.go:1053-1060):
	// AfterQueryHook dispatches to it via bob.Hooks.RunHooks instead of
	// containing hook logic directly, so this test exercises the same
	// AppendHooks/RunHooks path real registered hooks use.
	p16AfterSelectHooks *bob.Hooks[p16FakeParentSlice, bob.SkipModelHooksKey]
	p16HookTrace        []string
	p16HookSawValue     string
	p16HookErr          error
)

func (s p16FakeParentSlice) AfterQueryHook(
	ctx context.Context,
	exec bob.Executor,
	queryType bob.QueryType,
) error {
	var err error
	switch queryType {
	case bob.QueryTypeSelect:
		_, err = p16AfterSelectHooks.RunHooks(ctx, exec, s)
	}
	return err
}

// p16RegisterAfterSelectHook installs a fresh AfterSelectHooks registration,
// mirroring how generated code registers hooks via AppendHooks. Call this
// before sectionload.All in each test.
func p16RegisterAfterSelectHook() {
	p16AfterSelectHooks = new(bob.Hooks[p16FakeParentSlice, bob.SkipModelHooksKey])
	p16AfterSelectHooks.AppendHooks(
		func(
			ctx context.Context,
			_ bob.Executor,
			parents p16FakeParentSlice,
		) (context.Context, error) {
			p16HookTrace = append(p16HookTrace, "hook")
			if len(parents) > 0 {
				p16HookSawValue = parents[0].LoadedAt
			}
			return ctx, p16HookErr
		},
	)
}

// --- TableParts (dbsections 生成コード相当) --------------------------------

type p16FakeParentBuf struct {
	ID int64
}

func (b *p16FakeParentBuf) TargetsFor(cols []sectionload.Column) []any { return []any{&b.ID} }
func (b *p16FakeParentBuf) BuildFor(modelCols []sectionload.Column) *p16FakeParent {
	return &p16FakeParent{ID: b.ID}
}

var p16FakeParentParts = sectionload.TableParts[*p16FakeParent]{
	Schema: "",
	Table:  "p16_fake_parents",
	Columns: []sectionload.Column{
		{Name: "id", DBType: "bigint"},
	},
	NewBuf:  func() sectionload.Buf[*p16FakeParent] { return &p16FakeParentBuf{} },
	AsSlice: func(v []*p16FakeParent) any { return p16FakeParentSlice(v) },
}

// --- 親クエリに loader を直付けするラッパー ---------------------------------
//
// bob.Loadable(load.go:12)を満たし、テスト用 loader を返すラッパー。
// psql.Select が返す bob.BaseQuery[*dialect.SelectQuery] を embed する(旧版は
// psql.RawQuery の bob.BaseQuery[expr.Clause] を embed していたが、P-06の
// ビルド時MapperMod検査(run.go installBuildTimeMapperModCheck)は
// Apply(...bob.Mod[*dialect.SelectQuery]) を公開する具体型だけを安全側とし
// 動的型が一致しないラッパーは UncheckableParentQueryError で拒否する —
// bob.BaseQuery[expr.Clause] を embed した名前付き構造体はこの型チェックに
// 一致しないため、P-02統合後にこのテストが失敗するようになった。GetLoaders
// はこのラッパーで上書きし、親クエリに直付けされた loader を明示的に返す)。
type p16LoadableQuery struct {
	bob.BaseQuery[*dialect.SelectQuery]
	loaders []bob.Loader
}

func (q p16LoadableQuery) GetLoaders() []bob.Loader { return q.loaders }

var (
	_ bob.Loadable = psql.RawQuery("SELECT 1")
	_ bob.Loadable = p16LoadableQuery{}
)

func p16NewLoadableParentQuery(loader bob.Loader) p16LoadableQuery {
	return p16LoadableQuery{
		BaseQuery: psql.Select(sm.From(psql.Raw("p16_fake_parents")), sm.Columns(psql.Raw("id"))),
		loaders:   []bob.Loader{loader},
	}
}

// --- フェイク Executor -------------------------------------------------------

type p16FakeExecutor struct {
	calls int
}

func (f *p16FakeExecutor) QueryContext(_ context.Context, query string, _ ...any) (scan.Rows, error) {
	f.calls++
	if f.calls != 1 {
		return nil, fmt.Errorf("p16FakeExecutor: unexpected QueryContext call #%d: %s", f.calls, query)
	}
	// メイン島の唯一の行。列レイアウトは固定(_sl_set, _sl_ord, <root cols>)。
	return p16NewFakeRows(
		[]string{"_sl_set", "_sl_ord", "id"},
		[][]any{
			{int64(0), sql.NullInt64{Int64: 1, Valid: true}, int64(1)},
		},
	), nil
}

func (f *p16FakeExecutor) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	return nil, fmt.Errorf("p16FakeExecutor: unexpected ExecContext call: %s", query)
}

// p16FakeRows is the minimal scan.Rows (interfaces.go, scan@v0.9.0)
// implementation: Scan/Columns/Next/Close/Err over a hardcoded [][]any row
// set.
type p16FakeRows struct {
	cols []string
	data [][]any
	pos  int
}

func p16NewFakeRows(cols []string, data [][]any) *p16FakeRows {
	return &p16FakeRows{cols: cols, data: data, pos: -1}
}

func (r *p16FakeRows) Columns() ([]string, error) { return r.cols, nil }

func (r *p16FakeRows) Next() bool {
	r.pos++
	return r.pos < len(r.data)
}

func (r *p16FakeRows) Scan(dest ...any) error {
	row := r.data[r.pos]
	if len(dest) != len(row) {
		return fmt.Errorf("p16FakeRows: dest count %d != row width %d", len(dest), len(row))
	}
	for i, d := range dest {
		switch d := d.(type) {
		case *int64:
			v, ok := row[i].(int64)
			if !ok {
				return fmt.Errorf("p16FakeRows: col %d: want int64, got %T", i, row[i])
			}
			*d = v
		case *sql.NullInt64:
			v, ok := row[i].(sql.NullInt64)
			if !ok {
				return fmt.Errorf("p16FakeRows: col %d: want sql.NullInt64, got %T", i, row[i])
			}
			*d = v
		default:
			return fmt.Errorf("p16FakeRows: unsupported scan dest type %T at col %d", d, i)
		}
	}
	return nil
}

func (r *p16FakeRows) Close() error { return nil }
func (r *p16FakeRows) Err() error   { return nil }

// --- 本体 --------------------------------------------------------------------

// TestParentLoaderRunsBeforeParentHook asserts that by the time the
// parent's AfterQueryHook fires, a loader attached directly to the parent
// query has already run, and that the hook observes the loader's side
// effect on the shared model slice.
func TestParentLoaderRunsBeforeParentHook(t *testing.T) {
	p16HookTrace = nil
	p16HookSawValue = ""
	p16HookErr = nil
	p16RegisterAfterSelectHook()
	ctx := context.Background()
	exec := &p16FakeExecutor{}

	loader := bob.LoaderFunc(func(_ context.Context, _ bob.Executor, retrieved any) error {
		p16HookTrace = append(p16HookTrace, "loader")
		parents, ok := retrieved.(p16FakeParentSlice)
		if !ok {
			return fmt.Errorf("loader: unexpected retrieved type %T", retrieved)
		}
		for _, p := range parents {
			p.LoadedAt = "loaded"
		}
		return nil
	})

	got, err := sectionload.All(
		ctx,
		exec,
		p16NewLoadableParentQuery(loader),
		p16FakeParentParts,
	)
	if err != nil {
		t.Fatalf("sectionload.All: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("親の件数が違う: got %d want 1", len(got))
	}
	if want := []string{"loader", "hook"}; len(p16HookTrace) != 2 || p16HookTrace[0] != want[0] || p16HookTrace[1] != want[1] {
		t.Fatalf("loaderとhookの実行順が違う: got %v want %v", p16HookTrace, want)
	}
	if p16HookSawValue != "loaded" {
		t.Fatalf("AfterQueryHook 実行時点で loader の書き込み(LoadedAt)が見えていない: got %q", p16HookSawValue)
	}
	if got[0].LoadedAt != "loaded" {
		t.Fatalf("最終結果に loader の書き込みが反映されていない: %+v", got[0])
	}
}

// TestParentLoaderErrorSuppressesParentHook asserts that when the
// parent-attached loader returns an error, the parent's AfterQueryHook
// does not run at all — matching bob.Allx's short-circuit behavior at
// exec.go:178-190(loader エラーはフックより前に短絡する)。
func TestParentLoaderErrorSuppressesParentHook(t *testing.T) {
	p16HookTrace = nil
	p16HookErr = nil
	p16RegisterAfterSelectHook()
	ctx := context.Background()
	exec := &p16FakeExecutor{}
	wantErr := errors.New("loader boom")

	loader := bob.LoaderFunc(func(_ context.Context, _ bob.Executor, _ any) error {
		p16HookTrace = append(p16HookTrace, "loader")
		return wantErr
	})

	_, err := sectionload.All(
		ctx,
		exec,
		p16NewLoadableParentQuery(loader),
		p16FakeParentParts,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("sectionload.All: got err %v, want wrapping %v", err, wantErr)
	}
	if len(p16HookTrace) != 1 || p16HookTrace[0] != "loader" {
		t.Fatalf("loaderエラー時のトレースが違う(hookが実行された可能性): got %v want [loader]", p16HookTrace)
	}
}

// TestParentHookErrorAfterLoaderRan asserts that a hook error occurs only
// after the loader has already run: the loader is never skipped merely
// because the hook is destined to fail.
func TestParentHookErrorAfterLoaderRan(t *testing.T) {
	p16HookTrace = nil
	wantErr := errors.New("hook boom")
	p16HookErr = wantErr
	p16RegisterAfterSelectHook()
	defer func() { p16HookErr = nil }()
	ctx := context.Background()
	exec := &p16FakeExecutor{}

	loader := bob.LoaderFunc(func(_ context.Context, _ bob.Executor, _ any) error {
		p16HookTrace = append(p16HookTrace, "loader")
		return nil
	})

	_, err := sectionload.All(
		ctx,
		exec,
		p16NewLoadableParentQuery(loader),
		p16FakeParentParts,
	)
	if !errors.Is(err, wantErr) {
		t.Fatalf("sectionload.All: got err %v, want wrapping %v", err, wantErr)
	}
	if len(p16HookTrace) != 2 || p16HookTrace[0] != "loader" || p16HookTrace[1] != "hook" {
		t.Fatalf("hookエラー時もloaderが先に実行されているはず: got %v want [loader hook]", p16HookTrace)
	}
}
