package sectionload

// SQL build / plan tests for multi-hop (through) relationships, DB-free
// (bob.Build only, no execution) — the same style as the other white-box
// tests: hand-rolled Edge/TableParts/Buf fixtures, bob.Build to inspect
// rendered SQL.
//
// Fixture shape (reused across every test in this file):
//
//	multiHopParent --(Sides[0])--> mh_mid --(Sides[1])--> multiHopLeaf
//
// A 2-hop Edge (multiHopTwoHopEdge) exercises the base case; a 3-hop Edge
// (multiHopThreeHopEdge, multiHopParent -> mh_a -> mh_b -> multiHopLeaf) exercises the
// requirement that "the same structure must work for 3 hops, rather than
// being 2-hop-only code". A composite-key
// variant (multiHopTwoHopCompositeEdge) exercises 2 carried columns landing
// side by side. multiHopLeafGrandEdge nests a plain single-hop edge under the
// 2-hop edge to exercise route 3 (CTE body already joined; the UNION ALL
// read-back must not re-JOIN).

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

// --- shared fixtures -------------------------------------------------

type (
	multiHopParent    struct{ ID int }
	multiHopParentBuf struct{ id int }
)

func (b *multiHopParentBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *multiHopParentBuf) BuildFor(modelCols []Column) multiHopParent {
	return multiHopParent{ID: b.id}
}

func multiHopParentParts() TableParts[multiHopParent] {
	return TableParts[multiHopParent]{
		Table:   "mh_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[multiHopParent] { return &multiHopParentBuf{} },
	}
}

type multiHopLeaf struct {
	ID    int
	Label string
}
type multiHopLeafBuf struct {
	id    int
	label string
}

func (b *multiHopLeafBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.label} }
func (b *multiHopLeafBuf) BuildFor(modelCols []Column) multiHopLeaf {
	return multiHopLeaf{ID: b.id, Label: b.label}
}

func multiHopLeafParts() TableParts[multiHopLeaf] {
	return TableParts[multiHopLeaf]{
		Table: "mh_leaves",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "label", DBType: "text"},
		},
		NewBuf: func() Buf[multiHopLeaf] { return &multiHopLeafBuf{} },
	}
}

// multiHopTwoHopEdge: mh_parents -> mh_mid -> mh_leaves, single-column key.
func multiHopTwoHopEdge() Edge[multiHopParent, multiHopLeaf] {
	return Edge[multiHopParent, multiHopLeaf]{
		RelName: "Leaves",
		Sides: []Side{
			{FromTable: "mh_parents", FromCols: []string{"id"}, ToTable: "mh_mid", ToCols: []string{"parent_id"}},
			{FromTable: "mh_mid", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
		},
		ToMany:        true,
		Child:         multiHopLeafParts(),
		Carried:       []Column{{Name: "_sl_k0", DBType: "int4"}},
		NewCarriedBuf: func() Buf[multiHopParent] { return &multiHopParentBuf{} },
		ParentKey:     func(p multiHopParent) []any { return []any{p.ID} },
		ChildKey:      func(c multiHopLeaf) []any { return []any{c.ID} },
		Stitch:        func(p multiHopParent, cs []multiHopLeaf) error { return nil },
	}
}

// multiHopTwoHopEdgeWithWhere: same shape as multiHopTwoHopEdge, plus a fixed
// condition on EACH side (side 0's own to/from and side 1's own
// from/to), to pin where each lands:
//
//	Sides[0].FromWhere -> parent-key subquery's own WHERE (no alias)
//	Sides[0].ToWhere   -> matchAlias (hopAlias(1) here)
//	Sides[1].FromWhere -> hopAlias(1) (Sides[1].From's own alias), in the JOIN's ON
//	Sides[1].ToWhere   -> srcAlias (Sides[1].To == the final table), in the JOIN's ON
func multiHopTwoHopEdgeWithWhere() Edge[multiHopParent, multiHopLeaf] {
	e := multiHopTwoHopEdge()
	e.Sides = []Side{
		{
			FromTable: "mh_parents", FromCols: []string{"id"},
			ToTable: "mh_mid", ToCols: []string{"parent_id"},
			Where: []EdgeWhere{
				{On: "from", Column: "is_active", SQLValue: "true"},
				{On: "to", Column: "mid_flag", SQLValue: "'y'"},
			},
		},
		{
			FromTable: "mh_mid", FromCols: []string{"leaf_id"},
			ToTable: "mh_leaves", ToCols: []string{"id"},
			Where: []EdgeWhere{
				{On: "from", Column: "mid_status", SQLValue: "'ready'"},
				{On: "to", Column: "leaf_status", SQLValue: "'ok'"},
			},
		},
	}
	return e
}

// multiHopThreeHopEdge: mh_parents -> mh_a -> mh_b -> mh_leaves. Exercises
// the alias/carried-key scheme must generalize
// past 2 hops without new code, not just read correctly for n==2.
func multiHopThreeHopEdge() Edge[multiHopParent, multiHopLeaf] {
	return Edge[multiHopParent, multiHopLeaf]{
		RelName: "ThreeHopLeaves",
		Sides: []Side{
			{FromTable: "mh_parents", FromCols: []string{"id"}, ToTable: "mh_a", ToCols: []string{"parent_id"}},
			{FromTable: "mh_a", FromCols: []string{"b_id"}, ToTable: "mh_b", ToCols: []string{"id"}},
			{FromTable: "mh_b", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
		},
		ToMany:        true,
		Child:         multiHopLeafParts(),
		Carried:       []Column{{Name: "_sl_k0", DBType: "int4"}},
		NewCarriedBuf: func() Buf[multiHopParent] { return &multiHopParentBuf{} },
		ParentKey:     func(p multiHopParent) []any { return []any{p.ID} },
		ChildKey:      func(c multiHopLeaf) []any { return []any{c.ID} },
		Stitch:        func(p multiHopParent, cs []multiHopLeaf) error { return nil },
	}
}

// multiHopCompositeParent/multiHopCompositeParentBuf: a 2-column key parent, for the
// composite-key 2-hop fixture.
type (
	multiHopCompositeParent    struct{ ID1, ID2 int }
	multiHopCompositeParentBuf struct{ id1, id2 int }
)

func (b *multiHopCompositeParentBuf) TargetsFor(cols []Column) []any { return []any{&b.id1, &b.id2} }

func (b *multiHopCompositeParentBuf) BuildFor(modelCols []Column) multiHopCompositeParent {
	return multiHopCompositeParent{ID1: b.id1, ID2: b.id2}
}

// multiHopTwoHopCompositeEdge: same mh_mid -> mh_leaves final hop, but side 0
// carries a 2-column key (id1, id2), so 2 carried columns (_sl_k0,
// _sl_k1) must land side by side.
func multiHopTwoHopCompositeEdge() Edge[multiHopCompositeParent, multiHopLeaf] {
	return Edge[multiHopCompositeParent, multiHopLeaf]{
		RelName: "CompositeLeaves",
		Sides: []Side{
			{
				FromTable: "mh_composite_parents", FromCols: []string{"id1", "id2"},
				ToTable: "mh_mid", ToCols: []string{"parent_id1", "parent_id2"},
			},
			{FromTable: "mh_mid", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
		},
		ToMany: true,
		Child:  multiHopLeafParts(),
		Carried: []Column{
			{Name: "_sl_k0", DBType: "int4"},
			{Name: "_sl_k1", DBType: "int4"},
		},
		NewCarriedBuf: func() Buf[multiHopCompositeParent] { return &multiHopCompositeParentBuf{} },
		ParentKey:     func(p multiHopCompositeParent) []any { return []any{p.ID1, p.ID2} },
		ChildKey:      func(c multiHopLeaf) []any { return []any{c.ID} },
		Stitch:        func(p multiHopCompositeParent, cs []multiHopLeaf) error { return nil },
	}
}

// multiHopGrand/multiHopLeafGrandEdge: a plain single-hop edge nested UNDER the
// 2-hop edge, so multiHopTwoHopEdge's own section gets hasKids=true (route 1
// CTE body + route 3 read-back), instead of only ever being exercised as
// a leaf (route 2).
type (
	multiHopGrand    struct{ ID int }
	multiHopGrandBuf struct{ id int }
)

func (b *multiHopGrandBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *multiHopGrandBuf) BuildFor(modelCols []Column) multiHopGrand {
	return multiHopGrand{ID: b.id}
}

func multiHopLeafGrandEdge() Edge[multiHopLeaf, multiHopGrand] {
	return Edge[multiHopLeaf, multiHopGrand]{
		RelName:  "Grands",
		FromCols: []string{"id"},
		ToCols:   []string{"leaf_id"},
		ToMany:   true,
		Child: TableParts[multiHopGrand]{
			Table:   "mh_grands",
			Columns: []Column{{Name: "id", DBType: "int4"}},
			NewBuf:  func() Buf[multiHopGrand] { return &multiHopGrandBuf{} },
		},
		ParentKey: func(p multiHopLeaf) []any { return []any{p.ID} },
		ChildKey:  func(c multiHopGrand) []any { return []any{c.ID} },
		Stitch:    func(p multiHopLeaf, cs []multiHopGrand) error { return nil },
	}
}

// --- tests -------------------------------------------------------------

// TestMultiHopTwoHop_JoinChainAndCarriedKey pins the 2-hop shape: exactly one INNER JOIN, aliased
// "_sl_h1", the semi-join and carried key both read from that alias (not
// srcAlias — the final table has no parent-key column of its own), and
// the outer layout's expectedColumns ends with the carried column name.
func TestMultiHopTwoHop_JoinChainAndCarriedKey(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(multiHopTwoHopEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	if n := strings.Count(sql, "INNER JOIN"); n != 1 {
		t.Fatalf("want exactly 1 INNER JOIN, got %d:\n%s", n, sql)
	}
	for _, frag := range []string{
		`FROM "mh_leaves" AS "_sl_t"`,
		`INNER JOIN "mh_mid" AS "_sl_h1" ON ("_sl_t"."id" = "_sl_h1"."leaf_id")`,
		`"_sl_h1"."parent_id" AS "_sl_k0"`,
		`WHERE (("_sl_h1"."parent_id") IN ((SELECT`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}

	// expectedColumns: root cols, then this section's childCols() —
	// childScanCols() (id, label) followed by carriedCols() (_sl_k0).
	plan, err := planIsland([]Node{NewNode(multiHopTwoHopEdge())})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := partsAdapter[multiHopParent]{multiHopParentParts()}
	cols := expectedColumns(root, plan)
	if got, want := cols[len(cols)-1], "_sl_k0"; got != want {
		t.Fatalf("expectedColumns tail = %q, want %q (full: %v)", got, want, cols)
	}
	if got, want := cols[len(cols)-2], "label"; got != want {
		t.Fatalf("expectedColumns[-2] = %q, want %q (carried must come AFTER the model columns): %v", got, want, cols)
	}
}

// TestMultiHopTwoHop_HopWiseFixedConditions pins where each side's own fixed
// condition lands: Sides[0].From -> the parent-key
// subquery's own WHERE (unaliased); Sides[0].To -> matchAlias
// (hopAlias(1)); Sides[1].From/To -> the JOIN's own ON, on hopAlias(1)/
// srcAlias respectively (Sides[1].From is mh_mid == hopAlias(1);
// Sides[1].To is mh_leaves == the final table == srcAlias).
func TestMultiHopTwoHop_HopWiseFixedConditions(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(multiHopTwoHopEdgeWithWhere()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	for _, frag := range []string{
		// Sides[0].To ("mid_flag") — matchAlias (_sl_h1), ANDed onto the
		// semi-join condition (matchWhere's toWhere).
		`"_sl_h1"."mid_flag" = 'y'`,
		// Sides[0].From ("is_active") — the parent-key subquery's own
		// WHERE, unaliased (parentKeySubquery quotes columns bare).
		`"is_active" = true`,
		// Sides[1].From ("mid_status") — hopAlias(1), inside the JOIN ON.
		`INNER JOIN "mh_mid" AS "_sl_h1" ON ("_sl_t"."id" = "_sl_h1"."leaf_id") AND ("_sl_h1"."mid_status" = 'ready')`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
	// Sides[1].To ("leaf_status") — srcAlias (the final table itself),
	// inside the SAME JOIN's ON (order after mid_status is not asserted;
	// only presence and alias are).
	if !strings.Contains(sql, `"_sl_t"."leaf_status" = 'ok'`) {
		t.Fatalf("missing Sides[1].To fixed condition on srcAlias:\n%s", sql)
	}
	// The parent-key subquery's own WHERE must be inside the "IN (SELECT
	// ...)" subquery, not accidentally promoted to the outer query.
	if !strings.Contains(sql, `FROM "_sl_src"`) {
		t.Fatalf("parent-key subquery must still read from _sl_src:\n%s", sql)
	}
}

// TestMultiHopThreeHop_JoinChainGeneralizes is the direct check that the SAME alias/carried-key scheme used for 2 hops
// produces a correct, self-consistent 3-hop chain — 2 JOINs, aliased
// "_sl_h1"/"_sl_h2", the carried key read from "_sl_h1" (Sides[0].To's
// alias, still the FIRST intermediate table for n==3), and
// Sides[i-1].ToTable == Sides[i].FromTable actually
// holds for the fixture used (bob's own Relationship.Validate() invariant,
// pinned here at the sectionload-fixture level too).
func TestMultiHopThreeHop_JoinChainGeneralizes(t *testing.T) {
	edge := multiHopThreeHopEdge()
	sides := edge.sides()
	if len(sides) != 3 {
		t.Fatalf("len(sides()) = %d, want 3", len(sides))
	}
	for i := 1; i < len(sides); i++ {
		if sides[i-1].ToTable != sides[i].FromTable {
			t.Fatalf("Sides[%d].ToTable (%q) != Sides[%d].FromTable (%q); chain broken",
				i-1, sides[i-1].ToTable, i, sides[i].FromTable)
		}
	}

	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(edge),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	if n := strings.Count(sql, "INNER JOIN"); n != 2 {
		t.Fatalf("want exactly 2 INNER JOIN (3 hops = 2 joins), got %d:\n%s", n, sql)
	}
	for _, frag := range []string{
		`FROM "mh_leaves" AS "_sl_t"`,
		`INNER JOIN "mh_b" AS "_sl_h2" ON ("_sl_t"."id" = "_sl_h2"."leaf_id")`,
		`INNER JOIN "mh_a" AS "_sl_h1" ON ("_sl_h2"."id" = "_sl_h1"."b_id")`,
		// The carried key is Sides[0].ToCols, read from Sides[0].To's own
		// alias — which for n==3 is STILL hopAlias(1) (Sides[0].To ==
		// Sides[1].From == mh_a), not hopAlias(2).
		`"_sl_h1"."parent_id" AS "_sl_k0"`,
		`WHERE (("_sl_h1"."parent_id") IN ((SELECT`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
}

// TestMultiHopTwoHopComposite_BothCarriedColumnsLandSideBySide pins the
// composite-key case: 2 FromCols on side 0 means 2 carried columns
// (_sl_k0, _sl_k1), both read from the same matchAlias, in order.
func TestMultiHopTwoHopComposite_BothCarriedColumnsLandSideBySide(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id1, id2 FROM mh_composite_parents"),
		TableParts[multiHopCompositeParent]{
			Table: "mh_composite_parents",
			Columns: []Column{
				{Name: "id1", DBType: "int4"},
				{Name: "id2", DBType: "int4"},
			},
			NewBuf: func() Buf[multiHopCompositeParent] { return &multiHopCompositeParentBuf{} },
		},
		NewNode(multiHopTwoHopCompositeEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	for _, frag := range []string{
		`"_sl_h1"."parent_id1" AS "_sl_k0"`,
		`"_sl_h1"."parent_id2" AS "_sl_k1"`,
		`ON ("_sl_t"."id" = "_sl_h1"."leaf_id")`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
	// Both carried columns must appear in the physical SELECT list in
	// order, immediately after the model columns.
	wantOrder := `"_sl_t"."id", "_sl_t"."label", "_sl_h1"."parent_id1" AS "_sl_k0", "_sl_h1"."parent_id2" AS "_sl_k1"`
	if !strings.Contains(sql, wantOrder) {
		t.Fatalf("carried columns not in expected order after model columns:\nwant substring: %s\ngot SQL:\n%s", wantOrder, sql)
	}
}

// TestMultiHopRoute3_CTEHasJoinReadBackDoesNot pins route 3:
// when a multi-hop edge has its own children (hasKids=true), the JOIN
// chain and carried-key computation happen ONCE, in the CTE body (route
// 1) — the UNION ALL branch that reads the section's own rows back
// (route 3) must read the already-materialized carried column by its
// plain SQL-layout name ("_sl_t"."_sl_k0") and must NOT re-run the JOIN.
func TestMultiHopRoute3_CTEHasJoinReadBackDoesNot(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(multiHopTwoHopEdge(), NewNode(multiHopLeafGrandEdge())),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	// Exactly one INNER JOIN total: the CTE body's own (route 1). The
	// route-3 read-back and the grandchild's route-2 leaf select must add
	// none of their own.
	if n := strings.Count(sql, "INNER JOIN"); n != 1 {
		t.Fatalf("want exactly 1 INNER JOIN (only the CTE body's), got %d:\n%s", n, sql)
	}
	if !strings.Contains(sql, `INNER JOIN "mh_mid" AS "_sl_h1" ON`) {
		t.Fatalf("CTE body must contain the multi-hop edge's own JOIN:\n%s", sql)
	}
	// The CTE materializes the carried column under its final name.
	if !strings.Contains(sql, `"_sl_h1"."parent_id" AS "_sl_k0"`) {
		t.Fatalf("CTE body must carry the join key forward as _sl_k0:\n%s", sql)
	}
	// The route-3 read-back selects the CTE's own columns by plain name —
	// "_sl_t"."_sl_k0" (srcAlias here is the CTE's OWN alias in that
	// SELECT, not the final table) — never re-quoting a hop alias.
	if !strings.Contains(sql, `"_sl_t"."_sl_k0"`) {
		t.Fatalf("route-3 read-back must select the carried column back by its plain name (\"_sl_t\".\"_sl_k0\"):\n%s", sql)
	}
	if !strings.Contains(sql, `FROM "_sl_n1" AS "_sl_t"`) {
		t.Fatalf("route-3 read-back must read from the CTE (_sl_n1), not the raw table again:\n%s", sql)
	}
}

// TestMultiHopRoute4_MultiHopOwnCarriedColumnIsNotSelfRejected is a DB-free
// regression guard for a real bug a DB-backed test found (and
// this test would have caught without a DB): a multi-hop edge's route-4
// "parent query" (built here exactly as hooks.go's runThen builds it —
// buildEdgeSelect + edgeJoinChain, keys-matched) legitimately SELECTs its
// own carried-key column under the name "_sl_k0" — the same identifier
// parentexpr.go's reservedNameRe rejects a CALLER's parent query
// for using. Wrapping that query in trustedParentQuery (parentexpr.go)
// before handing it to composeIsland must exempt it from that check,
// while composeIsland fed an UNTRUSTED query containing the same
// substring must still reject it (the check itself must not be
// defeated, only correctly scoped to route 4's own output).
func TestMultiHopRoute4_MultiHopOwnCarriedColumnIsNotSelfRejected(t *testing.T) {
	edge := multiHopTwoHopEdge()
	childRoot := edgeRootParts{edge.child(), edge}
	joins, carried, matchAlias := edgeJoinChain(edge)
	parentQuery, err := buildEdgeSelect(edgeSelectSpec{
		edge:            edge,
		cols:            childRoot.scanCols(),
		from:            tableExpr(childRoot.schemaName(), childRoot.tableName()),
		alias:           srcAlias,
		joins:           joins,
		carried:         carried,
		matchAlias:      matchAlias,
		keys:            [][]any{{1}},
		collapseOrderBy: true,
	})
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}

	// Sanity: this route-4 query really does contain its own carried
	// column under the reserved-looking name — if it didn't, the trusted
	// wrap below would be proving nothing.
	rawSQL, _, err := bob.Build(context.Background(), parentQuery)
	if err != nil {
		t.Fatalf("bob.Build(parentQuery): %v", err)
	}
	if !strings.Contains(rawSQL, `AS "_sl_k0"`) {
		t.Fatalf("fixture precondition failed: route-4 query must select its own carried column as _sl_k0:\n%s", rawSQL)
	}

	plan, err := planIsland(nil)
	if err != nil {
		t.Fatalf("planIsland(nil): %v", err)
	}

	// Trusted: composeIsland must accept it and the carried column must
	// survive into the composed SQL.
	trustedQ, err := composeIsland(trustedParentQuery{parentQuery}, childRoot, plan, "")
	if err != nil {
		t.Fatalf("composeIsland(trustedParentQuery{...}): %v, want nil (route-4's own carried column must not self-reject)", err)
	}
	trustedSQL, _, err := bob.Build(context.Background(), trustedQ)
	if err != nil {
		t.Fatalf("bob.Build(trusted): %v", err)
	}
	if !strings.Contains(trustedSQL, "_sl_k0") {
		t.Fatalf("composed SQL must still carry the _sl_k0 column through:\n%s", trustedSQL)
	}

	// Untrusted: the SAME query, NOT wrapped, must still be rejected —
	// proving the exemption is scoped to the wrapper, not a change to
	// reservedNameRe or parentSource itself. The check only fires at
	// render time (parentSource.WriteSQL, called from bob.Build), not at
	// composeIsland's own construction time — composeIsland itself
	// returning nil here is expected and does not mean acceptance.
	untrustedQ, err := composeIsland(parentQuery, childRoot, plan, "")
	if err != nil {
		t.Fatalf("composeIsland(parentQuery) (untrusted): %v", err)
	}
	_, _, err = bob.Build(context.Background(), untrustedQ)
	var want *ReservedNameError
	if !errors.As(err, &want) {
		t.Fatalf("bob.Build(untrusted) = %v, want *ReservedNameError (the check itself must still fire for a non-trusted query)", err)
	}
	if want.Name != "_sl_k0" {
		t.Fatalf("ReservedNameError.Name = %q, want %q", want.Name, "_sl_k0")
	}
}
