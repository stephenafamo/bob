package sectionload

// P-05 Task 2: 単段(pre-P-05)の合成SQLを golden として固定する。
//
// §13.5-1(P-05設計書)は「単段のSQLはバイト単位で不変にできる」と主張してい
// る: 単段Edgeは flat な FromCols/ToCols/Where しか持たず、sides() がそれを
// 長さ1のSideへ正規化するので、buildEdgeSelectのjoins/carried列は常に空にな
// り、レンダリング結果は現行と同一になるはずだ、という設計上の論証である。
// この主張は実測で裏付けなければ主張のままになる(issue P-05 §14 Task 2の
// 「なぜ最初にやるか」)。このファイルはTask 3〜9の各コミットの直後に必ず比較
// する golden を、ランタイム変更に着手する前に採取する。
//
// fixtureは新設せず、既存の手書きfixtureを流用する(p28_thenkey_test.goの
// p28Root/p28Mid/p28Leaf・p28RootMidEdge()/p28MidLeafEdge()、count_test.go
// のp04CountParent/p04RowEdge()/p04CountEdge())。
//
// 採取する木は5本:
//  1. single_edge:    単段1本(root -> mid)
//  2. siblings:       兄弟2本(root -> mid, root -> mid の2セクション)
//  3. nested:         入れ子(root -> mid -> leaf、Thenなし)
//  4. then_boundary:  Then(...)境界を含む木(root -> mid -[Then]-> leaf)。
//     Build()は最初の島しかレンダリングしない(run.go:339-341)ので、この
//     goldenが検証するのは「Then境界を切ったことで最初の島(root -> mid)
//     のSQLがどう変わるか」であって、Then側(leaf)のSQLではない。leafが
//     Thenで括られるとmidのhasKidsがfalseになりCTEが消えるため、golden 3
//     (nested)とは意図的に異なるSQLになる。
//  5. with_count:     countノード混在の木(root -> Rows(行), root -> Children
//     (count))。p04CountParentを共通の親型として使う。
//
// いずれのfixtureもWhere/SelectOptionsを持たないため、bob.Buildが返す args
// は常に0個になる — この前提が崩れたら(何かのfixtureでargsが出たら)テスト
// が失敗するので、そのこと自体が「単段の挙動が変わった」シグナルになる。

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// -update-p05-baseline: 立てるとtestdata/p05_baseline/*.sqlを現在の
// buildEdgeSelect/composeIsland出力で上書きする。立てなければ既存の
// goldenと完全一致で比較する。
var updateP05Baseline = flag.Bool(
	"update-p05-baseline",
	false,
	"rewrite testdata/p05_baseline/*.sql golden files from the current sectionload SQL builder output",
)

// p05BaselineCase describes one golden comparison: name (-> testdata file
// name) と、そのSQL/argsを生成するクロージャ。
type p05BaselineCase struct {
	name  string
	build func(t *testing.T) (sql string, args []any)
}

func p05BaselineRootParts() TableParts[p28Root] {
	return TableParts[p28Root]{
		Table:   "p28_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p28Root] { return &p28RootBuf{} },
	}
}

func p05BaselineCases() []p05BaselineCase {
	return []p05BaselineCase{
		{
			name: "single_edge",
			build: func(t *testing.T) (string, []any) {
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM p28_root"),
					p05BaselineRootParts(),
					NewNode(p28RootMidEdge()),
				)
				if err != nil {
					t.Fatalf("Build (single_edge): %v", err)
				}
				return built.SQL, built.Args
			},
		},
		{
			name: "siblings",
			build: func(t *testing.T) (string, []any) {
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM p28_root"),
					p05BaselineRootParts(),
					NewNode(p28RootMidEdge()),
					NewNode(p28RootMidEdge()),
				)
				if err != nil {
					t.Fatalf("Build (siblings): %v", err)
				}
				return built.SQL, built.Args
			},
		},
		{
			name: "nested",
			build: func(t *testing.T) (string, []any) {
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM p28_root"),
					p05BaselineRootParts(),
					NewNode(p28RootMidEdge(), NewNode(p28MidLeafEdge())),
				)
				if err != nil {
					t.Fatalf("Build (nested): %v", err)
				}
				return built.SQL, built.Args
			},
		},
		{
			name: "then_boundary",
			build: func(t *testing.T) (string, []any) {
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM p28_root"),
					p05BaselineRootParts(),
					NewNode(p28RootMidEdge(), Then(NewNode(p28MidLeafEdge()))),
				)
				if err != nil {
					t.Fatalf("Build (then_boundary): %v", err)
				}
				return built.SQL, built.Args
			},
		},
		{
			name: "with_count",
			build: func(t *testing.T) (string, []any) {
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM p04_count_parents"),
					p04CountParts(),
					NewNode(p04RowEdge()),
					NewCountNode(p04CountEdge()),
				)
				if err != nil {
					t.Fatalf("Build (with_count): %v", err)
				}
				return built.SQL, built.Args
			},
		},
	}
}

func p05BaselineGoldenPath(name string) string {
	return filepath.Join("testdata", "p05_baseline", name+".sql")
}

// runP05BaselineCase compares (or, with -update-p05-baseline, records) one
// case's SQL against its golden file, and asserts its arg count is 0 (every
// fixture here is Where/Options-free — see file doc comment).
func runP05BaselineCase(t *testing.T, c p05BaselineCase) {
	t.Helper()
	sqlText, args := c.build(t)
	path := p05BaselineGoldenPath(c.name)

	if *updateP05Baseline {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
		}
		if err := os.WriteFile(path, []byte(sqlText), 0o644); err != nil {
			t.Fatalf("write golden %s: %v", path, err)
		}
		t.Logf("wrote golden %s (%d args)", path, len(args))
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read golden %s: %v (run `go test ./sectionload -run TestP05Baseline -update-p05-baseline` to create it)", path, err)
	}
	if sqlText != string(want) {
		t.Fatalf("SQL for %q changed from baseline:\n--- golden (%s) ---\n%s\n--- got ---\n%s",
			c.name, path, string(want), sqlText)
	}
	if len(args) != 0 {
		t.Fatalf("%q: this fixture carries no Where/Options, so args must stay empty; got %d: %v", c.name, len(args), args)
	}
}

// TestP05Baseline is Task 2's golden fixation. Every Task from 3 onward
// (Side/Sides/Carried, the JOIN-chain SQL builder, the childKeyFrom
// dispatch, etc.) must re-run this test unmodified and see all 5 cases
// still PASS — a single-hop edge normalizes to a length-1 Sides internally
// and must render byte-identical SQL (§13.5-1 / §14 stop condition 3).
func TestP05Baseline(t *testing.T) {
	for _, c := range p05BaselineCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			runP05BaselineCase(t, c)
		})
	}
}
