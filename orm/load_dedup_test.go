package orm

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

// both child-mapper shapes for every dedup test; nil = the reflection fallback
var testDedupMappers = []struct {
	name   string
	mapper PreloadMapper[*testPreloadChild]
}{
	{"reflection", nil},
	{"typed", testPreloadChildMapper},
}

// a no-op loader; attaching one makes the AfterPreloader actually collect
func testCollectLoader() Loader[testPreloadQuery] {
	return func(context.Context, bob.Executor, any) error { return nil }
}

func testCollectedCount(t *testing.T, l bob.Loader) int {
	t.Helper()

	after, ok := l.(*AfterPreloader)
	if !ok {
		t.Fatalf("loader is %T, not *AfterPreloader", l)
	}

	return after.numCollected()
}

func scanTestPreloadRows(t *testing.T, scanner scan.Mapper[*testPreloadParent], cols []string, rows [][]any) []*testPreloadParent {
	t.Helper()

	res, err := scan.AllFromRows(context.Background(), scanner, &testRows{cols: cols, rows: rows})
	if err != nil {
		t.Fatalf("scanning: %v", err)
	}

	return res
}

// mirrors the generated typed mapper shape, like testPreloadChildMapper
func testPreloadGrandChildMapper(prefix string) scan.Mapper[*testPreloadGrandChild] {
	return func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any) (*testPreloadGrandChild, error)) {
		type gchildBuf struct {
			id  sql.Null[int64]
			tag sql.Null[string]
		}
		type target struct {
			idx int
			dst func(*gchildBuf) any
		}

		targets := make([]target, 0, 2)
		for i, col := range cols {
			name, ok := strings.CutPrefix(col, prefix)
			if !ok {
				continue
			}
			switch name {
			case "id":
				targets = append(targets, target{i, func(b *gchildBuf) any { return &b.id }})
			case "tag":
				targets = append(targets, target{i, func(b *gchildBuf) any { return &b.tag }})
			}
		}

		return func(row *scan.Row) (any, error) {
				buf := new(gchildBuf)
				for _, t := range targets {
					row.ScheduleScanByIndex(t.idx, t.dst(buf))
				}
				return buf, nil
			}, func(link any) (*testPreloadGrandChild, error) {
				buf := link.(*gchildBuf)
				if !buf.id.Valid && !buf.tag.Valid {
					return nil, nil
				}

				o := new(testPreloadGrandChild)
				if buf.id.Valid {
					o.ID = buf.id.V
				}
				o.Tag = buf.tag
				return o, nil
			}
	}
}

func testNestedGChildPreloader(mapper PreloadMapper[*testPreloadGrandChild], opts ...PreloadOption[testPreloadQuery]) Preloader[testPreloadQuery] {
	rel := PreloadRel[bob.Expression]{
		Name: "GChild",
		Sides: []PreloadSide[bob.Expression]{{
			From:        testNameable{name: "children", alias: "children"},
			To:          testNameable{name: "grandchildren", alias: "grandchildren"},
			FromColumns: []string{"gchild_id"},
			ToColumns:   []string{"id"},
		}},
	}

	return Preload[*testPreloadGrandChild, testPreloadGrandChildSlice](
		rel, []string{"id", "tag"}, mapper,
		append([]PreloadOption[testPreloadQuery]{PreloadAs[testPreloadQuery]("g")}, opts...)...,
	)
}

// compares by value, not pointer identity: dedup must never change what is loaded
func assertSameChildValues(t *testing.T, got, want []*testPreloadParent) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("got %d parents, want %d", len(got), len(want))
	}

	for i := range got {
		if got[i].ID != want[i].ID {
			t.Errorf("parent %d: got ID %d, want %d", i, got[i].ID, want[i].ID)
		}

		g, w := got[i].Child, want[i].Child
		if (g == nil) != (w == nil) {
			t.Errorf("parent %d: got child %v, want %v", i, g, w)
			continue
		}
		if g == nil {
			continue
		}
		if g.ID != w.ID || g.Name != w.Name {
			t.Errorf("parent %d: got child %+v, want %+v", i, *g, *w)
		}

		gg, wg := g.GChild, w.GChild
		if (gg == nil) != (wg == nil) {
			t.Errorf("parent %d: got grandchild %v, want %v", i, gg, wg)
			continue
		}
		if gg != nil && (gg.ID != wg.ID || gg.Tag != wg.Tag) {
			t.Errorf("parent %d: got grandchild %+v, want %+v", i, *gg, *wg)
		}
	}
}

func TestPreloadDedupSharesInstance(t *testing.T) {
	cols := []string{"id", "c.id", "c.name"}
	rows := [][]any{
		{int64(1), int64(10), "x"},
		{int64(2), int64(10), "x"},
		{int64(3), int64(10), "x"},
	}

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			scanner, loaders := buildTestPreloadScannerLoaders(m.mapper, testCollectLoader())
			res := scanTestPreloadRows(t, scanner, cols, rows)

			if len(res) != 3 {
				t.Fatalf("got %d parents, want 3", len(res))
			}
			if res[0].Child == nil || res[0].Child.ID != 10 || res[0].Child.Name.V != "x" {
				t.Fatalf("unexpected child: %+v", res[0].Child)
			}
			for i, p := range res {
				if p.Child != res[0].Child {
					t.Errorf("parent %d does not share the child instance", i)
				}
			}
			if got := testCollectedCount(t, loaders[0]); got != 1 {
				t.Errorf("collected %d children, want 1", got)
			}
		})
	}
}

// non-consecutive duplicates exercise the map path, not just the last-key fast path
func TestPreloadDedupNonConsecutiveKeys(t *testing.T) {
	cols := []string{"id", "c.id", "c.name"}
	rows := [][]any{
		{int64(1), int64(10), "x"},
		{int64(2), int64(10), "x"},
		{int64(3), int64(20), "y"},
		{int64(4), int64(10), "x"},
		{int64(5), int64(20), "y"},
	}

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			scanner, loaders := buildTestPreloadScannerLoaders(m.mapper, testCollectLoader())
			res := scanTestPreloadRows(t, scanner, cols, rows)

			if res[0].Child == nil || res[0].Child.ID != 10 || res[2].Child == nil || res[2].Child.ID != 20 {
				t.Fatalf("unexpected children: %+v, %+v", res[0].Child, res[2].Child)
			}
			if res[1].Child != res[0].Child || res[3].Child != res[0].Child {
				t.Error("parents with key 10 do not share the child instance")
			}
			if res[4].Child != res[2].Child {
				t.Error("parents with key 20 do not share the child instance")
			}
			if res[0].Child == res[2].Child {
				t.Error("different keys share the same child instance")
			}
			if got := testCollectedCount(t, loaders[0]); got != 2 {
				t.Errorf("collected %d children, want 2", got)
			}
		})
	}
}

func TestPreloadDedupKeyColumnNotSelected(t *testing.T) {
	cols := []string{"id", "c.name"} // "c.id" (the join key) is not selected
	rows := [][]any{
		{int64(1), "x"},
		{int64(2), "x"},
	}

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			scanner, loaders := buildTestPreloadScannerLoaders(
				m.mapper,
				PreloadOnly[testPreloadQuery]{"name"},
				testCollectLoader(),
			)
			res := scanTestPreloadRows(t, scanner, cols, rows)

			if len(res) != 2 {
				t.Fatalf("got %d parents, want 2", len(res))
			}
			for i, p := range res {
				if p.Child == nil || p.Child.Name.V != "x" {
					t.Errorf("parent %d has unexpected child: %+v", i, p.Child)
				}
			}
			if res[0].Child == res[1].Child {
				t.Error("children are shared even though the key column is not selected")
			}
			if got := testCollectedCount(t, loaders[0]); got != 2 {
				t.Errorf("collected %d children, want 2", got)
			}
		})
	}
}

// unmatched LEFT JOIN rows must keep a nil child and must not pollute the cache
func TestPreloadDedupNullRows(t *testing.T) {
	cols := []string{"id", "c.id", "c.name"}
	rows := [][]any{
		{int64(1), int64(10), "x"},
		{int64(2), nil, nil},
		{int64(3), int64(10), "x"},
	}

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			scanner, loaders := buildTestPreloadScannerLoaders(m.mapper, testCollectLoader())
			res := scanTestPreloadRows(t, scanner, cols, rows)

			if res[1].Child != nil {
				t.Errorf("unmatched row got child %+v", res[1].Child)
			}
			if res[0].Child == nil || res[0].Child.ID != 10 {
				t.Fatalf("unexpected child: %+v", res[0].Child)
			}
			if res[2].Child != res[0].Child {
				t.Error("duplicate rows around a NULL row do not share the child instance")
			}
			// the NULL row still collects its zero value, so 2 = one real child + one nil
			if got := testCollectedCount(t, loaders[0]); got != 2 {
				t.Errorf("collected %d children, want 2", got)
			}
		})
	}
}

// The safety valve: preloadDedupColdLimit distinct keys without a single hit
// turn dedup off; the remaining rows get fresh per-row instances.
func TestPreloadDedupAutoOff(t *testing.T) {
	cols := []string{"id", "c.id", "c.name"}

	// keys 1..limit are all distinct, then key 1 repeats twice
	rows := make([][]any, 0, preloadDedupColdLimit+2)
	for i := range preloadDedupColdLimit {
		rows = append(rows, []any{int64(i), int64(i + 1), "x"})
	}
	rows = append(
		rows,
		[]any{int64(9001), int64(1), "x"},
		[]any{int64(9002), int64(1), "x"},
	)

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			scanner, _ := buildTestPreloadScannerLoaders(m.mapper)
			res := scanTestPreloadRows(t, scanner, cols, rows)

			last, prev := res[len(res)-1], res[len(res)-2]
			if last.Child == nil || last.Child.ID != 1 || prev.Child == nil || prev.Child.ID != 1 {
				t.Fatalf("unexpected children: %+v, %+v", prev.Child, last.Child)
			}
			// dedup switched off before these rows: no sharing anymore
			if last.Child == res[0].Child || last.Child == prev.Child {
				t.Error("children are still shared after the auto-off limit")
			}
		})
	}
}

// A single early hit disables the safety valve: dedup stays on well past the
// limit and still shares instances.
func TestPreloadDedupAutoOffNotTriggeredAfterHit(t *testing.T) {
	cols := []string{"id", "c.id", "c.name"}

	// key 1 repeats immediately (a hit), then limit+1 distinct keys, then key 2 again
	rows := make([][]any, 0, preloadDedupColdLimit+4)
	rows = append(
		rows,
		[]any{int64(0), int64(1), "x"},
		[]any{int64(1), int64(1), "x"},
	)
	for i := range preloadDedupColdLimit + 1 {
		rows = append(rows, []any{int64(i + 2), int64(i + 2), "x"})
	}
	rows = append(rows, []any{int64(9001), int64(2), "x"})

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			scanner, _ := buildTestPreloadScannerLoaders(m.mapper)
			res := scanTestPreloadRows(t, scanner, cols, rows)

			if res[1].Child != res[0].Child {
				t.Error("early duplicate rows do not share the child instance")
			}
			last := res[len(res)-1]
			if last.Child == nil || last.Child.ID != 2 {
				t.Fatalf("unexpected child: %+v", last.Child)
			}
			if last.Child != res[2].Child {
				t.Error("late duplicate is not shared even though dedup saw an early hit")
			}
		})
	}
}

func TestPreloadDedupCompositeKeyEncoding(t *testing.T) {
	encode := func(parts ...string) []byte {
		var key []byte
		for _, p := range parts {
			key = appendPreloadKeyValue(key, reflect.ValueOf(p), true)
		}
		return key
	}

	k1 := encode("a\x00b", "c")
	k2 := encode("a", "b\x00c")
	if bytes.Equal(k1, k2) {
		t.Errorf("composite keys collide: %q", k1)
	}
}

func TestPreloadDedupCompositeKey(t *testing.T) {
	rel := PreloadRel[bob.Expression]{
		Name: "Child",
		Sides: []PreloadSide[bob.Expression]{{
			From:        testNameable{name: "parents", alias: "parents"},
			To:          testNameable{name: "children", alias: "children"},
			FromColumns: []string{"child_id", "child_name"},
			ToColumns:   []string{"id", "name"},
		}},
	}

	cols := []string{"id", "c.id", "c.name"}
	rows := [][]any{
		{int64(1), int64(10), "x"},
		{int64(2), int64(10), "x"},
		{int64(3), int64(10), "y"},
	}

	for _, m := range testDedupMappers {
		t.Run(m.name, func(t *testing.T) {
			loader := Preload[*testPreloadChild, testPreloadChildSlice](
				rel, []string{"id", "name"}, m.mapper,
				PreloadAs[testPreloadQuery]("c"),
			)
			_, mapperMod, _ := loader("")
			scanner := scan.Mod(scan.StructMapper[*testPreloadParent](), mapperMod)

			res := scanTestPreloadRows(t, scanner, cols, rows)

			if res[1].Child != res[0].Child {
				t.Error("parents with the same composite key do not share the child instance")
			}
			if res[2].Child == res[0].Child {
				t.Error("parents with different composite keys share the same child instance")
			}
			if res[2].Child == nil || res[2].Child.Name.V != "y" {
				t.Errorf("unexpected child: %+v", res[2].Child)
			}
		})
	}
}

// Nested preloads: the reflection path skips them on duplicate rows; the typed
// path builds the child per row but the nested preloader dedups its own
// collects. The tree must match the baseline.
func TestPreloadDedupNested(t *testing.T) {
	cols := []string{"id", "c.id", "c.name", "g.id", "g.tag"}
	rows := [][]any{
		{int64(1), int64(10), "x", int64(100), "t"},
		{int64(2), int64(10), "x", int64(100), "t"},
		{int64(3), int64(10), "x", int64(100), "t"},
	}

	variants := []struct {
		name              string
		child             PreloadMapper[*testPreloadChild]
		gchild            PreloadMapper[*testPreloadGrandChild]
		wantNestedCollect int
	}{
		{"reflection", nil, nil, 1},
		{"typed", testPreloadChildMapper, testPreloadGrandChildMapper, 1},
	}

	for _, v := range variants {
		t.Run(v.name, func(t *testing.T) {
			scanner, loaders := buildTestPreloadScannerLoaders(
				v.child,
				testCollectLoader(),
				testNestedGChildPreloader(v.gchild, testCollectLoader()),
			)
			res := scanTestPreloadRows(t, scanner, cols, rows)

			baseline := scanTestPreloadRows(t, buildTestPreloadScanner(
				v.child, testNestedGChildPreloader(v.gchild),
			), cols, rows)
			assertSameChildValues(t, res, baseline)

			for i, p := range res {
				if p.Child != res[0].Child {
					t.Errorf("parent %d does not share the child instance", i)
				}
			}
			gchild := res[0].Child.GChild
			if gchild == nil || gchild.ID != 100 || gchild.Tag.V != "t" {
				t.Fatalf("unexpected grandchild: %+v", gchild)
			}

			if got := testCollectedCount(t, loaders[0]); got != 1 {
				t.Errorf("collected %d children, want 1", got)
			}
			if got := testCollectedCount(t, loaders[1]); got != v.wantNestedCollect {
				t.Errorf("collected %d grandchildren, want %d", got, v.wantNestedCollect)
			}
		})
	}
}

func BenchmarkPreloadDedup(b *testing.B) {
	const total = 10_000
	cols := []string{"id", "c.id", "c.name"}

	shapes := []struct {
		name      string
		distinct  int
		scattered bool
	}{
		// many parents share few children, sorted by key (fast path)
		{"M50_consecutive", 50, false},
		// same duplication but interleaved keys (map path)
		{"M50_scattered", 50, true},
		// no duplication at all: the upper bound of the key-matching cost
		{"M10000_distinct", 10_000, false},
	}

	for _, m := range testDedupMappers {
		for _, shape := range shapes {
			rows := make([][]any, total)
			for i := range rows {
				key := i / (total / shape.distinct)
				if shape.scattered {
					key = i % shape.distinct
				}
				rows[i] = []any{int64(i), int64(key), fmt.Sprintf("name-%d", key)}
			}

			b.Run(fmt.Sprintf("%s/%s", m.name, shape.name),
				benchPreload(m.mapper, cols, rows))
		}
	}
}
