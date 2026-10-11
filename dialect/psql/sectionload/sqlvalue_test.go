package sectionload

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

// P-09: PostgreSQL's jsonb operators (?, ?|, ?&) are legal in a
// relationship's fixed "where" (EdgeWhere.SQLValue), e.g.
// `has_active = (data ? 'active')`. bob's own RawQuery/psql.Raw treat "?"
// in a string as an argument placeholder to be rewritten "$N" -- fine for
// sectionload's own composed SQL text, wrong for a value that is already a
// complete SQL expression. matchWhere/parentKeySubquery (compose.go, built
// by P-11) embed EdgeWhere.SQLValue via expr.Raw, which writes it verbatim
// with no placeholder scanning at all, so these operators must survive
// composeIsland's full pipeline unchanged and must never appear in the
// query's argument count.

type (
	p09Parent struct{ ID int }
	p09Child  struct {
		ID        int
		ParentID  int
		Data      string
		HasActive bool
	}
)

type p09ParentBuf struct{ id int }

func (b *p09ParentBuf) TargetsFor(cols []Column) []any         { return []any{&b.id} }
func (b *p09ParentBuf) BuildFor(modelCols []Column) *p09Parent { return &p09Parent{ID: b.id} }

type p09FromParent struct {
	ID        int
	Data      string
	HasActive bool
}

type p09FromParentBuf struct {
	id        int
	data      string
	hasActive bool
}

func (b *p09FromParentBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.data, &b.hasActive}
}

func (b *p09FromParentBuf) BuildFor(modelCols []Column) *p09FromParent {
	return &p09FromParent{
		ID:        b.id,
		Data:      b.data,
		HasActive: b.hasActive,
	}
}

// p09ChildBuf is the scan buffer for p09ChildParts' 4-column TableParts
// (id, parent_id, data, has_active). Targets() must return exactly one
// destination per Columns entry, in the same order (parts.go's generator
// contract) — see p09ChildParts' doc comment for the round-3 Codex review
// finding this fixes (Important 2).
type p09ChildBuf struct {
	id        int
	parentID  int
	data      string
	hasActive bool
}

func (b *p09ChildBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.parentID, &b.data, &b.hasActive}
}

func (b *p09ChildBuf) BuildFor(modelCols []Column) *p09Child {
	return &p09Child{ID: b.id, ParentID: b.parentID, Data: b.data, HasActive: b.hasActive}
}

// p09ChildParts returns the single shared TableParts[*p09Child] used by
// both buildP09Query (the "to"-side/matchWhere path) and
// buildP09FromSideQuery (the "from"-side/parentKeySubquery path), so their
// Columns and p09ChildBuf's Targets() can never drift apart between the two
// call sites again. Before this (P-09 round 3 Important 2, Codex review
// 2026-08-09), buildP09Query declared 4 Columns (id, parent_id, data,
// has_active) against p09ChildBuf's then-2-entry Targets() — harmless today
// because these are no-DB tests (NewBuf/Targets/Build are never invoked
// here), but it would fail the moment a DB-executing test scanned rows
// through this fixture (rows.Scan: "expected 2 destination arguments in
// Scan, not 4").
func p09ChildParts() TableParts[*p09Child] {
	return TableParts[*p09Child]{
		Table: "p09_children",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			{Name: "parent_id", DBType: "bigint"},
			{Name: "data", DBType: "jsonb"},
			{Name: "has_active", DBType: "boolean"},
		},
		NewBuf: func() Buf[*p09Child] { return &p09ChildBuf{} },
	}
}

// buildP09Query composes a one-edge island whose edge carries a "to"-side
// EdgeWhere containing the given jsonb operator, and returns the rendered
// SQL and args exactly as sectionload.All would produce them (minus
// execution).
func buildP09Query(t *testing.T, sqlValue string) (string, []any) {
	t.Helper()
	edge := Edge[*p09Parent, *p09Child]{
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Where: []EdgeWhere{
			{On: "to", Column: "has_active", SQLValue: sqlValue},
		},
		Child: p09ChildParts(),
	}
	root := partsAdapter[*p09Parent]{TableParts[*p09Parent]{
		Table:   "p09_parents",
		Columns: []Column{{Name: "id", DBType: "bigint"}},
		NewBuf:  func() Buf[*p09Parent] { return &p09ParentBuf{} },
	}}

	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p09_parents WHERE id = ?", 7)
	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
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
			sqlStr, args := buildP09Query(t, tt.sqlValue)
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

// buildP09FromSideQuery composes a complete one-edge island carrying a
// from-side EdgeWhere. The parent query exposes every column referenced by
// that condition, and composeIsland defines _sl_src, so the rendered SQL is
// independently executable against p09_parents and p09_children.
func buildP09FromSideQuery(t *testing.T, sqlValue string) (string, []any) {
	t.Helper()

	edge := Edge[*p09FromParent, *p09Child]{
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		Where: []EdgeWhere{
			{On: "from", Column: "has_active", SQLValue: sqlValue},
		},
		Child: p09ChildParts(),
	}

	root := partsAdapter[*p09FromParent]{TableParts[*p09FromParent]{
		Table: "p09_parents",
		Columns: []Column{
			{Name: "id", DBType: "bigint"},
			{Name: "data", DBType: "jsonb"},
			{Name: "has_active", DBType: "boolean"},
		},
		NewBuf: func() Buf[*p09FromParent] {
			return &p09FromParentBuf{}
		},
	}}

	plan, err := planIsland([]Node{NewNode(edge)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery(
		"SELECT id, data, has_active FROM p09_parents WHERE id = ?",
		7,
	)
	q, err := composeIsland(
		parent,
		root,
		plan,
		DefaultMaxColumns,
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
			sqlStr, args := buildP09FromSideQuery(t, tt.sqlValue)

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
				`SELECT id, data, has_active FROM p09_parents WHERE id = $1`,
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
	sqlStr, args := buildP09Query(t, "'active'")
	if !strings.Contains(sqlStr, "= 'active'") {
		t.Fatalf("composed SQL does not contain \"= 'active'\":\n%s", sqlStr)
	}
	if len(args) != 1 || args[0] != 7 {
		t.Fatalf("args = %v, want exactly [7]", args)
	}
}

// TestP09FixtureColumnsMatchTargets is the regression test for issue P-09's
// round-3 Important-2 finding (Codex review, 2026-08-09): p09ChildParts'
// Columns must stay in 1:1 order/count correspondence with p09ChildBuf's
// Targets() (parts.go's generator contract). buildP09Query and
// buildP09FromSideQuery both build no-DB queries today, so a drift here
// would build clean and pass silently — see p09ChildParts' doc comment for
// why that is not actually harmless.
func TestP09FixtureColumnsMatchTargets(t *testing.T) {
	parts := p09ChildParts()

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
