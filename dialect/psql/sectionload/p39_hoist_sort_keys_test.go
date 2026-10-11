package sectionload

// P-39: outerParentOrd(run.go)は、親クエリの ORDER BY が親テーブル自身の
// 裸の列参照でない要素(JOIN 先の列・EXISTS 式)を含んでいても、その要素だけ
// を親クエリの SELECT リスト末尾に "_sl_row_sort<i>"(sortColPrefix)として
// 持ち上げ、外側 window の ORDER BY をその別名参照に差し替えることで、外側
// 方式(row_number() を親サブクエリの外側に置き、LIMIT の top-N heapsort を
// 保つ配置)を使えるようにする。親自身の裸の列参照はそのまま(混在方式、
// issue P-39 §6.1)。
//
// これらは p35_outer_ord_test.go と同じスタイルの Build()-only(DB 不要)の
// SQL テキストアサーションで、bob.Build した合成 SQL の中の断片を確認する。

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// p39ExistsExpr builds a correlated EXISTS(...) expression with exactly one
// bind parameter (the "= 1" comparison), the shape 4.1/5.5 in issue P-39
// describe as the reason sortIsExternalScan (GetSupplyChainRisks) fails the
// pre-P-39 len(args) != 0 rejection.
func p39ExistsExpr() bob.Expression {
	return psql.Exists(psql.Select(
		sm.Columns(psql.S("1")),
		sm.From(psql.Quote("other_table")),
		sm.Where(psql.Quote("other_table", "root_id").EQ(psql.Arg(1))),
	))
}

func TestP39Hoist_JoinedColumnUsesOuterWindow(t *testing.T) {
	parent := p35ParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")))

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `"other_table"."name" AS "_sl_row_sort0"`) {
		t.Fatalf("expected the joined column hoisted onto the parent's own SELECT list:\n%s", built.SQL)
	}
	if !strings.Contains(built.SQL, `row_number()OVER ( ORDER BY "p28_root"."_sl_row_sort0") AS "_sl_ord"`) {
		t.Fatalf("expected _sl_src's row_number() to ORDER BY the hoisted alias:\n%s", built.SQL)
	}
	// The parent subquery itself (the "FROM (SELECT ... ) AS "p28_root""
	// span, not _sl_src's own wrapping SELECT which carries the outer
	// row_number()) must carry no row_number() of its own — that would put
	// a WindowAgg back inside the LIMIT-ed subquery and defeat the top-N
	// sort this whole mechanism exists to keep (issue P-39 §2.3).
	parentSubqueryStart := strings.Index(built.SQL, "FROM (SELECT")
	parentSubqueryEnd := strings.Index(built.SQL, ") AS \"p28_root\"")
	if parentSubqueryStart < 0 || parentSubqueryEnd < 0 || parentSubqueryEnd < parentSubqueryStart {
		t.Fatalf(`missing parent subquery span ("FROM (SELECT" .. ") AS \"p28_root\""):`+"\n%s", built.SQL)
	}
	if strings.Count(built.SQL[parentSubqueryStart:parentSubqueryEnd], "row_number()") != 0 {
		t.Fatalf("parent subquery must not carry its own row_number():\n%s", built.SQL)
	}
}

func TestP39Hoist_ExistsExprUsesOuterWindow(t *testing.T) {
	parent := p35ParentQuery(
		10,
		sm.OrderBy(p39ExistsExpr()).Desc(),
		sm.OrderBy(psql.Quote("p28_root", "id")),
	)

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `AS "_sl_row_sort0"`) {
		t.Fatalf("expected the EXISTS expression hoisted onto the parent's own SELECT list:\n%s", built.SQL)
	}
	wantWindow := `ORDER BY "p28_root"."_sl_row_sort0" DESC, "p28_root"."id") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected the outer window to ORDER BY the hoisted alias then the bare parent column, missing %q:\n%s", wantWindow, built.SQL)
	}
	// The EXISTS expression is rendered twice (parent's own ORDER BY, plus
	// the hoisted SELECT column) — issue P-39 §5.5 accepts this as a known,
	// unmeasured tradeoff rather than rewriting the parent's own ORDER BY.
	// Each rendering carries its own "= $N" bind parameter, so two EXISTS
	// renderings means two bind parameters in the built query.
	if got := strings.Count(built.SQL, "EXISTS"); got != 2 {
		t.Fatalf("expected the EXISTS expression to appear twice (parent ORDER BY + hoisted SELECT column), got %d occurrences:\n%s", got, built.SQL)
	}
	if got := len(built.Args); got != 2 {
		t.Fatalf("expected 2 bind arguments (one per EXISTS rendering), got %d: %#v\n%s", got, built.Args, built.SQL)
	}
}

func TestP39Hoist_MixedKeysHoistOnlyForeign(t *testing.T) {
	parent := p35ParentQuery(
		10,
		sm.OrderBy(psql.Quote("other_table", "name")).Desc(),
		sm.OrderBy(psql.Quote("p28_root", "id")),
	)

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `"other_table"."name" AS "_sl_row_sort0"`) {
		t.Fatalf("expected the joined column hoisted as _sl_row_sort0:\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, "_sl_row_sort1") {
		t.Fatalf("the parent's own bare column must not be hoisted a second time as _sl_row_sort1:\n%s", built.SQL)
	}
	wantWindow := `ORDER BY "p28_root"."_sl_row_sort0" DESC, "p28_root"."id") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected the outer window's 2nd element to stay the bare parent column reference, missing %q:\n%s", wantWindow, built.SQL)
	}
}

func TestP39Hoist_DirectionAndNullsPreserved(t *testing.T) {
	parent := p35ParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")).Desc().NullsLast())

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	wantWindow := `row_number()OVER ( ORDER BY "p28_root"."_sl_row_sort0" DESC NULLS LAST) AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected Direction/Nulls carried over from the original clause.OrderDef onto the hoisted alias, missing %q:\n%s", wantWindow, built.SQL)
	}
}

func TestP39Hoist_NoLimitFallsBackToInside(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("p28_root", "id")),
		sm.From(psql.Quote("p28_root")),
		sm.OrderBy(psql.Quote("other_table", "name")),
	)

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_sort0") {
		t.Fatalf("without a LIMIT, outerParentOrd must decline (no hoisting) and fall back to installParentOrdColumn:\n%s", built.SQL)
	}
	wantInner := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantInner) {
		t.Fatalf("expected the inside installParentOrdColumn fallback, missing %q:\n%s", wantInner, built.SQL)
	}
}

func TestP39Hoist_DistinctFallsBackToInside(t *testing.T) {
	parent := psql.Select(
		sm.Distinct(),
		sm.Columns(psql.Quote("p28_root", "id")),
		sm.From(psql.Quote("p28_root")),
		sm.OrderBy(psql.Quote("other_table", "name")),
		sm.Limit(10),
	)

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_sort0") {
		t.Fatalf("DISTINCT must not trigger hoisting either (same opt-out outerParentOrd already had before P-39):\n%s", built.SQL)
	}
	if !strings.Contains(built.SQL, `row_number() OVER () AS "_sl_ord"`) {
		t.Fatalf("DISTINCT must keep the no-ORDER-BY-guarantee fallback row_number() OVER ():\n%s", built.SQL)
	}
}

func TestP39Hoist_RestoreTrimsHoistedColumn(t *testing.T) {
	parent := p35ParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")))

	built1, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build (1st): %v", err)
	}
	built2, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build (2nd): %v", err)
	}

	if built1.SQL != built2.SQL {
		t.Fatalf("a second Build on the same parent query value must produce byte-identical SQL (hoistParentSortKeys's restore func must trim the mod back off):\nfirst:\n%s\nsecond:\n%s", built1.SQL, built2.SQL)
	}
	if strings.Count(built2.SQL, "_sl_row_sort0") != strings.Count(built1.SQL, "_sl_row_sort0") {
		t.Fatalf("the hoisted column must not be added a second time:\n%s", built2.SQL)
	}
}

func TestP39Hoist_BareParentColumnsUnchanged(t *testing.T) {
	parent := p35ParentQuery(
		10,
		sm.OrderBy(psql.Quote("p28_root", "id")).Desc(),
		sm.OrderBy(psql.Quote("p28_root", "ssvc_priority")),
	)

	built, err := Build(context.Background(), parent, p35RootParts(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_sort") {
		t.Fatalf("an ORDER BY made entirely of bare parent column references must not hoist anything (byte-identical to pre-P-39, issue P-39 §5.3):\n%s", built.SQL)
	}
	wantWindow := `"p28_root".*, row_number()OVER ( ORDER BY "p28_root"."id" DESC, "p28_root"."ssvc_priority") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantWindow) {
		t.Fatalf("expected the same outer-window SQL p35_outer_ord_test.go's TestP35OuterOrd_LimitUsesOuterWindow produces, missing %q:\n%s", wantWindow, built.SQL)
	}
}

// p39RootPartsWithReservedSortName is p35RootParts's variant that also
// declares a real parent column literally named "_sl_row_sort0" — the
// exact name outerParentOrd would assign the first hoisted sort key
// (sortColPrefix + "0"). It exercises the gpt-review LOW fix (P-39):
// outerParentOrd must decline the whole outer placement rather than
// collide with (or silently resolve to) this column.
func p39RootPartsWithReservedSortName() TableParts[p28Root] {
	return TableParts[p28Root]{
		Table: "p28_root",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "ssvc_priority", DBType: "text"},
			{Name: "_sl_row_sort0", DBType: "text"},
		},
		NewBuf: func() Buf[p28Root] { return &p28RootBuf{} },
	}
}

func TestP39Hoist_ReservedNameCollisionFallsBackToInside(t *testing.T) {
	// A joined-table column ORDER BY key would normally be hoisted onto
	// the parent's own SELECT list as "_sl_row_sort0" (see
	// TestP39Hoist_JoinedColumnUsesOuterWindow above) — but the parent
	// table here already has a real column of that exact name (see
	// p39RootPartsWithReservedSortName), so outerParentOrd must decline
	// the outer placement entirely and fall back to
	// installParentOrdColumn's inside placement instead.
	parent := p35ParentQuery(10, sm.OrderBy(psql.Quote("other_table", "name")))

	built, err := Build(context.Background(), parent, p39RootPartsWithReservedSortName(), NewNode(p28RootMidEdge()))
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	// "_sl_row_sort0" itself legitimately appears in the built SQL here —
	// p39RootPartsWithReservedSortName declares it as a real parent
	// column, so it is selected (and NULL-padded on the sibling UNION ALL
	// branch, "(CAST(NULL AS text)) AS \"_sl_row_sort0\"") like any other
	// column of p28_root. What must NOT appear is hoistParentSortKeys's
	// own rendering of the hoisted ORDER BY expression under that exact
	// alias — the same "<joined column> AS \"_sl_row_sort0\"" shape
	// TestP39Hoist_JoinedColumnUsesOuterWindow above asserts DOES appear
	// when there is no name collision.
	if strings.Contains(built.SQL, `"other_table"."name" AS "_sl_row_sort0"`) {
		t.Fatalf("outerParentOrd must not hoist \"other_table\".\"name\" under an alias that collides with the parent's own \"_sl_row_sort0\" column:\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, `ORDER BY "p28_root"."_sl_row_sort0"`) || strings.Contains(built.SQL, `ORDER BY "q"."_sl_row_sort0"`) {
		t.Fatalf("the outer window must not ORDER BY the reserved-name alias once outerParentOrd declines the outer placement:\n%s", built.SQL)
	}
	wantInner := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantInner) {
		t.Fatalf("expected the inside installParentOrdColumn fallback, missing %q:\n%s", wantInner, built.SQL)
	}
}
