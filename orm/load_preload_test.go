package orm

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/aarondl/opt"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/scan"
)

type testPreloadChild struct {
	ID   int64            `db:"id"`
	Name sql.Null[string] `db:"name"`
}

func (c *testPreloadChild) Preload(name string, rel any) error {
	return fmt.Errorf("child has no relationship %q", name)
}

type testPreloadChildSlice []*testPreloadChild

type testPreloadParent struct {
	ID    int64             `db:"id"`
	Child *testPreloadChild `db:"-"`
}

func (p *testPreloadParent) Preload(name string, rel any) error {
	if name != "Child" {
		return fmt.Errorf("parent has no relationship %q", name)
	}

	child, ok := rel.(*testPreloadChild)
	if !ok {
		return fmt.Errorf("cannot load %T as %q", rel, name)
	}

	p.Child = child
	return nil
}

type testNameable struct{ name, alias string }

func (n testNameable) NameExpr() bob.Expression { return expr.Quote(n.name) }
func (n testNameable) Alias() string            { return n.alias }

type testPreloadQuery struct{}

func (testPreloadQuery) AppendLoader(...bob.Loader)         {}
func (testPreloadQuery) AppendMapperMod(scan.MapperMod)     {}
func (testPreloadQuery) AppendJoin(clause.Join)             {}
func (testPreloadQuery) AppendPreloadSelect(columns ...any) {}

// testPreloadChildMapper mirrors the shape of the generated
// <table>ScanMapperNullable: prefix-filtered column resolution once per
// query, scanning into a nullable buffer, nil on an all-NULL row.
func testPreloadChildMapper(prefix string) scan.Mapper[*testPreloadChild] {
	return func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (*testPreloadChild, error)) {
		type childBuf struct {
			id   sql.Null[int64]
			name sql.Null[string]
		}
		type target struct {
			idx int
			dst func(*childBuf) any
		}

		targets := make([]target, 0, 2)
		for i, col := range cols {
			name, ok := strings.CutPrefix(col, prefix)
			if !ok {
				continue
			}
			switch name {
			case "id":
				targets = append(targets, target{i, func(b *childBuf) any { return &b.id }})
			case "name":
				targets = append(targets, target{i, func(b *childBuf) any { return &b.name }})
			}
		}

		buf := new(childBuf)
		return func(row *scan.Row) (any, error) {
				for _, t := range targets {
					row.ScheduleScanByIndex(t.idx, t.dst(buf))
				}
				return buf, nil
			}, func(link any) (*testPreloadChild, error) {
				buf := link.(*childBuf)
				if !buf.id.Valid && !buf.name.Valid {
					return nil, nil
				}

				o := new(testPreloadChild)
				if buf.id.Valid {
					o.ID = buf.id.V
				}
				o.Name = buf.name
				return o, nil
			}
	}
}

// testRows implements scan.Rows on static data, mimicking database/sql's
// scan behavior (sql.Scanner destinations first, ConvertAssign otherwise).
type testRows struct {
	cols []string
	rows [][]any
	idx  int
}

func (r *testRows) Columns() ([]string, error) { return r.cols, nil }
func (r *testRows) Close() error               { return nil }
func (r *testRows) Err() error                 { return nil }

func (r *testRows) Next() bool {
	r.idx++
	return r.idx <= len(r.rows)
}

func (r *testRows) Scan(dest ...any) error {
	row := r.rows[r.idx-1]
	for i, d := range dest {
		if sc, ok := d.(sql.Scanner); ok {
			if err := sc.Scan(row[i]); err != nil {
				return err
			}
			continue
		}
		if err := opt.ConvertAssign(d, row[i]); err != nil {
			return err
		}
	}
	return nil
}

// buildTestPreloadScanner builds the full row scanner the way bob does for a
// preloading query: the parent's base mapper extended with the preloader's
// mapper mod. The child join is aliased "c" so column names are predictable.
func buildTestPreloadScanner(mapper PreloadMapper[*testPreloadChild]) scan.Mapper[*testPreloadParent] {
	rel := PreloadRel[bob.Expression]{
		Name: "Child",
		Sides: []PreloadSide[bob.Expression]{{
			From:        testNameable{name: "parents", alias: "parents"},
			To:          testNameable{name: "children", alias: "children"},
			FromColumns: []string{"child_id"},
			ToColumns:   []string{"id"},
		}},
	}

	loader := Preload[*testPreloadChild, testPreloadChildSlice](
		rel, []string{"id", "name"}, mapper, PreloadAs[testPreloadQuery]("c"),
	)
	_, mapperMod, _ := loader("")

	return scan.Mod(scan.StructMapper[*testPreloadParent](), mapperMod)
}

func runTestPreload(t *testing.T, mapper PreloadMapper[*testPreloadChild], cols []string, rows [][]any) []*testPreloadParent {
	t.Helper()

	full := buildTestPreloadScanner(mapper)
	res, err := scan.AllFromRows(context.Background(), full, &testRows{cols: cols, rows: rows})
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}

	return res
}

// TestPreloadMapperParity checks that a typed PreloadMapper produces exactly
// the same results as the default reflection-based mapper, including the LEFT
// JOIN NULL semantics (all-NULL row -> no child, partial NULL -> zero values).
func TestPreloadMapperParity(t *testing.T) {
	cases := map[string]struct {
		cols []string
		rows [][]any
		want []*testPreloadParent
	}{
		"matched child": {
			cols: []string{"id", "c.id", "c.name"},
			rows: [][]any{{int64(1), int64(10), "x"}},
			want: []*testPreloadParent{{ID: 1, Child: &testPreloadChild{
				ID: 10, Name: sql.Null[string]{V: "x", Valid: true},
			}}},
		},
		"unmatched join, all child columns NULL": {
			cols: []string{"id", "c.id", "c.name"},
			rows: [][]any{{int64(2), nil, nil}},
			want: []*testPreloadParent{{ID: 2, Child: nil}},
		},
		"partial NULL": {
			cols: []string{"id", "c.id", "c.name"},
			rows: [][]any{{int64(3), int64(11), nil}},
			want: []*testPreloadParent{{ID: 3, Child: &testPreloadChild{ID: 11}}},
		},
		"column subset (PreloadOnly)": {
			cols: []string{"id", "c.id"},
			rows: [][]any{
				{int64(4), int64(12)},
				{int64(5), nil},
			},
			want: []*testPreloadParent{
				{ID: 4, Child: &testPreloadChild{ID: 12}},
				{ID: 5, Child: nil},
			},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			byReflection := runTestPreload(t, nil, tc.cols, tc.rows)
			typed := runTestPreload(t, testPreloadChildMapper, tc.cols, tc.rows)

			if !reflect.DeepEqual(byReflection, tc.want) {
				t.Errorf("reflection mapper: got %s, want %s", printTestParents(byReflection), printTestParents(tc.want))
			}
			if !reflect.DeepEqual(typed, tc.want) {
				t.Errorf("typed mapper: got %s, want %s", printTestParents(typed), printTestParents(tc.want))
			}
			if !reflect.DeepEqual(byReflection, typed) {
				t.Errorf("mappers disagree: reflection %s, typed %s", printTestParents(byReflection), printTestParents(typed))
			}
		})
	}
}

// BenchmarkAfterPreloader isolates the AfterPreloader collect/load path: N
// per-row Collect calls followed by one Load that assembles the collected
// objects for the sub-loaders. A no-op sub-loader is appended so Collect and
// Load do not short-circuit on an empty func list.
func BenchmarkAfterPreloader(b *testing.B) {
	children := make([]*testPreloadChild, 10000)
	for i := range children {
		children[i] = &testPreloadChild{
			ID:   int64(i),
			Name: sql.Null[string]{V: fmt.Sprintf("name-%d", i), Valid: true},
		}
	}
	noop := bob.LoaderFunc(func(context.Context, bob.Executor, any) error { return nil })

	for _, n := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.ReportAllocs()
			for range b.N {
				ap := NewAfterPreloader[*testPreloadChild, testPreloadChildSlice]()
				ap.AppendLoader(noop)
				for _, c := range children[:n] {
					if err := ap.Collect(c); err != nil {
						b.Fatal(err)
					}
				}
				if err := ap.Load(context.Background(), nil, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkPreloadMapper(b *testing.B) {
	cols := []string{"id", "c.id", "c.name"}

	for _, n := range []int{100, 1000, 10000} {
		rows := make([][]any, n)
		for i := range rows {
			rows[i] = []any{int64(i), int64(i * 10), fmt.Sprintf("name-%d", i)}
		}

		b.Run(fmt.Sprintf("reflection/%d", n), benchPreload(nil, cols, rows))
		b.Run(fmt.Sprintf("typed/%d", n), benchPreload(testPreloadChildMapper, cols, rows))
	}
}

func benchPreload(mapper PreloadMapper[*testPreloadChild], cols []string, rows [][]any) func(*testing.B) {
	full := buildTestPreloadScanner(mapper)

	return func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			if _, err := scan.AllFromRows(context.Background(), full, &testRows{cols: cols, rows: rows}); err != nil {
				b.Fatal(err)
			}
		}
	}
}

func printTestParents(ps []*testPreloadParent) string {
	b := new(strings.Builder)
	b.WriteString("[")
	for i, p := range ps {
		if i > 0 {
			b.WriteString(" ")
		}
		if p.Child == nil {
			fmt.Fprintf(b, "{ID:%d Child:nil}", p.ID)
			continue
		}
		fmt.Fprintf(b, "{ID:%d Child:%+v}", p.ID, *p.Child)
	}
	b.WriteString("]")
	return b.String()
}
