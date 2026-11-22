package sectionload

// Golden files pinning the composed SQL of single-hop trees.
//
// The claim is that single-hop SQL stays byte-identical when multi-hop
// support exists: a single-hop Edge only has flat FromCols/ToCols/Where,
// sides() normalizes it into a length-1 Side, so buildEdgeSelect's
// joins/carried columns are always empty and the rendered SQL is the same
// as before. A claim like that needs a measurement behind it, so these
// goldens are captured before the multi-hop runtime changes and compared
// after each subsequent change.
//
// No new fixtures are introduced; existing hand-written ones are reused
// (thenKeyRoot/thenKeyMid/thenKeyLeaf and thenKeyRootMidEdge()/thenKeyMidLeafEdge() from
// thenkey_test.go, countParent/rowEdge()/countEdge() from
// count_test.go).
//
// Five trees are captured:
//  1. single_edge:    one single-hop edge (root -> mid)
//  2. siblings:       two siblings (root -> mid, root -> mid as two sections)
//  3. nested:         nested (root -> mid -> leaf, no Then)
//  4. then_boundary:  a tree with a Then(...) boundary (root -> mid -[Then]-> leaf).
//     Build() renders only the first island, so this golden checks how the
//     SQL of the first island (root -> mid) changes once the Then boundary
//     cuts the tree, not the SQL of the Then side (leaf). With leaf wrapped
//     in Then, mid's hasKids becomes false and the CTE disappears, so the
//     SQL intentionally differs from golden 3 (nested).
//  5. with_count:     a tree mixing a count node (root -> Rows (rows),
//     root -> Children (count)). countParent is used as the shared
//     parent type.
//
// None of the fixtures has Where/SelectOptions, so bob.Build always returns
// zero args. If that premise breaks (some fixture starts producing args), the
// test fails, which is itself a signal that single-hop behavior changed.

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// -update-baseline: when set, overwrites testdata/baseline/*.sql with
// the current buildEdgeSelect/composeIsland output. When unset, the output is
// compared for an exact match against the existing goldens.
var updateMultiHopBaseline = flag.Bool(
	"update-baseline",
	false,
	"rewrite testdata/baseline/*.sql golden files from the current sectionload SQL builder output",
)

// multiHopBaselineCase describes one golden comparison: name (-> testdata file
// name) and a closure that produces its SQL/args.
type multiHopBaselineCase struct {
	name  string
	build func(t *testing.T) (sql string, args []any)
}

func multiHopBaselineRootParts() TableParts[thenKeyRoot] {
	return TableParts[thenKeyRoot]{
		Table:   "tk_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[thenKeyRoot] { return &thenKeyRootBuf{} },
	}
}

func multiHopBaselineCases() []multiHopBaselineCase {
	return []multiHopBaselineCase{
		{
			name: "single_edge",
			build: func(t *testing.T) (string, []any) {
				t.Helper()
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM tk_root"),
					multiHopBaselineRootParts(),
					NewNode(thenKeyRootMidEdge()),
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
				t.Helper()
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM tk_root"),
					multiHopBaselineRootParts(),
					NewNode(thenKeyRootMidEdge()),
					NewNode(thenKeyRootMidEdge()),
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
				t.Helper()
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM tk_root"),
					multiHopBaselineRootParts(),
					NewNode(thenKeyRootMidEdge(), NewNode(thenKeyMidLeafEdge())),
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
				t.Helper()
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM tk_root"),
					multiHopBaselineRootParts(),
					NewNode(thenKeyRootMidEdge(), Then(NewNode(thenKeyMidLeafEdge()))),
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
				t.Helper()
				built, err := Build(
					context.Background(),
					psql.RawQuery("SELECT id FROM count_parents"),
					countParts(),
					NewNode(rowEdge()),
					NewCountNode(countEdge()),
				)
				if err != nil {
					t.Fatalf("Build (with_count): %v", err)
				}
				return built.SQL, built.Args
			},
		},
	}
}

func multiHopBaselineGoldenPath(name string) string {
	return filepath.Join("testdata", "baseline", name+".sql")
}

// runMultiHopBaselineCase compares (or, with -update-baseline, records) one
// case's SQL against its golden file, and asserts its arg count is 0 (every
// fixture here is Where/Options-free — see file doc comment).
func runMultiHopBaselineCase(t *testing.T, c multiHopBaselineCase) {
	t.Helper()
	sqlText, args := c.build(t)
	path := multiHopBaselineGoldenPath(c.name)

	if *updateMultiHopBaseline {
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
		t.Fatalf("read golden %s: %v (run `go test ./sectionload -run TestMultiHopBaseline -update-baseline` to create it)", path, err)
	}
	if sqlText != string(want) {
		t.Fatalf("SQL for %q changed from baseline:\n--- golden (%s) ---\n%s\n--- got ---\n%s",
			c.name, path, string(want), sqlText)
	}
	if len(args) != 0 {
		t.Fatalf("%q: this fixture carries no Where/Options, so args must stay empty; got %d: %v", c.name, len(args), args)
	}
}

// TestMultiHopBaseline is the golden fixation of the baseline shapes. Every later change
// (Side/Sides/Carried, the JOIN-chain SQL builder, the childKeyFrom
// dispatch, etc.) must re-run this test unmodified and see all 5 cases
// still PASS — a single-hop edge normalizes to a length-1 Sides internally
// and must render byte-identical SQL.
func TestMultiHopBaseline(t *testing.T) {
	for _, c := range multiHopBaselineCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			runMultiHopBaselineCase(t, c)
		})
	}
}
