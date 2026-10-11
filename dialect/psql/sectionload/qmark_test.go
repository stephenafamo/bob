package sectionload

// section0List (compose.go) joins column names and NULL-cast DB type names
// into one string. Embedding it via psql.Raw would be wrong: Raw scans the
// result for "?" and treats any occurrence as a bob argument placeholder,
// the same failure class as in matchWhere/parentKeySubquery. PostgreSQL
// allows "?" inside a quoted identifier (e.g. a column legally named
// "flag?"), so a Column whose Name or DBType contains "?" would hit an
// args/placeholder-count mismatch at bob.Build time. These tests pin that
// such names survive unchanged.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

type (
	qmarkParent    struct{ ID int }
	qmarkParentBuf struct {
		id   int
		flag int
	}
)

func (b *qmarkParentBuf) TargetsFor(cols []Column) []any           { return []any{&b.id, &b.flag} }
func (b *qmarkParentBuf) BuildFor(modelCols []Column) *qmarkParent { return &qmarkParent{ID: b.id} }

// qmarkChildBuf is a 2-column scan buffer (id, weird) dedicated to
// TestSection0List_DBTypeWithQuestionMarkSurvivesUnchanged's own 2-entry
// Child.Columns below. It intentionally does not reuse sqlvalue_test.go's
// sqlValueChildBuf: that type has 4 fields (see sqlValueChildParts), and sharing it
// here would silently reintroduce a Columns/Targets count mismatch.
type qmarkChildBuf struct {
	id    int
	weird int
}

func (b *qmarkChildBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.weird} }

func (b *qmarkChildBuf) BuildFor(modelCols []Column) *sqlValueChild { return &sqlValueChild{ID: b.id} }

func TestSection0List_ColumnNameWithQuestionMarkSurvivesUnchanged(t *testing.T) {
	root := partsAdapter[*qmarkParent]{TableParts[*qmarkParent]{
		Table: "qm_parents",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			// A legally quoted PostgreSQL identifier containing "?".
			{Name: "flag?", DBType: "boolean"},
		},
		NewBuf: func() Buf[*qmarkParent] { return &qmarkParentBuf{} },
	}}

	// SELECT * (not an explicit column list): the parent query is the
	// caller's opaque SQL, wrapped as the "_sl_src" CTE — it never needs to
	// spell out "flag?" itself in text form. Only section0List (which
	// builds "_sl_src"."flag?" by name from root.cols()) does.
	parent := psql.RawQuery("SELECT * FROM qm_parents WHERE id = ?", 7)
	q, err := composeIsland(parent, root, islandPlan{}, "")
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
	root := partsAdapter[*qmarkParent]{TableParts[*qmarkParent]{
		Table: "qm_parents",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			// An unusual but legal DBType string containing "?" (e.g. a
			// domain type named with one). This is what NULL::<DBType> casts
			// embed for section0List's non-root columns; simulate it here
			// directly on the root, which is embedded the same way.
		},
		NewBuf: func() Buf[*qmarkParent] { return &qmarkParentBuf{} },
	}}
	// A child section forces section0List's NULL-padding branch, which is
	// where DBType (not just Name) gets interpolated.
	edge := Edge[*qmarkParent, *sqlValueChild]{
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Child: TableParts[*sqlValueChild]{
			Table: "qm_children",
			Columns: []Column{
				{Name: "id", DBType: "bigint"},
				// A DB type name containing "?" (contrived but legal as a
				// quoted PostgreSQL type/domain identifier).
				{Name: "weird", DBType: `"odd?type"`},
			},
			NewBuf: func() Buf[*sqlValueChild] { return &qmarkChildBuf{} },
		},
	}
	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	parent := psql.RawQuery("SELECT id FROM qm_parents WHERE id = ?", 7)
	q, err := composeIsland(parent, root, plan, "")
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
