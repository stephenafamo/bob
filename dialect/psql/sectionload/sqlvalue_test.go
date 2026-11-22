package sectionload

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

// PostgreSQL's jsonb operators (?, ?|, ?&) are legal in a
// relationship's fixed "where" (EdgeWhere.SQLValue), e.g.
// `has_active = (data ? 'active')`. bob's own RawQuery/psql.Raw treat "?"
// in a string as an argument placeholder to be rewritten "$N" -- fine for
// sectionload's own composed SQL text, wrong for a value that is already a
// complete SQL expression. matchWhere/parentKeySubquery (compose.go) embed EdgeWhere.SQLValue via expr.Raw, which writes it verbatim
// with no placeholder scanning at all, so these operators must survive
// composeIsland's full pipeline unchanged and must never appear in the
// query's argument count.

type (
	sqlValueParent struct{ ID int }
	sqlValueChild  struct {
		ID        int
		ParentID  int
		Data      string
		HasActive bool
	}
)

type sqlValueParentBuf struct{ id int }

func (b *sqlValueParentBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *sqlValueParentBuf) BuildFor(modelCols []Column) *sqlValueParent {
	return &sqlValueParent{ID: b.id}
}

type sqlValueFromParent struct {
	ID        int
	Data      string
	HasActive bool
}

type sqlValueFromParentBuf struct {
	id        int
	data      string
	hasActive bool
}

func (b *sqlValueFromParentBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.data, &b.hasActive}
}

func (b *sqlValueFromParentBuf) BuildFor(modelCols []Column) *sqlValueFromParent {
	return &sqlValueFromParent{
		ID:        b.id,
		Data:      b.data,
		HasActive: b.hasActive,
	}
}

// sqlValueChildBuf is the scan buffer for sqlValueChildParts' 4-column TableParts
// (id, parent_id, data, has_active). Targets() must return exactly one
// destination per Columns entry, in the same order (parts.go's generator
// contract) — see sqlValueChildParts' doc comment for why this matters.
type sqlValueChildBuf struct {
	id        int
	parentID  int
	data      string
	hasActive bool
}

func (b *sqlValueChildBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.parentID, &b.data, &b.hasActive}
}

func (b *sqlValueChildBuf) BuildFor(modelCols []Column) *sqlValueChild {
	return &sqlValueChild{ID: b.id, ParentID: b.parentID, Data: b.data, HasActive: b.hasActive}
}

// sqlValueChildParts returns the single shared TableParts[*sqlValueChild] used by
// both buildSQLValueQuery (the "to"-side/matchWhere path) and
// buildSQLValueFromSideQuery (the "from"-side/parentKeySubquery path), so their
// Columns and sqlValueChildBuf's Targets() can never drift apart between the two
// call sites again. A mismatch (say, 4 Columns (id, parent_id, data,
// has_active) against a 2-entry Targets()) would be harmless today
// because these are no-DB tests (NewBuf/Targets/Build are never invoked
// here), but it would fail the moment a DB-executing test scanned rows
// through this fixture (rows.Scan: "expected 2 destination arguments in
// Scan, not 4").
func sqlValueChildParts() TableParts[*sqlValueChild] {
	return TableParts[*sqlValueChild]{
		Table: "sv_children",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			{Name: "parent_id", DBType: "bigint"},
			{Name: "data", DBType: "jsonb"},
			{Name: "has_active", DBType: "boolean"},
		},
		NewBuf: func() Buf[*sqlValueChild] { return &sqlValueChildBuf{} },
	}
}

// buildSQLValueQuery composes a one-edge island whose edge carries a "to"-side
// EdgeWhere containing the given jsonb operator, and returns the rendered
// SQL and args exactly as sectionload.All would produce them (minus
// execution).
func buildSQLValueQuery(t *testing.T, sqlValue string) (string, []any) {
	t.Helper()
	edge := Edge[*sqlValueParent, *sqlValueChild]{
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Where: []EdgeWhere{
			{On: "to", Column: "has_active", SQLValue: sqlValue},
		},
		Child: sqlValueChildParts(),
	}
	root := partsAdapter[*sqlValueParent]{TableParts[*sqlValueParent]{
		Table:   "sv_parents",
		Columns: []Column{{Name: "id", DBType: "bigint"}},
		NewBuf:  func() Buf[*sqlValueParent] { return &sqlValueParentBuf{} },
	}}

	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM sv_parents WHERE id = ?", 7)
	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sqlStr, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	return sqlStr, args
}

func TestSQLValueQuestionMarkOperatorSurvivesUnchanged(t *testing.T) {
	// A parenthesized SQLValue is a right-hand-side SQL expression, per
	// A-09-2/A-09-3: EdgeWhere renders "<column> = <SQLValue>".
	for _, tt := range []struct {
		name     string
		sqlValue string
	}{
		{"exists", "(data ? 'active')"},
		{"any-of", "(data ?| array['a','b'])"},
		{"all-of", "(data ?& array['a','b'])"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, args := buildSQLValueQuery(t, tt.sqlValue)
			if !strings.Contains(sqlStr, tt.sqlValue) {
				t.Fatalf("composed SQL does not contain the literal SQLValue %q verbatim:\n%s", tt.sqlValue, sqlStr)
			}
			// The only real placeholder is the parent query's "id = ?"
			// (rewritten to "id = $1" by RawQuery); SQLValue's "?" must
			// not add any more.
			if len(args) != 1 || args[0] != 7 {
				t.Fatalf("args = %v, want exactly [7] (SQLValue's jsonb operator must not be miscounted as a placeholder)", args)
			}
		})
	}
}

// buildSQLValueFromSideQuery composes a complete one-edge island carrying a
// from-side EdgeWhere. The parent query exposes every column referenced by
// that condition, and composeIsland defines _sl_src, so the rendered SQL is
// independently executable against sv_parents and sv_children.
func buildSQLValueFromSideQuery(t *testing.T, sqlValue string) (string, []any) {
	t.Helper()

	edge := Edge[*sqlValueFromParent, *sqlValueChild]{
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Where: []EdgeWhere{
			{On: "from", Column: "has_active", SQLValue: sqlValue},
		},
		Child: sqlValueChildParts(),
	}

	root := partsAdapter[*sqlValueFromParent]{TableParts[*sqlValueFromParent]{
		Table: "sv_parents",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			{Name: "data", DBType: "jsonb"},
			{Name: "has_active", DBType: "boolean"},
		},
		NewBuf: func() Buf[*sqlValueFromParent] {
			return &sqlValueFromParentBuf{}
		},
	}}

	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery(
		"SELECT id, data, has_active FROM sv_parents WHERE id = ?",
		7,
	)
	q, err := composeIsland(
		parent,
		root,
		plan,
		"",
	)
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}

	sqlStr, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	return sqlStr, args
}

func TestSQLValueQuestionMarkOperatorSurvivesUnchanged_FromSide(t *testing.T) {
	tests := []struct {
		name     string
		sqlValue string
	}{
		{"exists", "(data ? 'active')"},
		{"any-of", "(data ?| array['a','b'])"},
		{"all-of", "(data ?& array['a','b'])"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sqlStr, args := buildSQLValueFromSideQuery(t, tt.sqlValue)

			if !strings.Contains(sqlStr, tt.sqlValue) {
				t.Fatalf(
					"composed SQL does not contain SQLValue %q verbatim:\n%s",
					tt.sqlValue,
					sqlStr,
				)
			}
			if !strings.Contains(sqlStr, `"_sl_src" AS MATERIALIZED`) {
				t.Fatalf("composed SQL does not define the parent CTE:\n%s", sqlStr)
			}
			if !strings.Contains(
				sqlStr,
				`SELECT id, data, has_active FROM sv_parents WHERE id = $1`,
			) {
				t.Fatalf(
					"parent CTE does not expose from-side condition columns:\n%s",
					sqlStr,
				)
			}
			if len(args) != 1 || args[0] != 7 {
				t.Fatalf(
					"args = %v, want exactly [7] "+
						"(SQLValue's jsonb operator must not add a placeholder)",
					args,
				)
			}
		})
	}
}

func TestSQLValueOrdinaryLiteralUnaffected(t *testing.T) {
	// Regression guard: ordinary fixed values (no "?") must render exactly
	// as before -- this fix must not change any existing relationship's
	// composed SQL.
	sqlStr, args := buildSQLValueQuery(t, "'active'")
	if !strings.Contains(sqlStr, "= 'active'") {
		t.Fatalf("composed SQL does not contain \"= 'active'\":\n%s", sqlStr)
	}
	if len(args) != 1 || args[0] != 7 {
		t.Fatalf("args = %v, want exactly [7]", args)
	}
}

// TestSQLValueFixtureColumnsMatchTargets pins that sqlValueChildParts'
// Columns must stay in 1:1 order/count correspondence with sqlValueChildBuf's
// Targets() (parts.go's generator contract). buildSQLValueQuery and
// buildSQLValueFromSideQuery both build no-DB queries today, so a drift here
// would build clean and pass silently — see sqlValueChildParts' doc comment for
// why that is not actually harmless.
func TestSQLValueFixtureColumnsMatchTargets(t *testing.T) {
	parts := sqlValueChildParts()

	wantColumns := []struct {
		name   string
		dbType string
	}{
		{"id", "bigint"},
		{"parent_id", "bigint"},
		{"data", "jsonb"},
		{"has_active", "boolean"},
	}
	if len(parts.Columns) != len(wantColumns) {
		t.Fatalf("len(Columns)=%d, want %d", len(parts.Columns), len(wantColumns))
	}
	for i, want := range wantColumns {
		got := parts.Columns[i]
		if got.Name != want.name || got.DBType != want.dbType {
			t.Fatalf(
				"Columns[%d]=%#v, want Name=%q DBType=%q",
				i,
				got,
				want.name,
				want.dbType,
			)
		}
	}

	buf := parts.NewBuf()
	targets := buf.TargetsFor(nil)
	if len(targets) != len(wantColumns) {
		t.Fatalf(
			"len(Targets())=%d, len(Columns)=%d",
			len(targets),
			len(wantColumns),
		)
	}

	id, ok := targets[0].(*int)
	if !ok {
		t.Fatalf("Targets()[0] has type %T, want *int for id", targets[0])
	}
	parentID, ok := targets[1].(*int)
	if !ok {
		t.Fatalf("Targets()[1] has type %T, want *int for parent_id", targets[1])
	}
	data, ok := targets[2].(*string)
	if !ok {
		t.Fatalf("Targets()[2] has type %T, want *string for data", targets[2])
	}
	hasActive, ok := targets[3].(*bool)
	if !ok {
		t.Fatalf("Targets()[3] has type %T, want *bool for has_active", targets[3])
	}

	*id = 11
	*parentID = 22
	*data = `{"active":true}`
	*hasActive = true

	got := buf.BuildFor(nil)
	if got.ID != 11 ||
		got.ParentID != 22 ||
		got.Data != `{"active":true}` ||
		!got.HasActive {
		t.Fatalf("Build()=%#v; Targets order does not match Columns order", got)
	}
}
