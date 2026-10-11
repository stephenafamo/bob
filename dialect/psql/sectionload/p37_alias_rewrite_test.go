package sectionload

// P-37(B4): 呼び出し側が dbmodels.<Table>.Columns.<Col> のような完全修飾の列
// 参照(常に "<テーブル名>"."<列名>" の形でレンダリングされる)を
// sectionload.Mods(...) に渡すと、その名前は子セクションの FROM 句に存在し
// ない(FROM には別名 "_sl_t"(多段では "_sl_h<i>")しか無い)ため PostgreSQL
// 側でエラーになる。この付け替え(sectionAliasMap / rewriteQualifiers /
// aliasRewrite / rewriteQualifiedRefs、いずれも P-37 で新設)が正しく効くこ
// とを固定するテスト群。
//
// フィクスチャは既存のもの(p11CapEdge: 単段・子テーブル "p11_children"、
// p05TwoHopEdge: 多段・子テーブル "p05_leaves"、中間テーブル "p05_mid")を
// そのまま再利用する。leafSpec/p11CapEdgeWith/buildSQL は options_test.go
// にある。

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// --- Task 1: sectionAliasMap -----------------------------------------

func TestSectionAliasMapSingleHop(t *testing.T) {
	e := p11CapEdge()
	m := sectionAliasMap(e)

	if len(m) != 1 {
		t.Fatalf("want 1 entry (single-hop has only the child table), got %d: %v", len(m), m)
	}
	if got, want := m["p11_children"], srcAlias; got != want {
		t.Fatalf(`m["p11_children"] = %q, want %q`, got, want)
	}
}

func TestSectionAliasMapMultiHop(t *testing.T) {
	e := p05TwoHopEdge()
	m := sectionAliasMap(e)

	if len(m) != 2 {
		t.Fatalf("want 2 entries (1 intermediate + 1 final), got %d: %v", len(m), m)
	}
	if got, want := m["p05_mid"], hopAlias(1); got != want {
		t.Fatalf(`m["p05_mid"] (intermediate table) = %q, want %q`, got, want)
	}
	if got, want := m["p05_leaves"], srcAlias; got != want {
		t.Fatalf(`m["p05_leaves"] (final/child table) = %q, want %q`, got, want)
	}
}

// p37DuplicateTableEdge: root -(side0)-> dup -(side1)-> dup -(side2)->
// p05_leaves. "dup" is read at TWO different hop aliases (side0's own To
// is hopAlias(1), side1's own To is hopAlias(2)) — the table name alone
// cannot say which occurrence a caller mod's qualified reference means.
func p37DuplicateTableEdge() Edge[p05Parent, p05Leaf] {
	return Edge[p05Parent, p05Leaf]{
		RelName: "Dup",
		Sides: []Side{
			{FromTable: "root", FromCols: []string{"id"}, ToTable: "dup", ToCols: []string{"root_id"}},
			{FromTable: "dup", FromCols: []string{"other_id"}, ToTable: "dup", ToCols: []string{"id"}},
			{FromTable: "dup", FromCols: []string{"leaf_id"}, ToTable: "p05_leaves", ToCols: []string{"id"}},
		},
		ToMany:        true,
		Child:         p05LeafParts(),
		Carried:       []Column{{Name: "_sl_k0", DBType: "int4"}},
		NewCarriedBuf: func() Buf[p05Parent] { return &p05ParentBuf{} },
		ParentKey:     func(p p05Parent) []any { return []any{p.ID} },
		ChildKey:      func(c p05Leaf) []any { return []any{c.ID} },
		Stitch:        func(p p05Parent, cs []p05Leaf) error { return nil },
	}
}

// TestSectionAliasMapDuplicateTableNameIsExcluded (P-37 Important-1,
// gpt-review): a table name recorded at two different aliases must be
// dropped from the map, not silently resolved to whichever occurrence
// sectionAliasMap happened to record last.
func TestSectionAliasMapDuplicateTableNameIsExcluded(t *testing.T) {
	e := p37DuplicateTableEdge()
	m := sectionAliasMap(e)

	if _, ok := m["dup"]; ok {
		t.Fatalf(`"dup" is read at two different aliases (hopAlias(1) and hopAlias(2)); it must be excluded from the map, got %v`, m)
	}
	if len(m) != 1 {
		t.Fatalf("want exactly 1 entry (only the unambiguous child table), got %d: %v", len(m), m)
	}
	if got, want := m["p05_leaves"], srcAlias; got != want {
		t.Fatalf(`m["p05_leaves"] = %q, want %q`, got, want)
	}
}

// --- Task 2: rewriteQualifiers ----------------------------------------

func TestRewriteQualifiers(t *testing.T) {
	// "public" is a dummy schema-name entry (P-37 gpt-review, fable
	// 2026-09-05 判定): sectionAliasMap never actually produces an entry
	// keyed by a schema name — it's added here only so the "schema
	// qualified reference" cases below are a load-bearing negative test.
	// Without the 3rd-segment exclusion in
	// rewriteTwoPartQualifiedIdentifier, "public". would get rewritten to
	// "_sl_h9". because "public" IS in this map, which is exactly the
	// wrong behavior the exclusion exists to prevent.
	m := map[string]string{"users_groups": "_sl_t", "users\"groups": "_sl_t", "public": "_sl_h9"}

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "mapped table qualifies a column",
			in:   `"users_groups"."user_id"`,
			want: `"_sl_t"."user_id"`,
		},
		{
			name: "unmapped table name is left alone",
			in:   `"servers"."id"`,
			want: `"servers"."id"`,
		},
		{
			name: "bare table reference (no trailing column) is not a qualifier and is left alone",
			in:   `"users_groups"`,
			want: `"users_groups"`,
		},
		{
			name: "a string literal matching a mapped name is left alone",
			in:   `'users_groups'`,
			want: `'users_groups'`,
		},
		{
			name: "qualifier outside a literal is rewritten; the same text inside a literal is not",
			in:   `"users_groups"."name" = 'a "users_groups". b'`,
			want: `"_sl_t"."name" = 'a "users_groups". b'`,
		},
		{
			// MEDIUM-1(gpt-review): $$...$$ dollar quoting must not be
			// mistaken for a place to rewrite — the literal here embeds a
			// COMPLETE `"users_groups"."id"` two-part reference (the
			// exact shape rewriteTwoPartQualifiedIdentifier rewrites
			// outside a literal), so this case only stays unchanged if
			// the dollar-quote scan actually skips the whole literal;
			// without it, the reference inside would get rewritten too.
			name: "dollar quoted literal is left alone",
			in:   `note = $$x "users_groups"."id" y$$ AND "users_groups"."user_id" IS NOT NULL`,
			want: `note = $$x "users_groups"."id" y$$ AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// MEDIUM-1(gpt-review): a tagged dollar-quote ($tag$...$tag$)
			// must be matched by its own tag, not by the first bare $$ —
			// same embedded complete two-part reference as above.
			name: "tagged dollar quoted literal is left alone",
			in:   `note = $tag$x "users_groups"."id" y$tag$ AND "users_groups"."user_id" IS NOT NULL`,
			want: `note = $tag$x "users_groups"."id" y$tag$ AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// MEDIUM-1(gpt-review): inside E'...', a backslash escapes
			// the next byte (including a '), so \' does not close the
			// literal early — the qualifier-looking text stays opaque.
			name: "escaped string literal is left alone",
			in:   `note = E'a \' "users_groups". b' AND "users_groups"."user_id" IS NOT NULL`,
			want: `note = E'a \' "users_groups". b' AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// MEDIUM-1(gpt-review): a doubled '' inside a plain
			// single-quoted literal is an escaped quote, not a
			// terminator — the literal keeps running past it.
			name: "single quoted literal with doubled quote is left alone",
			in:   `note = 'a '' "users_groups". b' AND "users_groups"."user_id" IS NOT NULL`,
			want: `note = 'a '' "users_groups". b' AND "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// MEDIUM-2(gpt-review): a doubled "" inside a quoted
			// identifier is an escaped literal '"' byte in the name
			// itself, not the identifier's closing quote.
			name: "quoted identifier with escaped double quote is rewritten by exact identifier name",
			in:   `"users""groups"."user_id"`,
			want: `"_sl_t"."user_id"`,
		},
		{
			// MEDIUM-2(gpt-review): a schema-qualified 3-part reference
			// must be left byte-identical in its entirety. m maps
			// "public" to a dummy alias ("_sl_h9", never produced by the
			// real sectionAliasMap) so this case is load-bearing: without
			// the 3rd-segment exclusion in
			// rewriteTwoPartQualifiedIdentifier, "public". would get
			// rewritten too, producing "_sl_h9"."users_groups"."user_id"
			// (or with the middle segment rewritten instead, depending on
			// scan order) — either way something the FROM clause (bare
			// "_sl_t") does not declare.
			name: "schema qualified reference is left unchanged",
			in:   `"public"."users_groups"."user_id"`,
			want: `"public"."users_groups"."user_id"`,
		},
		{
			// MEDIUM-2(gpt-review): the 3-part and 2-part shapes must be
			// told apart independently within the same statement — same
			// dummy "public" alias entry as above makes this load-bearing
			// too.
			name: "schema qualified reference and ordinary two part reference differ",
			in:   `"public"."users_groups"."user_id" OR "users_groups"."user_id" IS NOT NULL`,
			want: `"public"."users_groups"."user_id" OR "_sl_t"."user_id" IS NOT NULL`,
		},
		{
			// Minor-1(re-review, P-37): a bind placeholder like $1
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
			in:   `"users_groups"."user_id" = $1$ AND "users_groups"."name" = $2`,
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

// --- Task 3: rewriteQualifiedRefs / buildEdgeSelect wiring -------------

// TestBuildEdgeSelectRewritesQualifiedOrderBy: an ORDER BY opt that
// qualifies its column with the child table's own name (exactly what
// every dbmodels.<Table>.Columns.<Col> expression renders) must resolve
// against the section's own alias, not the bare table name.
func TestBuildEdgeSelectRewritesQualifiedOrderBy(t *testing.T) {
	e := p11CapEdgeWith(Mods(sm.OrderBy(psql.Quote("p11_children", "id")).Desc()))
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
	if strings.Contains(sql, `"p11_children"."id"`) {
		t.Fatalf("qualified reference to the child table name must not survive rewriting; got:\n%s", sql)
	}
}

// TestBuildEdgeSelectRewritesQualifiedWhere: same as above, for WHERE.
func TestBuildEdgeSelectRewritesQualifiedWhere(t *testing.T) {
	e := p11CapEdgeWith(Mods(sm.Where(psql.Quote("p11_children", "parent_id").IsNull())))
	sql, _ := buildSQL(t, leafSpec(e))

	if !strings.Contains(sql, `"_sl_t"."parent_id" IS NULL`) {
		t.Fatalf("expected WHERE rewritten to the section alias %q; got:\n%s", srcAlias, sql)
	}
	if strings.Contains(sql, `"p11_children"."parent_id"`) {
		t.Fatalf("qualified reference to the child table name must not survive rewriting; got:\n%s", sql)
	}
}

// TestAliasRewritePreservesArgs: the rewrite happens on the SQL text a
// wrapped expression's own WriteSQL produced — bind parameters (args) must
// pass through unchanged, in count and order.
func TestAliasRewritePreservesArgs(t *testing.T) {
	e := p11CapEdgeWith(Mods(
		sm.Where(psql.Quote("p11_children", "id").EQ(psql.Arg("user"))),
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
// This is the "safe to apply unconditionally" guarantee §5 item 4 of the
// issue relies on.
func TestAliasRewriteIsTransparentWithoutMappedNames(t *testing.T) {
	m := map[string]string{"users_groups": "_sl_t"}
	inner := psql.Quote("servers", "id")

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
	e := p11CapEdgeWith(Mods(sm.Where(psql.Quote("servers", "id").IsNull())))
	sql, _ := buildSQL(t, leafSpec(e))

	if !strings.Contains(sql, `"servers"."id" IS NULL`) {
		t.Fatalf(`expected unmapped table reference "servers"."id" to survive unchanged; got:\n%s`, sql)
	}
}

// --- P-37 Critical (gpt-review): unionBranch's own ORDER BY extraction --

// TestUnionBranchRewritesQualifiedOrderByInOrdinal pins the Critical fix:
// unionBranch (compose.go) extracts its own permittedFragment BEFORE
// buildEdgeSelect ever runs — unionBranchOrdColumn reads
// fragment.orderBy.Expressions straight out of it to build _sl_ord's
// row_number() OVER (ORDER BY ...) ordinal column. Before this fix, a
// caller mod's qualified ORDER BY column (e.g.
// dbmodels.<Table>.Columns.<Col>, rendering "<table>"."<col>") reached
// _sl_ord unrewritten, because buildEdgeSelect's OWN rewriteQualifiedRefs
// call (options.go/compose.go) only ever sees the SELECT's top-level
// WHERE/ORDER BY/GROUP BY/HAVING — not the ordinal column
// unionBranchProjection already baked into the projection list before
// buildEdgeSelect is even called. This test drives the real
// planIsland/unionBranch path (not a hand-built edgeSelectSpec), a
// hasKids=false leaf node with an ORDER BY that qualifies its column with
// the (multi-hop) edge's own final/child table name — the exact
// dbmodels.<Table>.Columns.<Col> shape a caller mod renders.
func TestUnionBranchRewritesQualifiedOrderByInOrdinal(t *testing.T) {
	e := p05TwoHopEdge()
	e.Options = Mods(sm.OrderBy(psql.Quote("p05_leaves", "id")).Desc())

	plan, err := planIsland([]Node{NewNode(e)})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if plan.sections[0].hasKids {
		t.Fatalf("fixture must produce a leaf (hasKids=false) node; got hasKids=true")
	}

	root := partsAdapter[p05Parent]{p05ParentParts()}
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
	if strings.Contains(sql, `"p05_leaves"."id"`) {
		t.Fatalf("qualified reference to the child table name must not survive inside _sl_ord; got:\n%s", sql)
	}
}
