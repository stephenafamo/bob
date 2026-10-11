package sectionload

// A fully qualified column reference such as a generated
// <Table>.Columns.<Col> expression (always rendered as "<table>"."<column>")
// passed to sectionload.Mods(...) names a table that does not exist in the
// child section's FROM clause: FROM only has the alias "_sl_t" ("_sl_h<i>" for
// multi-hop edges), so PostgreSQL would raise an error. These tests pin that
// the rewrite (sectionAliasMap / rewriteQualifiers / aliasRewrite /
// rewriteQualifiedRefs) resolves such references to the section aliases.
//
// The fixtures are existing ones: builderCapEdge (single hop, child table
// "bld_children") and multiHopTwoHopEdge (multi-hop, child table "mh_leaves",
// intermediate table "mh_mid"). leafSpec/builderCapEdgeWith/buildSQL live in
// options_test.go.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// --- sectionAliasMap -----------------------------------------

func TestSectionAliasMapSingleHop(t *testing.T) {
	e := builderCapEdge()
	m := sectionAliasMap(e)

	if len(m) != 1 {
		t.Fatalf("want 1 entry (single-hop has only the child table), got %d: %v", len(m), m)
	}
	if got, want := m["bld_children"], srcAlias; got != want {
		t.Fatalf(`m["bld_children"] = %q, want %q`, got, want)
	}
}

func TestSectionAliasMapMultiHop(t *testing.T) {
	e := multiHopTwoHopEdge()
	m := sectionAliasMap(e)

	if len(m) != 2 {
		t.Fatalf("want 2 entries (1 intermediate + 1 final), got %d: %v", len(m), m)
	}
	if got, want := m["mh_mid"], hopAlias(1); got != want {
		t.Fatalf(`m["mh_mid"] (intermediate table) = %q, want %q`, got, want)
	}
	if got, want := m["mh_leaves"], srcAlias; got != want {
		t.Fatalf(`m["mh_leaves"] (final/child table) = %q, want %q`, got, want)
	}
}

// aliasRewriteDuplicateTableEdge: root -(side0)-> dup -(side1)-> dup -(side2)->
// mh_leaves. "dup" is read at TWO different hop aliases (side0's own To
// is hopAlias(1), side1's own To is hopAlias(2)) — the table name alone
// cannot say which occurrence a caller mod's qualified reference means.
func aliasRewriteDuplicateTableEdge() Edge[multiHopParent, multiHopLeaf] {
	return Edge[multiHopParent, multiHopLeaf]{
		RelName: "Dup",
		Sides: []Side{
			{FromTable: "root", FromCols: []string{"id"}, ToTable: "dup", ToCols: []string{"root_id"}},
			{FromTable: "dup", FromCols: []string{"other_id"}, ToTable: "dup", ToCols: []string{"id"}},
			{FromTable: "dup", FromCols: []string{"leaf_id"}, ToTable: "mh_leaves", ToCols: []string{"id"}},
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

// TestSectionAliasMapDuplicateTableNameIsExcluded: a table name recorded at two different aliases must be
// dropped from the map, not silently resolved to whichever occurrence
// sectionAliasMap happened to record last.
func TestSectionAliasMapDuplicateTableNameIsExcluded(t *testing.T) {
	e := aliasRewriteDuplicateTableEdge()
	m := sectionAliasMap(e)

	if _, ok := m["dup"]; ok {
		t.Fatalf(`"dup" is read at two different aliases (hopAlias(1) and hopAlias(2)); it must be excluded from the map, got %v`, m)
	}
	if len(m) != 1 {
		t.Fatalf("want exactly 1 entry (only the unambiguous child table), got %d: %v", len(m), m)
	}
	if got, want := m["mh_leaves"], srcAlias; got != want {
		t.Fatalf(`m["mh_leaves"] = %q, want %q`, got, want)
	}
}

// --- rewriteQualifiers ----------------------------------------

func TestRewriteQualifiers(t *testing.T) {
	// "public" is a dummy schema-name entry: sectionAliasMap never actually produces an entry
	// keyed by a schema name — it's added here only so the "schema
	// qualified reference" cases below are a load-bearing negative test.
	// Without the 3rd-segment exclusion in
	// rewriteTwoPartQualifiedIdentifier, "public". would get rewritten to
	// "_sl_h9". because "public" IS in this map, which is exactly the
	// wrong behavior the exclusion exists to prevent.
	m := map[string]string{"user_teams": "_sl_t", "user\"teams": "_sl_t", "public": "_sl_h9"}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mapped table qualifies a column",
			in:   `"user_teams"."user_id"`,
			want: `"_sl_t"."user_id"`,
		},
		{
			name: "unmapped table name is left alone",
			in:   `"authors"."id"`,
			want: `"authors"."id"`,
		},
		{
			name: "bare table reference (no trailing column) is not a qualifier and is left alone",
			in:   `"user_teams"`,
			want: `"user_teams"`,
		},
		{
			name: "a string literal matching a mapped name is left alone",
			in:   `'user_teams'`,
			want: `'user_teams'`,
		},
		{
			name: "qualifier outside a literal is rewritten; the same text inside a literal is not",
			in:   `"user_teams"."name" = 'a "user_teams". b'`,
			want: `"_sl_t"."name" = 'a "user_teams". b'`,
		},
		{
			// $$...$$ dollar quoting must not be
			// mistaken for a place to rewrite — the literal here embeds a
			// COMPLETE `"user_teams"."id"` two-part reference (the
			// exact shape rewriteTwoPartQualifiedIdentifier rewrites
			// outside a literal), so this case only stays unchanged if
			// the dollar-quote scan actually skips the whole literal;
			// without it, the reference inside would get rewritten too.
			name: "dollar quoted literal is left alone",
			in:   `note = $$x "user_teams"."id" y$$ AND "user_teams"."user_id" IS NOT NULL`,
			want: `note = $$x "user_teams"."id" y$$ AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// a tagged dollar-quote ($tag$...$tag$)
			// must be matched by its own tag, not by the first bare $$ —
			// same embedded complete two-part reference as above.
			name: "tagged dollar quoted literal is left alone",
			in:   `note = $tag$x "user_teams"."id" y$tag$ AND "user_teams"."user_id" IS NOT NULL`,
			want: `note = $tag$x "user_teams"."id" y$tag$ AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// inside E'...', a backslash escapes
			// the next byte (including a '), so \' does not close the
			// literal early — the qualifier-looking text stays opaque.
			name: "escaped string literal is left alone",
			in:   `note = E'a \' "user_teams". b' AND "user_teams"."user_id" IS NOT NULL`,
			want: `note = E'a \' "user_teams". b' AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// a doubled '' inside a plain
			// single-quoted literal is an escaped quote, not a
			// terminator — the literal keeps running past it.
			name: "single quoted literal with doubled quote is left alone",
			in:   `note = 'a '' "user_teams". b' AND "user_teams"."user_id" IS NOT NULL`,
			want: `note = 'a '' "user_teams". b' AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// a doubled "" inside a quoted
			// identifier is an escaped literal '"' byte in the name
			// itself, not the identifier's closing quote.
			name: "quoted identifier with escaped double quote is rewritten by exact identifier name",
			in:   `"user""teams"."user_id"`,
			want: `"_sl_t"."user_id"`,
		},
		{
			// a schema-qualified 3-part reference
			// must be left byte-identical in its entirety. m maps
			// "public" to a dummy alias ("_sl_h9", never produced by the
			// real sectionAliasMap) so this case is load-bearing: without
			// the 3rd-segment exclusion in
			// rewriteTwoPartQualifiedIdentifier, "public". would get
			// rewritten too, producing "_sl_h9"."user_teams"."user_id"
			// (or with the middle segment rewritten instead, depending on
			// scan order) — either way something the FROM clause (bare
			// "_sl_t") does not declare.
			name: "schema qualified reference is left unchanged",
			in:   `"public"."user_teams"."user_id"`,
			want: `"public"."user_teams"."user_id"`,
		},
		{
			// the 3-part and 2-part shapes must be
			// told apart independently within the same statement — same
			// dummy "public" alias entry as above makes this load-bearing
			// too.
			name: "schema qualified reference and ordinary two part reference differ",
			in:   `"public"."user_teams"."user_id" OR "user_teams"."user_id" IS NOT NULL`,
			want: `"public"."user_teams"."user_id" OR "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// A bind placeholder like $1
			// immediately followed by another '$' (e.g. "$1$" from a
			// second placeholder starting right after) must not be
			// mistaken for the start of a tagged dollar-quote — a
			// dollar-quote tag's first byte can never be a digit, so `1`
			// right after the leading '$' rules out the dollar-quote
			// interpretation and both '$' bytes are ordinary characters.
			// This case is load-bearing: without isDollarTagStartByte
			// rejecting a leading digit, scanDollarQuotedLiteral would
			// treat "$1$" as opening a `1`-tagged dollar-quote, skip past
			// the second mapped-name reference below without rewriting
			// it, and only stop at the AND'ed IS NOT NULL by
			// running off the end of the string unterminated.
			name: "bind placeholder followed by another dollar sign is not a dollar quote tag",
			in:   `"user_teams"."user_id" = $1$ AND "user_teams"."name" = $2`,
			want: `"_sl_t"."user_id" = $1$ AND "_sl_t"."name" = $2`,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := rewriteQualifiers(c.in, m)
			if got != c.want {
				t.Fatalf("rewriteQualifiers(%q, %v) = %q, want %q", c.in, m, got, c.want)
			}
		})
	}
}

// --- rewriteQualifiedRefs / buildEdgeSelect wiring -------------

// TestBuildEdgeSelectRewritesQualifiedOrderBy: an ORDER BY opt that
// qualifies its column with the child table's own name (exactly what
// every generated <Table>.Columns.<Col> expression renders) must resolve
// against the section's own alias, not the bare table name.
func TestBuildEdgeSelectRewritesQualifiedOrderBy(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.OrderBy(psql.Quote("bld_children", "id")).Desc()))
	child := e.child()
	spec := edgeSelectSpec{
		edge:  e,
		cols:  child.cols(),
		alias: srcAlias,
		from:  tableExpr(child.schemaName(), child.tableName()),
		keys:  [][]any{{1}, {2}},
		opts:  e.options(),
		// collapseOrderBy left false so the ORDER BY stays literal text —
		// the collapsed (row_number() OVER (...)) form is covered
		// separately by TestBuildEdgeSelect_OptionsOrderBy_Collapsed
		// (options_test.go), which this change does not touch.
	}
	sql, _ := buildSQL(t, spec)

	if !strings.Contains(sql, `ORDER BY "_sl_t"."id"`) {
		t.Fatalf("expected ORDER BY rewritten to the section alias %q; got:\n%s", srcAlias, sql)
	}
	if strings.Contains(sql, `"bld_children"."id"`) {
		t.Fatalf("qualified reference to the child table name must not survive rewriting; got:\n%s", sql)
	}
}

// TestBuildEdgeSelectRewritesQualifiedWhere: same as above, for WHERE.
func TestBuildEdgeSelectRewritesQualifiedWhere(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.Where(psql.Quote("bld_children", "parent_id").IsNull())))
	sql, _ := buildSQL(t, leafSpec(e))

	if !strings.Contains(sql, `"_sl_t"."parent_id" IS NULL`) {
		t.Fatalf("expected WHERE rewritten to the section alias %q; got:\n%s", srcAlias, sql)
	}
	if strings.Contains(sql, `"bld_children"."parent_id"`) {
		t.Fatalf("qualified reference to the child table name must not survive rewriting; got:\n%s", sql)
	}
}

// TestAliasRewritePreservesArgs: the rewrite happens on the SQL text a
// wrapped expression's own WriteSQL produced — bind parameters (args) must
// pass through unchanged, in count and order.
func TestAliasRewritePreservesArgs(t *testing.T) {
	e := builderCapEdgeWith(Mods(
		sm.Where(psql.Quote("bld_children", "id").EQ(psql.Arg("user"))),
	))
	sql, args := buildSQL(t, leafSpec(e))

	if len(args) != 1 || args[0] != "user" {
		t.Fatalf("args: got %v, want [\"user\"]", args)
	}
	if strings.Count(sql, "$1") != 1 && strings.Count(sql, "?") != 1 {
		t.Fatalf("expected exactly one bind placeholder; got:\n%s", sql)
	}
}

// TestAliasRewriteIsTransparentWithoutMappedNames: when a wrapped
// expression never renders a name present in the alias map, wrapping it
// must not change the SQL text at all — not even by adding parentheses.
// This is the "safe to apply unconditionally" guarantee the wrapping relies
// on.
func TestAliasRewriteIsTransparentWithoutMappedNames(t *testing.T) {
	m := map[string]string{"user_teams": "_sl_t"}
	inner := psql.Quote("authors", "id")

	var beforeBuf strings.Builder
	if _, err := inner.WriteSQL(context.Background(), &beforeBuf, pdialect.Dialect, 1); err != nil {
		t.Fatalf("inner.WriteSQL: %v", err)
	}
	before := beforeBuf.String()

	wrapped := aliasRewrite{inner: inner, m: m}
	var afterBuf strings.Builder
	if _, err := wrapped.WriteSQL(context.Background(), &afterBuf, pdialect.Dialect, 1); err != nil {
		t.Fatalf("wrapped.WriteSQL: %v", err)
	}
	after := afterBuf.String()

	if before != after {
		t.Fatalf("wrapping changed the SQL text with no mapped name present:\nbefore: %q\nafter:  %q", before, after)
	}
	if strings.Contains(after, "(") || strings.Contains(after, ")") {
		t.Fatalf("wrapping must not add parentheses; got: %q", after)
	}
}

// TestBuildEdgeSelectLeavesUnmappedNamesAlone: a qualified reference to a
// table this section does NOT read under an alias (not in the alias map)
// must be left exactly as the caller wrote it.
func TestBuildEdgeSelectLeavesUnmappedNamesAlone(t *testing.T) {
	e := builderCapEdgeWith(Mods(sm.Where(psql.Quote("authors", "id").IsNull())))
	sql, _ := buildSQL(t, leafSpec(e))

	if !strings.Contains(sql, `"authors"."id" IS NULL`) {
		t.Fatalf(`expected unmapped table reference "authors"."id" to survive unchanged; got:\n%s`, sql)
	}
}

// --- unionBranch's own ORDER BY extraction --

// TestUnionBranchRewritesQualifiedOrderByInOrdinal: unionBranch (compose.go) extracts its own permittedFragment BEFORE
// buildEdgeSelect ever runs — unionBranchOrdColumn reads
// fragment.orderBy.Expressions straight out of it to build _sl_ord's
// row_number() OVER (ORDER BY ...) ordinal column. A caller mod's qualified
// ORDER BY column (e.g. a generated <Table>.Columns.<Col>, rendering
// "<table>"."<col>") would otherwise reach _sl_ord unrewritten, because buildEdgeSelect's OWN rewriteQualifiedRefs
// call (options.go/compose.go) only ever sees the SELECT's top-level
// WHERE/ORDER BY/GROUP BY/HAVING — not the ordinal column
// unionBranchProjection already baked into the projection list before
// buildEdgeSelect is even called. This test drives the real
// planIsland/unionBranch path (not a hand-built edgeSelectSpec), a
// hasKids=false leaf node with an ORDER BY that qualifies its column with
// the (multi-hop) edge's own final/child table name — the exact
// <Table>.Columns.<Col> shape a caller mod renders.
func TestUnionBranchRewritesQualifiedOrderByInOrdinal(t *testing.T) {
	e := multiHopTwoHopEdge()
	e.Options = Mods(sm.OrderBy(psql.Quote("mh_leaves", "id")).Desc())

	plan, err := planIsland([]Node{NewNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if plan.sections[0].hasKids {
		t.Fatalf("fixture must produce a leaf (hasKids=false) node; got hasKids=true")
	}

	root := partsAdapter[multiHopParent]{multiHopParentParts()}
	branch, err := unionBranch(root, plan, plan.sections[0], false)
	if err != nil {
		t.Fatalf("unionBranch: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), branch)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}

	if !strings.Contains(sql, "row_number()") || !strings.Contains(sql, "OVER") {
		t.Fatalf("expected a row_number() OVER (...) ordinal column; got:\n%s", sql)
	}
	if !strings.Contains(sql, `ORDER BY "_sl_t"."id"`) {
		t.Fatalf("expected the ordinal's ORDER BY rewritten to the section alias; got:\n%s", sql)
	}
	if strings.Contains(sql, `"mh_leaves"."id"`) {
		t.Fatalf("qualified reference to the child table name must not survive inside _sl_ord; got:\n%s", sql)
	}
}
