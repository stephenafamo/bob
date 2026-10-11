package sectionload

// R-24 (P-34-multihop-count.md): SQL build / plan tests for multi-hop
// (through) COUNT edges, DB-free (bob.Build only, no execution) — the
// same style p05_sql_test.go established for multi-hop ROW edges.
//
// Fixture shape (reuses p05_sql_test.go's p05Parent/p05Leaf/p05Grand):
//
//	p05Parent --(Sides[0])--> p05_mid --(Sides[1])--> p05Leaf   (T-1, T-2's 2-hop leg)
//	p05_a --(Sides[1])--> p05_b --(Sides[2])--> p05Leaf          (T-2's 3rd hop)
//	p05Leaf --(Sides[0])--> p05_leaf_mid --(Sides[1])--> p05Grand (T-11's nested count)

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// --- fixtures -----------------------------------------------------------

// p05GrandParts is p05Grand's TableParts, factored out of
// p05LeafGrandEdge (p05_sql_test.go) so r24 fixtures can also use it as a
// count edge's Child.
func p05GrandParts() TableParts[p05Grand] {
	return TableParts[p05Grand]{
		Table:   "p05_grands",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p05Grand] { return &p05GrandBuf{} },
	}
}

// p05TwoHopCountEdge: the CountEdge counterpart of p05TwoHopEdge
// (p05_sql_test.go) — same p05_parents -> p05_mid -> p05_leaves hop
// chain, but counting rows instead of loading them.
func p05TwoHopCountEdge() CountEdge[p05Parent, p05Leaf] {
	return CountEdge[p05Parent, p05Leaf]{
		RelName: "LeavesCount",
		Sides: []Side{
			{FromTable: "p05_parents", FromCols: []string{"id"}, ToTable: "p05_mid", ToCols: []string{"parent_id"}},
			{FromTable: "p05_mid", FromCols: []string{"leaf_id"}, ToTable: "p05_leaves", ToCols: []string{"id"}},
		},
		Child:         p05LeafParts(),
		KeyCols:       []Column{{Name: "parent_id", DBType: "int4"}},
		NewCarriedBuf: func() Buf[p05Parent] { return &p05ParentBuf{} },
		ParentKey:     func(p p05Parent) []any { return []any{p.ID} },
		ChildKey:      func(c p05Leaf) []any { return []any{c.ID} },
		SetCount:      func(p p05Parent, n int64) error { return nil },
	}
}

// p05ThreeHopCountEdge: the CountEdge counterpart of p05ThreeHopEdge
// (p05_sql_test.go) — p05_parents -> p05_a -> p05_b -> p05_leaves.
func p05ThreeHopCountEdge() CountEdge[p05Parent, p05Leaf] {
	return CountEdge[p05Parent, p05Leaf]{
		RelName: "ThreeHopLeavesCount",
		Sides: []Side{
			{FromTable: "p05_parents", FromCols: []string{"id"}, ToTable: "p05_a", ToCols: []string{"parent_id"}},
			{FromTable: "p05_a", FromCols: []string{"b_id"}, ToTable: "p05_b", ToCols: []string{"id"}},
			{FromTable: "p05_b", FromCols: []string{"leaf_id"}, ToTable: "p05_leaves", ToCols: []string{"id"}},
		},
		Child:         p05LeafParts(),
		KeyCols:       []Column{{Name: "parent_id", DBType: "int4"}},
		NewCarriedBuf: func() Buf[p05Parent] { return &p05ParentBuf{} },
		ParentKey:     func(p p05Parent) []any { return []any{p.ID} },
		ChildKey:      func(c p05Leaf) []any { return []any{c.ID} },
		SetCount:      func(p p05Parent, n int64) error { return nil },
	}
}

// p05NestedMultiHopCountEdge: a 2-hop count edge parented on p05Leaf
// (NOT the island's true root p05Parent) — p05_leaves -> p05_leaf_mid ->
// p05_grands. Used by T-11 to exercise a multi-hop count nested UNDER a
// row section, so its own semi-join targets that row section's own CTE
// (_sl_n<set>), not the island's root (_sl_src).
func p05NestedMultiHopCountEdge() CountEdge[p05Leaf, p05Grand] {
	return CountEdge[p05Leaf, p05Grand]{
		RelName: "GrandsCount",
		Sides: []Side{
			{FromTable: "p05_leaves", FromCols: []string{"id"}, ToTable: "p05_leaf_mid", ToCols: []string{"leaf_id"}},
			{FromTable: "p05_leaf_mid", FromCols: []string{"grand_id"}, ToTable: "p05_grands", ToCols: []string{"id"}},
		},
		Child:         p05GrandParts(),
		KeyCols:       []Column{{Name: "leaf_id", DBType: "int4"}},
		NewCarriedBuf: func() Buf[p05Leaf] { return &p05LeafBuf{} },
		ParentKey:     func(p p05Leaf) []any { return []any{p.ID} },
		ChildKey:      func(c p05Grand) []any { return []any{c.ID} },
		SetCount:      func(p p05Leaf, n int64) error { return nil },
	}
}

// --- tests ---------------------------------------------------------------

// TestR24TwoHopCount_JoinChainAndGroupByAlias (T-1): a top-level 2-hop
// count edge's SQL — exactly one INNER JOIN, and (unlike a row edge's
// carried-key SELECT, which aliases the value as "_sl_k0") the SELECT and
// GROUP BY both quote the FIRST intermediate table's own alias/column
// name directly, with no AS rename (§6.2/§6.4 of P-34-multihop-count.md:
// a count section's SELECT never adds a carried-style SQL column).
func TestR24TwoHopCount_JoinChainAndGroupByAlias(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
		NewCountNode(p05TwoHopCountEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	if n := strings.Count(sql, "INNER JOIN"); n != 1 {
		t.Fatalf("want exactly 1 INNER JOIN, got %d:\n%s", n, sql)
	}
	for _, frag := range []string{
		`FROM "p05_leaves" AS "_sl_t"`,
		`INNER JOIN "p05_mid" AS "_sl_h1" ON ("_sl_t"."id" = "_sl_h1"."leaf_id")`,
		// SELECT: the first intermediate table's own column, no AS rename
		// (unlike a row edge's carried key, which IS renamed to _sl_k0).
		`"_sl_h1"."parent_id"`,
		`count(*)`,
		// GROUP BY: same alias/column as the SELECT.
		`GROUP BY "_sl_h1"."parent_id"`,
		// The semi-join match column is ALSO quoted against _sl_h1, not
		// srcAlias (_sl_t) — the group key lives on the intermediate table.
		`WHERE (("_sl_h1"."parent_id") IN ((SELECT`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
	// The count section must never rename its SELECT column to a
	// "_sl_k<i>"-style carried name — that mechanism is for row edges only.
	if strings.Contains(sql, `AS "_sl_k0"`) {
		t.Fatalf("a count section must not add a carried-key SQL column (_sl_k0):\n%s", sql)
	}
}

// TestR24ThreeHopCount_JoinChainGeneralizes (T-2): the 2-hop shape above
// must generalize to 3 hops without new code — 2 JOINs, GROUP BY/SELECT
// still against _sl_h1 (Sides[0].To's alias, still the FIRST intermediate
// table for n==3 — same "stop condition 3" p05_sql_test.go's 3-hop row
// test already pins for row edges).
func TestR24ThreeHopCount_JoinChainGeneralizes(t *testing.T) {
	edge := p05ThreeHopCountEdge()
	sides := edge.sides()
	if len(sides) != 3 {
		t.Fatalf("len(sides()) = %d, want 3", len(sides))
	}

	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
		NewCountNode(edge),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	if n := strings.Count(sql, "INNER JOIN"); n != 2 {
		t.Fatalf("want exactly 2 INNER JOIN (3 hops = 2 joins), got %d:\n%s", n, sql)
	}
	for _, frag := range []string{
		`FROM "p05_leaves" AS "_sl_t"`,
		`INNER JOIN "p05_b" AS "_sl_h2" ON ("_sl_t"."id" = "_sl_h2"."leaf_id")`,
		`INNER JOIN "p05_a" AS "_sl_h1" ON ("_sl_h2"."id" = "_sl_h1"."b_id")`,
		`"_sl_h1"."parent_id"`,
		`GROUP BY "_sl_h1"."parent_id"`,
		`WHERE (("_sl_h1"."parent_id") IN ((SELECT`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
}

// TestR24NestedMultiHopCount_SemiJoinsParentSectionCTE (T-11): a
// multi-hop count NESTED under a row section (not the island's true
// root) must semi-join against that row section's own CTE (_sl_n<set>),
// not the island root (_sl_src) — parentKeySubquery/matchWhere's
// semiFromCTE is set from countPlan.parentSet, which planIsland resolves
// to whatever node this count is declared under (compose.go's
// semiFromCTE: cteName(c.parentSet), countBranch). GROUP BY/SELECT still
// target the count's own first-intermediate-table alias (_sl_h1), the
// SAME as the top-level case (T-1) — nesting only changes WHERE this
// count matches against, never how its own JOIN chain/GROUP BY render.
func TestR24NestedMultiHopCount_SemiJoinsParentSectionCTE(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM p05_parents"),
		p05ParentParts(),
		NewNode(p05TwoHopEdge(), NewCountNode(p05NestedMultiHopCountEdge())),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	// The nested count's own JOIN chain/GROUP BY alias, unchanged from the
	// top-level case.
	for _, frag := range []string{
		`FROM "p05_grands" AS "_sl_t"`,
		`INNER JOIN "p05_leaf_mid" AS "_sl_h1" ON ("_sl_t"."id" = "_sl_h1"."grand_id")`,
		`"_sl_h1"."leaf_id"`,
		`GROUP BY "_sl_h1"."leaf_id"`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
	// The semi-join must target the PARENT SECTION's own CTE (_sl_n1, the
	// materialized p05TwoHopEdge section this count is nested under), not
	// the island root (_sl_src) — the discriminating assertion for T-11.
	// The nested count's own subquery (FROM "p05_grands" ... GROUP BY
	// "_sl_h1"."leaf_id") is the SECOND "IN ((SELECT" block in the SQL —
	// the first belongs to the row section (_sl_n1's own CTE body, which
	// semi-joins against _sl_src) — so this asserts on that second block
	// specifically, not merely "_sl_n1 appears somewhere".
	if idx := strings.LastIndex(sql, `IN ((SELECT`); idx == -1 || !strings.Contains(sql[idx:], `FROM "_sl_n1"`) {
		t.Fatalf("nested count must semi-join against its parent section's own CTE (_sl_n1), not _sl_src:\n%s", sql)
	} else if strings.Contains(sql[idx:], `FROM "_sl_src"`) {
		t.Fatalf("nested count must NOT semi-join against the island root (_sl_src):\n%s", sql)
	}
}
