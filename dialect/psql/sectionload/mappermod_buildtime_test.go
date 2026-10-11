package sectionload

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

type (
	p06Model struct{ ID int }
	p06Buf   struct{ id int }
)

func (b *p06Buf) TargetsFor(cols []Column) []any       { return []any{&b.id} }
func (b *p06Buf) BuildFor(modelCols []Column) p06Model { return p06Model{ID: b.id} }

func p06Parts() TableParts[p06Model] {
	return TableParts[p06Model]{
		Table:   "p06_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p06Model] { return &p06Buf{} },
	}
}

type p06CountingExecutor struct{ queries int }

func (e *p06CountingExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	e.queries++
	return nil, errors.New("p06: reached the database")
}

func (e *p06CountingExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, fmt.Errorf("p06: ExecContext not implemented")
}

// p06DummyMapperMod: scan v0.9.0で scan.MapperMod = func(context.Context,
// cols) (BeforeFunc, AfterMod) であり、cols は package scan 内で
// `type cols = []string` という型エイリアスなので、外部からは []string を
// 直接書ける(確認済み: scan@v0.9.0/mapper.go:10-12, mod.go:8-11)。
// 中身は件数検査用のno-opでよい。
func p06DummyMapperMod(ctx context.Context, c []string) (scan.BeforeFunc, scan.AfterMod) {
	return nil, nil
}

// TestCheckMapperMods_DoesNotDetectSameCountSubstitution (P-13、Codex
// レビュー1回目・指摘2、2026-08-20): checkMapperMods(run.go)の1:1判定
// (防御強化案(a)適用後は宣言数・MapperMod数・PreloadColumns総数の3つの
// 一致)は、由来(そのMapperModが本当にPreloadCount由来か)までは検証
// できない。P-13適用後にSetMapperModsで、count preloadとは無関係な別の
// MapperMod(p06DummyMapperMod)へ同数差し替えられ、かつPreloadColumns
// 自体の総数は変えずにおいた場合(=差し替えがMapperModだけに留まり、
// SQLのSELECT列自体はそのまま残っているケース)、現状のcheckMapperModsは
// 拒否できず許可してしまう(declared=1・GetMapperMods()==1・
// PreloadColumns総数=1のまま)ことを特性テストとして固定する。
//
// これは仕様上のバグではなく、設計書§6.5が「認識済みの残存リスク」として
// 明記している経路そのもの(bob側APIでMapperModの由来を型的に証明する
// 手段が無いため — 調査メモ設問3で確定済み)。将来、bob側APIがMapperMod
// に識別子付きメタデータを持たせる拡張(§6.5「将来の閉じ方」参照)を実装
// すれば、このテストは「拒否されるべき」へ反転する — その時点でこの
// テスト自体を書き換えること。
func TestCheckMapperMods_DoesNotDetectSameCountSubstitution(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("p06_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	// PreloadColumnsは1件のまま(count preloadが元々足していたはずの
	// 相関サブクエリ列がSetMapperMods後もSELECT自体には残っている状態を
	// 模擬)。中身は判定に使われないのでダミーでよい。
	inner.AppendPreloadSelect(psql.Raw("1"))
	// count preload由来ではない、無関係なMapperModを1件だけ付ける
	// (SetMapperModsによる意図的な差し替えを模擬 — 差し替え自体の実施は
	// テストの本質ではなく、「差し替え後、由来を問わず1件だけ存在する」
	// という最終状態を作れれば十分)。
	inner.AppendMapperMod(p06DummyMapperMod)

	// declared側は「PreloadCountが1件宣言されている」という状態を模擬する
	// (実際のPreloadCount適用は、この関数の入力である宣言数だけを見て
	// 判定するため、declaredの中身自体は判定に影響しない)。
	declared := []declaredCountPreload{{RelName: "TaskComments", Alias: "__count_TaskComments"}}

	err := checkMapperMods(parent, declared, len(inner.PreloadColumns))
	if err != nil {
		t.Fatalf(
			"現状のcheckMapperMods(防御強化案(a)適用後)は、宣言数・"+
				"MapperMod数・PreloadColumns総数の3つの一致のみで由来を"+
				"検証しないため、count以外のMapperModへの同数差し替え"+
				"(かつPreloadColumns総数は変えない場合)を拒否できない — "+
				"このテストはその制約自体を固定するものであり、"+
				"エラーが返るようになったなら反転が起きている(§6.5参照): %v",
			err,
		)
	}
}

// TestCheckMapperMods_RejectsWhenPreloadColumnCountMismatches (P-13、
// Codexレビュー1回目・防御強化案(a)、2026-08-20): 宣言数とMapperMod数は
// 一致するが、PreloadColumns自体の総数が宣言数と食い違う場合(利用者が
// count宣言とは無関係な別のPreloadColumnsを追加で持っている状態を模擬)
// は、防御強化案(a)により拒否されることを固定する。この分岐が無ければ
// TestCheckMapperMods_DoesNotDetectSameCountSubstitutionと同じ理由で
// 素通りしてしまうケースであり、防御強化案(a)自体が機能していることの
// 直接の裏付けになる。
func TestCheckMapperMods_RejectsWhenPreloadColumnCountMismatches(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("p06_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	// PreloadColumnsを2件にする(宣言数1件とは食い違う数)。
	inner.AppendPreloadSelect(psql.Raw("1"), psql.Raw("2"))
	inner.AppendMapperMod(p06DummyMapperMod)

	declared := []declaredCountPreload{{RelName: "TaskComments", Alias: "__count_TaskComments"}}

	err := checkMapperMods(parent, declared, len(inner.PreloadColumns))
	if err == nil {
		t.Fatal("expected *UnsupportedMapperModsError (PreloadColumns総数が宣言数と食い違う), got nil")
	}
	var mmErr *UnsupportedMapperModsError
	if !errors.As(err, &mmErr) {
		t.Fatalf("expected *UnsupportedMapperModsError, got %T: %v", err, err)
	}
	if mmErr.Count != 1 {
		t.Fatalf("Count: got %d, want 1", mmErr.Count)
	}
}

// TestAllRejectsMapperModAddedByContextualMod: 実物のPreloader.Applyと
// 同じ経路(AppendContextualModFuncの中でMapperModを追加)を最小再現し、
// ビルド時検査が拒否することを固定する(P-06 round 2、判断撤回の実装)。
func TestAllRejectsMapperModAddedByContextualMod(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("p06_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	inner.AppendContextualModFunc(
		func(ctx context.Context, q *dialect.SelectQuery) (context.Context, error) {
			q.AppendMapperMod(p06DummyMapperMod)
			return ctx, nil
		},
	)

	exec := &p06CountingExecutor{}
	_, err := All(context.Background(), exec, parent, p06Parts())
	if err == nil {
		t.Fatal("expected UnsupportedMapperModsError, got nil")
	}
	var mmErr *UnsupportedMapperModsError
	if !errors.As(err, &mmErr) {
		t.Fatalf("expected *UnsupportedMapperModsError, got %T: %v", err, err)
	}
	if mmErr.Count != 1 {
		t.Fatalf("Count: got %d, want 1", mmErr.Count)
	}
	if exec.queries != 0 {
		t.Fatalf("must be rejected before any DB call, got %d queries", exec.queries)
	}
}

// TestBuildTimeMapperModCheckSkipsNonModdableParent: psql.RawQuery の実型
// bob.BaseQuery[expr.Clause] は既知の非contextual queryとして許可され、
// UncheckableParentQueryError にならず従来どおり executor まで進む
// (P-06 round 3 で逸脱1を撤回した後の許可リストの回帰テスト)。
func TestBuildTimeMapperModCheckSkipsNonModdableParent(t *testing.T) {
	exec := &p06CountingExecutor{}
	_, err := All(context.Background(), exec, psql.RawQuery("SELECT 1"), p06Parts())
	var mmErr *UnsupportedMapperModsError
	if errors.As(err, &mmErr) {
		t.Fatalf("non-moddable parent must be skipped, not rejected: %v", err)
	}
	if exec.queries != 1 {
		t.Fatalf("executor should have been reached exactly once, got %d", exec.queries)
	}
}

// TestBuildTimeMapperModCheckRestoresContextualMods verifies restoration by
// behavior: All temporarily wraps a contextual mod that adds a MapperMod and
// rejects it, but a direct bob.Build after All must run the original unwrapped
// mod successfully. If the restore assignment is removed, the direct build
// encounters the leaked checking wrapper and returns UnsupportedMapperModsError.
func TestBuildTimeMapperModCheckRestoresContextualMods(
	t *testing.T,
) {
	ctx := context.Background()
	parent := psql.Select(sm.From(psql.Raw("p06_parents")))

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			inner = q
		},
	))

	var modRuns []string
	inner.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			modRuns = append(modRuns, "first")
			return ctx, nil
		},
	)
	inner.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			modRuns = append(modRuns, "mapper")
			built.AppendMapperMod(p06DummyMapperMod)
			return ctx, nil
		},
	)

	wantRuns := []string{"first", "mapper"}
	for call := 1; call <= 2; call++ {
		modRuns = nil
		_, err := All(
			ctx,
			&p06CountingExecutor{},
			parent,
			p06Parts(),
		)

		var mmErr *UnsupportedMapperModsError
		if !errors.As(err, &mmErr) {
			t.Fatalf(
				"call %d: expected *UnsupportedMapperModsError from All, got %T: %v",
				call,
				err,
				err,
			)
		}
		if mmErr.Count != 1 {
			t.Fatalf(
				"call %d: MapperMod count: got %d, want 1",
				call,
				mmErr.Count,
			)
		}
		if !slices.Equal(modRuns, wantRuns) {
			t.Fatalf(
				"call %d: wrapped contextual-mod runs: got %v, want %v",
				call,
				modRuns,
				wantRuns,
			)
		}
		if got := len(inner.ContextualModdable.Mods); got != len(wantRuns) {
			t.Fatalf(
				"call %d: restored contextual-mod count: got %d, want %d",
				call,
				got,
				len(wantRuns),
			)
		}

		modRuns = nil
		if _, _, err := bob.Build(ctx, parent); err != nil {
			t.Fatalf(
				"call %d: direct build after All must use the restored contextual mod, got %T: %v",
				call,
				err,
				err,
			)
		}
		if !slices.Equal(modRuns, wantRuns) {
			t.Fatalf(
				"call %d: direct build contextual-mod runs: got %v, want %v",
				call,
				modRuns,
				wantRuns,
			)
		}
	}
}

type p06WrappedQuery struct {
	bob.Query
}

func TestP06RejectsWrappedQueryThatHidesContextualMods(t *testing.T) {
	inner := psql.Select(sm.From(psql.Raw("p06_parents")))

	var selectQuery *dialect.SelectQuery
	inner.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	selectQuery.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			built.AppendMapperMod(p06DummyMapperMod)
			return ctx, nil
		},
	)

	exec := &p06CountingExecutor{}
	_, err := All(
		context.Background(),
		exec,
		p06WrappedQuery{Query: inner},
		p06Parts(),
	)

	var checkErr *UncheckableParentQueryError
	if !errors.As(err, &checkErr) {
		t.Fatalf(
			"expected *UncheckableParentQueryError, got %T: %v",
			err,
			err,
		)
	}
	if exec.queries != 0 {
		t.Fatalf(
			"wrapped query must be rejected before the database, got %d queries",
			exec.queries,
		)
	}
}

func TestP06RejectsMapperModBeforeLaterContextualModClearsIt(t *testing.T) {
	parent := psql.Select(sm.From(psql.Raw("p06_parents")))

	var selectQuery *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) {
			selectQuery = q
		},
	))
	selectQuery.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			built.AppendMapperMod(p06DummyMapperMod)
			return ctx, nil
		},
	)
	selectQuery.AppendContextualModFunc(
		func(
			ctx context.Context,
			built *dialect.SelectQuery,
		) (context.Context, error) {
			built.SetMapperMods()
			return ctx, nil
		},
	)

	exec := &p06CountingExecutor{}
	_, err := All(
		context.Background(),
		exec,
		parent,
		p06Parts(),
	)

	var mmErr *UnsupportedMapperModsError
	if !errors.As(err, &mmErr) {
		t.Fatalf(
			"expected *UnsupportedMapperModsError, got %T: %v",
			err,
			err,
		)
	}
	if mmErr.Count != 1 {
		t.Fatalf("Count: got %d, want 1", mmErr.Count)
	}
	if exec.queries != 0 {
		t.Fatalf(
			"transient mapper mod must be rejected before the database, got %d queries",
			exec.queries,
		)
	}
}
