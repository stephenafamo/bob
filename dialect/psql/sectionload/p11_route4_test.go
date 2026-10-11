package sectionload

// P-11 characterization: buildEdgeSelect route 4 (a Then(...) boundary's
// parent query, matched against explicit parent key tuples).
//
// pkg/dbsections' TestSectionloadBuild (via sectionload.Build) already
// exercises routes 1/2/3 without a DB — the "siblings and nested" tree
// (Server + Cve(MasterCve)) produces a with-children node's CTE (route 1),
// a leaf node's semi-join UNION branch (route 2, for Server and for
// MasterCve against Cve's CTE) and a with-children node's own-CTE UNION
// branch (route 3, for Cve). Build() cannot reach route 4: it renders only
// the first island, and a Then(...) boundary's parent query only exists
// once real parent keys have been collected at runtime (run.go's own doc
// comment). So route 4 needs its own no-DB test, calling buildEdgeSelect
// directly with a synthetic edge and literal keys.
//
// Baseline (pre-refactor thenChildQuery(p11CapEdge(), [][]any{{1},{2},{3}})),
// captured before this file was rewritten to test buildEdgeSelect:
//
//	SQL:  SELECT "id", "parent_id" FROM "p11_children" WHERE ("parent_id") IN ((?), (?), (?)) AND "status" = 'active'
//	Args: [1 2 3]  (len 3)
//
// buildEdgeSelect must match args count/order/values exactly (A-11-1); the
// SQL text is allowed to differ (identifier quoting, structure) as long as
// the row-value IN / AND fragments are present (A-11-5).

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
)

type p11CapParent struct {
	ID     int
	Region string
}

type p11CapChild struct {
	ID           int
	ParentID     int
	ParentRegion string
	HasActive    bool
}

// p11CapEdge is the single-column-key baseline fixture (issue P-11's
// original characterization). Its Child.Columns lists exactly 2 columns
// (id, parent_id), so its NewBuf must return a Buf whose Targets() also has
// exactly 2 entries in the same order — p11CapBaseChildBuf below, not the
// 4-field p11CapChildBuf used by the composite-key jsonb fixture. Sharing
// the 4-field Buf here was a P-09 round-3 Codex review finding (Important
// 2): TableParts.Columns/Buf.Targets() must stay 1:1 per parts.go's
// generator contract, or a future DB-executing test would fail at
// rows.Scan with "expected 2 destination arguments in Scan, not 4" the
// moment this baseline fixture were pointed at a live row.
func p11CapEdge() Edge[p11CapParent, p11CapChild] {
	return Edge[p11CapParent, p11CapChild]{
		RelName:  "Child",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Where:    []EdgeWhere{{On: "to", Column: "status", SQLValue: "'active'"}},
		Child: TableParts[p11CapChild]{
			Schema: "",
			Table:  "p11_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
			},
			NewBuf: func() Buf[p11CapChild] { return &p11CapBaseChildBuf{} },
		},
		ParentKey: func(p p11CapParent) []any { return []any{p.ID} },
		ChildKey:  func(c p11CapChild) []any { return []any{c.ParentID} },
		Stitch:    func(p p11CapParent, cs []p11CapChild) error { return nil },
	}
}

// p11CapBaseChildBuf is the 2-column scan buffer for p11CapEdge's baseline
// fixture (id, parent_id only). Kept separate from the 4-field
// p11CapChildBuf so the two fixtures' Columns/Targets() counts can never
// drift apart from each other by sharing one Buf type.
type p11CapBaseChildBuf struct {
	id       int
	parentID int
}

func (b *p11CapBaseChildBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.parentID} }

func (b *p11CapBaseChildBuf) BuildFor(modelCols []Column) p11CapChild {
	return p11CapChild{ID: b.id, ParentID: b.parentID}
}

// p11CapChildBuf is the 4-column scan buffer for the composite-key jsonb
// fixture (p11CompositeJSONBEdge below): id, parent_id, parent_region,
// has_active, matching that fixture's 4-entry Child.Columns 1:1.
type p11CapChildBuf struct {
	id           int
	parentID     int
	parentRegion string
	hasActive    bool
}

func (b *p11CapChildBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.parentID, &b.parentRegion, &b.hasActive}
}

func (b *p11CapChildBuf) BuildFor(modelCols []Column) p11CapChild {
	return p11CapChild{
		ID:           b.id,
		ParentID:     b.parentID,
		ParentRegion: b.parentRegion,
		HasActive:    b.hasActive,
	}
}

// p11CompositeJSONBEdge builds the composite-key Then-boundary edge used by
// TestP11Route4ThenBoundary_CompositeKeysWithJSONBOperator and
// TestP11Route4FixtureColumnsMatchTargets. column and sqlValue are
// parameterized so a caller can perturb the "to" condition's target column
// (e.g. for the Important-1 negative-side check: pointing it at the
// integer column "parent_id" instead of the boolean column "has_active"
// must make the LHS-type assertion fail) without duplicating the whole
// fixture.
func p11CompositeJSONBEdge(column, sqlValue string) Edge[p11CapParent, p11CapChild] {
	return Edge[p11CapParent, p11CapChild]{
		RelName:  "Child",
		FromCols: []string{"id", "region"},
		ToCols:   []string{"parent_id", "parent_region"},
		ToMany:   true,
		Where: []EdgeWhere{{
			On:       "to",
			Column:   column,
			SQLValue: sqlValue,
		}},
		Child: TableParts[p11CapChild]{
			Table: "p11_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "parent_region", DBType: "text"},
				{Name: "has_active", DBType: "boolean"},
			},
			NewBuf: func() Buf[p11CapChild] { return &p11CapChildBuf{} },
		},
		ParentKey: func(p p11CapParent) []any { return []any{p.ID, p.Region} },
		ChildKey:  func(c p11CapChild) []any { return []any{c.ParentID, c.ParentRegion} },
		Stitch:    func(p p11CapParent, cs []p11CapChild) error { return nil },
	}
}

func TestP11Route4ThenBoundaryParentQuery(t *testing.T) {
	e := p11CapEdge()
	childRoot := e.child()
	keys := [][]any{{1}, {2}, {3}}

	q, err := buildEdgeSelect(edgeSelectSpec{
		edge:  e,
		cols:  childRoot.cols(),
		from:  tableExpr(childRoot.schemaName(), childRoot.tableName()),
		alias: srcAlias,
		keys:  keys,
	})
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}

	sql, args, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	t.Logf("SQL: %s", sql)
	t.Logf("Args: %v", args)

	// args: exact count/order/values must match the pre-refactor baseline.
	wantArgs := []any{1, 2, 3}
	if len(args) != len(wantArgs) {
		t.Fatalf("args count: got %d want %d (args=%v)", len(args), len(wantArgs), args)
	}
	for i, want := range wantArgs {
		got, ok := args[i].(int)
		if !ok || got != want {
			t.Fatalf("args[%d]: got %v want %v", i, args[i], want)
		}
	}

	// Structural fragments (A-11-5), tightened (P-11 Minor 2, Codex review
	// 2026-08-09): the original checks only confirmed that substrings like
	// `"status"` or "IN" appeared SOMEWHERE in the SQL, which would still
	// pass if the WHERE condition's right-hand side were dropped or
	// altered (e.g. `"status" = 'inactive'` or a join instead of a
	// semi-join). These now assert the exact rendered fragments, including
	// the fixed condition's right-hand-side literal and the FROM clause's
	// join method (a plain FROM + WHERE IN, not a JOIN), so a regression in
	// either would fail here instead of silently passing. Exact identifier
	// quoting/casing (bob's own formatting) is still not asserted, since
	// that is bob's concern, not buildEdgeSelect's.
	//
	// Tightened further (P-11 round 2 Minor 1, Codex review 2026-08-09): the
	// SELECT-column-list and FROM-clause checks used to live as two more
	// entries in the frag loop below, checked with strings.Contains. That is
	// a substring search: it does not anchor position and does not forbid
	// extra content elsewhere in the string, so it would still pass if bob
	// rendered an extra selected column or an extra JOIN anywhere else in
	// the SQL. Replace those two entries with a single strings.HasPrefix
	// assert against the exact rendered SELECT/FROM/WHERE structure —
	// anchored at the very start of the string, including bob's actual
	// newline placement after "SELECT" and after the column list — plus an
	// explicit assert that no " JOIN " token appears anywhere in the SQL:
	// route 4's literal-key matching must stay a plain FROM + WHERE IN (a
	// semi-join expressed as a WHERE clause), never an actual SQL JOIN.
	wantPrefix := "SELECT \n\"" + srcAlias + "\".\"id\", \"" +
		srcAlias + "\".\"parent_id\"\nFROM \"p11_children\" AS \"" +
		srcAlias + "\"\nWHERE "
	if !strings.HasPrefix(sql, wantPrefix) {
		t.Fatalf("unexpected SELECT/FROM structure:\n got: %q\nwant prefix: %q", sql, wantPrefix)
	}
	if strings.Contains(strings.ToUpper(sql), " JOIN ") {
		t.Fatalf("route4 literal-key matching must not introduce JOIN:\n%s", sql)
	}

	for _, frag := range []string{
		`("` + srcAlias + `"."parent_id") IN (($1), ($2), ($3))`, // exact row-value IN, one placeholder group per key
		`"` + srcAlias + `"."status" = 'active'`,                 // fixed edgeWhere "to" condition, right-hand side included
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
	if placeholders := strings.Count(sql, "$"); placeholders != len(wantArgs) {
		t.Fatalf("placeholder count: got %d want %d; SQL:\n%s", placeholders, len(wantArgs), sql)
	}
}

// TestP11Route4ThenBoundary_CompositeKeysWithJSONBOperator is the
// regression test for issue P-09's Important-1 finding (Codex round 2
// review, 2026-08-09): the original route4 test above only ever used a
// single-column key and an ordinary (no "?") fixed condition. Its
// jsonb-operator variant, `(data ? 'active')`, also does not actually type-
// check against PostgreSQL: `data` is a bare identifier with no such column
// in scope, and `jsonb ? text` returns boolean while it was ANDed as if it
// were a boolean-valued EQ target's right-hand side without a boolean
// column on the left. This rewrite uses a SQLValue that is valid in
// isolation (a boolean-returning jsonb expression, matched against a real
// boolean column `has_active`) and drives all three jsonb existence
// operators from issue §8/§11.4 — `?` (exists), `?|` (any-of), `?&`
// (all-of) — through the same Then-boundary composite-key path so the
// placeholder-miscount regression (P-09) is covered for all three, not just
// one.
//
// Round-3 Codex review (2026-08-09, Important 1): the round-2 version of
// this test only asserted that the SQLValue string and placeholder count
// survived — it never asserted which column the SQLValue was compared
// against. Codex proved this gap by changing Column from "has_active" to
// "parent_id" (an integer column) and watching all three subtests pass
// anyway, even though the generated SQL would then read
// `"parent_id" = ('{}'::jsonb ? 'active')` — an integer/boolean type
// mismatch PostgreSQL would reject. This version additionally asserts the
// exact `"<alias>"."has_active" = <SQLValue>` fragment and the fixture's
// has_active column DBType, so that specific mistake fails loudly here.
func TestP11Route4ThenBoundary_CompositeKeysWithJSONBOperator(t *testing.T) {
	// PostgreSQLで実際に型が成立するSQLValueを使う: 左辺は boolean 列
	// has_active、右辺は jsonb 演算子式(boolean を返す)。issue §8/§11.4
	// の3演算子(? / ?| / ?&)をすべて通す(P-09 round 2 Important 1)。
	tests := []struct {
		name     string
		sqlValue string
	}{
		{"exists", `('{}'::jsonb ? 'active')`},
		{"any-of", `('{}'::jsonb ?| array['a','b'])`},
		{"all-of", `('{}'::jsonb ?& array['a','b'])`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := p11CompositeJSONBEdge("has_active", tt.sqlValue)

			// Fixture self-check (P-09 round 3 Important 1): the LHS column
			// the SQLValue is compared against must actually be boolean —
			// jsonb ? / ?| / ?& all return boolean, so a non-boolean LHS
			// would be a type error no string/placeholder assertion below
			// would ever catch.
			var hasActiveDBType string
			var found bool
			for _, c := range e.Child.Columns {
				if c.Name == "has_active" {
					hasActiveDBType, found = c.DBType, true
				}
			}
			if !found || hasActiveDBType != "boolean" {
				t.Fatalf("fixture bug: has_active column DBType = %q (found=%v), want \"boolean\" — "+
					"the SQLValue's jsonb operators only ever return boolean", hasActiveDBType, found)
			}

			keys := [][]any{{1, "us"}, {2, "eu"}}
			childRoot := e.child()
			q, err := buildEdgeSelect(edgeSelectSpec{
				edge:  e,
				cols:  childRoot.cols(),
				from:  tableExpr(childRoot.schemaName(), childRoot.tableName()),
				alias: srcAlias,
				keys:  keys,
			})
			if err != nil {
				t.Fatalf("buildEdgeSelect: %v", err)
			}

			sql, args, err := bob.Build(context.Background(), q)
			if err != nil {
				t.Fatalf("build: %v", err)
			}

			// The LHS-type check above only proves the fixture's own
			// declared DBType is boolean; it says nothing about which
			// column the composed SQL actually compares tt.sqlValue
			// against. Assert the full "<alias>"."has_active" = <SQLValue>
			// fragment so a Column typo (e.g. "parent_id" instead of
			// "has_active") fails here instead of silently passing.
			wantFrag := fmt.Sprintf(`"%s"."has_active" = %s`, srcAlias, tt.sqlValue)
			if !strings.Contains(sql, wantFrag) {
				t.Fatalf("SQL must compare has_active against the SQLValue verbatim; missing fragment %q:\n%s", wantFrag, sql)
			}
			wantArgs := []any{1, "us", 2, "eu"}
			if !reflect.DeepEqual(args, wantArgs) {
				t.Fatalf("args = %#v, want %#v", args, wantArgs)
			}
			if got := strings.Count(sql, "$"); got != len(wantArgs) {
				t.Fatalf("placeholder count = %d, want %d — a miscounted '?' shifts every binding:\n%s",
					got, len(wantArgs), sql)
			}
		})
	}
}

// TestP11Route4FixtureColumnsMatchTargets is the regression test for issue
// P-09's round-3 Important-2 finding (Codex review, 2026-08-09): the
// generator contract in parts.go requires TableParts.Columns and
// Buf.Targets() to stay in 1:1 order/count correspondence, but nothing
// mechanically checked it for either route4 fixture. Both fixtures are
// no-DB tests today (buildEdgeSelect/bob.Build never call NewBuf/Targets),
// so a drift would build clean and pass silently — it would only surface
// the moment a future test actually scanned rows through this fixture
// (rows.Scan failing with a destination-count mismatch). This test makes
// that invariant explicit for both p11CapEdge (single-key baseline) and
// p11CompositeJSONBEdge (composite-key jsonb fixture).
func TestP11Route4FixtureColumnsMatchTargets(t *testing.T) {
	t.Run("single-key-baseline", func(t *testing.T) {
		e := p11CapEdge()
		wantNames := []string{"id", "parent_id"}

		if len(e.Child.Columns) != len(wantNames) {
			t.Fatalf(
				"len(Columns)=%d, want %d",
				len(e.Child.Columns),
				len(wantNames),
			)
		}
		for i, want := range wantNames {
			if got := e.Child.Columns[i].Name; got != want {
				t.Fatalf("Columns[%d].Name=%q, want %q", i, got, want)
			}
		}

		buf := e.Child.NewBuf()
		targets := buf.TargetsFor(nil)
		if len(targets) != len(wantNames) {
			t.Fatalf(
				"len(Targets())=%d, len(Columns)=%d",
				len(targets),
				len(wantNames),
			)
		}

		id, ok := targets[0].(*int)
		if !ok {
			t.Fatalf("Targets()[0] has type %T, want *int for id", targets[0])
		}
		parentID, ok := targets[1].(*int)
		if !ok {
			t.Fatalf(
				"Targets()[1] has type %T, want *int for parent_id",
				targets[1],
			)
		}
		*id = 11
		*parentID = 22

		got := buf.BuildFor(nil)
		if got.ID != 11 || got.ParentID != 22 {
			t.Fatalf("Build()=%#v; Targets order does not match Columns order", got)
		}
	})

	t.Run("composite-jsonb", func(t *testing.T) {
		e := p11CompositeJSONBEdge(
			"has_active",
			`('{}'::jsonb ? 'active')`,
		)
		wantColumns := []struct {
			name   string
			dbType string
		}{
			{"id", "int4"},
			{"parent_id", "int4"},
			{"parent_region", "text"},
			{"has_active", "boolean"},
		}

		if len(e.Child.Columns) != len(wantColumns) {
			t.Fatalf(
				"len(Columns)=%d, want %d",
				len(e.Child.Columns),
				len(wantColumns),
			)
		}
		for i, want := range wantColumns {
			got := e.Child.Columns[i]
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

		buf := e.Child.NewBuf()
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
			t.Fatalf(
				"Targets()[1] has type %T, want *int for parent_id",
				targets[1],
			)
		}
		parentRegion, ok := targets[2].(*string)
		if !ok {
			t.Fatalf(
				"Targets()[2] has type %T, want *string for parent_region",
				targets[2],
			)
		}
		hasActive, ok := targets[3].(*bool)
		if !ok {
			t.Fatalf(
				"Targets()[3] has type %T, want *bool for has_active",
				targets[3],
			)
		}

		*id = 11
		*parentID = 22
		*parentRegion = "us"
		*hasActive = true

		got := buf.BuildFor(nil)
		if got.ID != 11 ||
			got.ParentID != 22 ||
			got.ParentRegion != "us" ||
			!got.HasActive {
			t.Fatalf("Build()=%#v; Targets order does not match Columns order", got)
		}
	})
}
