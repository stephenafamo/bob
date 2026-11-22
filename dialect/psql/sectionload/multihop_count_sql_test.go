package sectionload

// SQL build / plan tests for multi-hop
// (through) COUNT edges, DB-free (bob.Build only, no execution) — the
// same style multihop_sql_test.go established for multi-hop ROW edges.
//
// Fixture shape (reuses multihop_sql_test.go's multiHopParent/multiHopLeaf/multiHopGrand):
//
//	multiHopParent --(Sides[0])--> mh_mid --(Sides[1])--> multiHopLeaf   (2-hop leg)
//	mh_a --(Sides[1])--> mh_b --(Sides[2])--> multiHopLeaf          (3rd hop)
//	multiHopLeaf --(Sides[0])--> mh_leaf_mid --(Sides[1])--> multiHopGrand (nested count)

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
)

// --- fixtures -----------------------------------------------------------

// multiHopGrandParts is multiHopGrand's TableParts, factored out of
// multiHopLeafGrandEdge (multihop_sql_test.go) so multi-hop count fixtures can also use it as a
// count edge's Child.
func multiHopGrandParts() TableParts[multiHopGrand] {
	return TableParts[multiHopGrand]{
		Table:   "mh_grands",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[multiHopGrand] { return &multiHopGrandBuf{} },
	}
}

// multiHopTwoHopCountEdge: the CountEdge counterpart of multiHopTwoHopEdge
// (multihop_sql_test.go) — same mh_parents -> mh_mid -> mh_leaves hop
// chain, but counting rows instead of loading them.
func multiHopTwoHopCountEdge() CountEdge[multiHopParent, multiHopLeaf] {
	return CountEdge[multiHopParent, multiHopLeaf]{
		RelName: "LeavesCount",
		Sides: []Side{
			{FromTable: "mh_parents", FromCols: []string{"id"}, ToTable: "mh_mid", ToCols: []string{"parent_id"}},
			{FromTable: "mh_mid", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
		},
		Child:         multiHopLeafParts(),
		KeyCols:       []Column{{Name: "parent_id", DBType: "int4"}},
		NewCarriedBuf: func() Buf[multiHopParent] { return &multiHopParentBuf{} },
		ParentKey:     func(p multiHopParent) []any { return []any{p.ID} },
		ChildKey:      func(c multiHopLeaf) []any { return []any{c.ID} },
		SetCount:      func(p multiHopParent, n int64) error { return nil },
	}
}

// multiHopThreeHopCountEdge: the CountEdge counterpart of multiHopThreeHopEdge
// (multihop_sql_test.go) — mh_parents -> mh_a -> mh_b -> mh_leaves.
func multiHopThreeHopCountEdge() CountEdge[multiHopParent, multiHopLeaf] {
	return CountEdge[multiHopParent, multiHopLeaf]{
		RelName: "ThreeHopLeavesCount",
		Sides: []Side{
			{FromTable: "mh_parents", FromCols: []string{"id"}, ToTable: "mh_a", ToCols: []string{"parent_id"}},
			{FromTable: "mh_a", FromCols: []string{"b_id"}, ToTable: "mh_b", ToCols: []string{"id"}},
			{FromTable: "mh_b", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
		},
		Child:         multiHopLeafParts(),
		KeyCols:       []Column{{Name: "parent_id", DBType: "int4"}},
		NewCarriedBuf: func() Buf[multiHopParent] { return &multiHopParentBuf{} },
		ParentKey:     func(p multiHopParent) []any { return []any{p.ID} },
		ChildKey:      func(c multiHopLeaf) []any { return []any{c.ID} },
		SetCount:      func(p multiHopParent, n int64) error { return nil },
	}
}

// multiHopNestedMultiHopCountEdge: a 2-hop count edge parented on multiHopLeaf
// (NOT the island's true root multiHopParent) — mh_leaves -> mh_leaf_mid ->
// mh_grands. Used to exercise a multi-hop count nested UNDER a
// row section, so its own semi-join targets that row section's own CTE
// (_sl_n<set>), not the island's root (_sl_src).
func multiHopNestedMultiHopCountEdge() CountEdge[multiHopLeaf, multiHopGrand] {
	return CountEdge[multiHopLeaf, multiHopGrand]{
		RelName: "GrandsCount",
		Sides: []Side{
			{FromTable: "mh_leaves", FromCols: []string{"id"}, ToTable: "mh_leaf_mid", ToCols: []string{"leaf_id"}},
			{FromTable: "mh_leaf_mid", FromCols: []string{"grand_id"}, ToTable: "mh_grands", ToCols: []string{"id"}},
		},
		Child:         multiHopGrandParts(),
		KeyCols:       []Column{{Name: "leaf_id", DBType: "int4"}},
		NewCarriedBuf: func() Buf[multiHopLeaf] { return &multiHopLeafBuf{} },
		ParentKey:     func(p multiHopLeaf) []any { return []any{p.ID} },
		ChildKey:      func(c multiHopGrand) []any { return []any{c.ID} },
		SetCount:      func(p multiHopLeaf, n int64) error { return nil },
	}
}

// --- tests ---------------------------------------------------------------

// TestMultiHopCountTwoHopCount_JoinChainAndGroupByAlias: a top-level 2-hop
// count edge's SQL — exactly one INNER JOIN, and (unlike a row edge's
// carried-key SELECT, which aliases the value as "_sl_k0") the SELECT and
// GROUP BY both quote the FIRST intermediate table's own alias/column
// name directly, with no AS rename (a count section's SELECT never adds a
// carried-style SQL column).
func TestMultiHopCountTwoHopCount_JoinChainAndGroupByAlias(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewCountNode(multiHopTwoHopCountEdge()),
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

// TestMultiHopCountThreeHopCount_JoinChainGeneralizes: the 2-hop shape above
// must generalize to 3 hops without new code — 2 JOINs, GROUP BY/SELECT
// still against _sl_h1 (Sides[0].To's alias, still the FIRST intermediate
// table for n==3 — same "stop condition 3" multihop_sql_test.go's 3-hop row
// test already pins for row edges).
func TestMultiHopCountThreeHopCount_JoinChainGeneralizes(t *testing.T) {
	edge := multiHopThreeHopCountEdge()
	sides := edge.sides()
	if len(sides) != 3 {
		t.Fatalf("len(sides()) = %d, want 3", len(sides))
	}

	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
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
		`FROM "mh_leaves" AS "_sl_t"`,
		`INNER JOIN "mh_b" AS "_sl_h2" ON ("_sl_t"."id" = "_sl_h2"."leaf_id")`,
		`INNER JOIN "mh_a" AS "_sl_h1" ON ("_sl_h2"."id" = "_sl_h1"."b_id")`,
		`"_sl_h1"."parent_id"`,
		`GROUP BY "_sl_h1"."parent_id"`,
		`WHERE (("_sl_h1"."parent_id") IN ((SELECT`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
}

// TestMultiHopCountNestedMultiHopCount_SemiJoinsParentSectionCTE: a
// multi-hop count NESTED under a row section (not the island's true
// root) must semi-join against that row section's own CTE (_sl_n<set>),
// not the island root (_sl_src) — parentKeySubquery/matchWhere's
// semiFromCTE is set from countPlan.parentSet, which planIsland resolves
// to whatever node this count is declared under (compose.go's
// semiFromCTE: cteName(c.parentSet), countBranch). GROUP BY/SELECT still
// target the count's own first-intermediate-table alias (_sl_h1), the
// SAME as the top-level case — nesting only changes WHERE this
// count matches against, never how its own JOIN chain/GROUP BY render.
func TestMultiHopCountNestedMultiHopCount_SemiJoinsParentSectionCTE(t *testing.T) {
	built, err := Build(
		context.Background(),
		psql.RawQuery("SELECT id FROM mh_parents"),
		multiHopParentParts(),
		NewNode(multiHopTwoHopEdge(), NewCountNode(multiHopNestedMultiHopCountEdge())),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql := built.SQL

	// The nested count's own JOIN chain/GROUP BY alias, unchanged from the
	// top-level case.
	for _, frag := range []string{
		`FROM "mh_grands" AS "_sl_t"`,
		`INNER JOIN "mh_leaf_mid" AS "_sl_h1" ON ("_sl_t"."id" = "_sl_h1"."grand_id")`,
		`"_sl_h1"."leaf_id"`,
		`GROUP BY "_sl_h1"."leaf_id"`,
	} {
		if !strings.Contains(sql, frag) {
			t.Fatalf("missing fragment %q:\n%s", frag, sql)
		}
	}
	// The semi-join must target the PARENT SECTION's own CTE (_sl_n1, the
	// materialized multiHopTwoHopEdge section this count is nested under), not
	// the island root (_sl_src) — the discriminating assertion for the nested case.
	// The nested count's own subquery (FROM "mh_grands" ... GROUP BY
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
