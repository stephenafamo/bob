package sectionload

// P-29: section0List (compose.go) joins column names and NULL-cast DB type
// names into one string and used to embed it via psql.Raw, which scans the
// result for "?" and treats any occurrence as a bob argument placeholder —
// the same failure class P-09 fixed in matchWhere/parentKeySubquery.
// PostgreSQL allows "?" inside a quoted identifier (e.g. a column legally
// named "flag?"), so a Column whose Name or DBType contains "?" would hit
// the same args/placeholder-count mismatch at bob.Build time. Discovered by
// issue P-09's own Codex review (2026-08-09) and registered as issue P-29;
// fixed together with P-09 since it is the same failure class in the same
// file.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

type (
	p29Parent    struct{ ID int }
	p29ParentBuf struct {
		id   int
		flag int
	}
)

func (b *p29ParentBuf) TargetsFor(cols []Column) []any         { return []any{&b.id, &b.flag} }
func (b *p29ParentBuf) BuildFor(modelCols []Column) *p29Parent { return &p29Parent{ID: b.id} }

// p29ChildBuf is a 2-column scan buffer (id, weird) dedicated to
// TestSection0List_DBTypeWithQuestionMarkSurvivesUnchanged's own 2-entry
// Child.Columns below. It intentionally does not reuse sqlvalue_test.go's
// p09ChildBuf: that type grew to 4 fields for P-09's round-3 fixture
// (p09ChildParts), and sharing it here would silently reintroduce the exact
// Columns/Targets count mismatch that fix was meant to eliminate (P-09
// round 3 Important 2, Codex review 2026-08-09 — found while wiring that
// fix, not by Codex directly, since this file was outside the reviewed
// diff).
type p29ChildBuf struct {
	id    int
	weird int
}

func (b *p29ChildBuf) TargetsFor(cols []Column) []any        { return []any{&b.id, &b.weird} }
func (b *p29ChildBuf) BuildFor(modelCols []Column) *p09Child { return &p09Child{ID: b.id} }

func TestSection0List_ColumnNameWithQuestionMarkSurvivesUnchanged(t *testing.T) {
	root := partsAdapter[*p29Parent]{TableParts[*p29Parent]{
		Table: "p29_parents",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			// A legally quoted PostgreSQL identifier containing "?".
			{Name: "flag?", DBType: "boolean"},
		},
		NewBuf: func() Buf[*p29Parent] { return &p29ParentBuf{} },
	}}

	// SELECT * (not an explicit column list): the parent query is the
	// caller's opaque SQL, wrapped as the "_sl_src" CTE — it never needs to
	// spell out "flag?" itself in text form. Only section0List (which
	// builds "_sl_src"."flag?" by name from root.cols()) does.
	parent := psql.RawQuery("SELECT * FROM p29_parents WHERE id = ?", 7)
	q, err := composeIsland(parent, root, islandPlan{}, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sqlStr, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v (a Column.Name containing \"?\" must not be miscounted as a placeholder)", err)
	}
	if !strings.Contains(sqlStr, `"flag?"`) {
		t.Fatalf("composed SQL does not contain the quoted identifier \"flag?\" verbatim:\n%s", sqlStr)
	}
	// The only real placeholder is the parent query's "id = ?".
	if len(args) != 1 || args[0] != 7 {
		t.Fatalf("args = %v, want exactly [7] (the column name's \"?\" must not add a second placeholder)", args)
	}
}

func TestSection0List_DBTypeWithQuestionMarkSurvivesUnchanged(t *testing.T) {
	root := partsAdapter[*p29Parent]{TableParts[*p29Parent]{
		Table: "p29_parents",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			// An unusual but legal DBType string containing "?" (e.g. a
			// domain type named with one). This is what NULL::<DBType> casts
			// embed for section0List's non-root columns; simulate it here
			// directly on the root, which is embedded the same way.
		},
		NewBuf: func() Buf[*p29Parent] { return &p29ParentBuf{} },
	}}
	// A child section forces section0List's NULL-padding branch, which is
	// where DBType (not just Name) gets interpolated.
	edge := Edge[*p29Parent, *p09Child]{
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*p09Child]{
			Table: "p29_children",
			Columns: []Column{
				{Name: "id", DBType: "bigint"},
				// A DB type name containing "?" (contrived but legal as a
				// quoted PostgreSQL type/domain identifier).
				{Name: "weird", DBType: `"odd?type"`},
			},
			NewBuf: func() Buf[*p09Child] { return &p29ChildBuf{} },
		},
	}
	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	parent := psql.RawQuery("SELECT id FROM p29_parents WHERE id = ?", 7)
	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sqlStr, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v (a Column.DBType containing \"?\" must not be miscounted as a placeholder)", err)
	}
	if !strings.Contains(sqlStr, `NULL::"odd?type"`) {
		t.Fatalf("composed SQL does not contain the literal NULL cast verbatim:\n%s", sqlStr)
	}
	if len(args) != 1 || args[0] != 7 {
		t.Fatalf("args = %v, want exactly [7] (the DB type name's \"?\" must not add a second placeholder)", args)
	}
}
