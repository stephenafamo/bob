package sectionload

// P-35 (stg実測): 内側方式(installParentOrdColumn)は、親クエリに LIMIT が
// あり ORDER BY 列に索引が無いとき、row_number() ウィンドウが
// Limit と Sort の間に挟まる(WindowAgg -> Sort -> Limit)ため、PostgreSQL
// が top-N heapsort を使えず全件ソートになる。ORDER BY が親テーブル自身の
// 列だけを参照する LIMIT 付き親クエリでは、row_number() を親サブクエリの
// 外側(outerParentOrd/run.go)に出すことで、LIMIT が先に効いた後の行だけを
// window の対象にできる。
//
// これらは p34_parent_orderby_test.go と同じスタイルの Build()-only(DB
// 不要)の SQL テキストアサーションで、bob.Build した合成 SQL の中の断片を
// 確認する。

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// p35RootParts is p05BaselineRootParts's 2-column variant (id,
// ssvc_priority): outerParentOrd requires every ORDER BY expression to
// name a column present in TableParts.Columns, so the multi-key case
// below needs a second column beyond p05BaselineRootParts's single "id".
// NewBuf is never invoked by these Build()-only tests, so reusing
// p28RootBuf (which only has an id field) is safe.
func p35RootParts() TableParts[p28Root] {
	return TableParts[p28Root]{
		Table: "p28_root",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "ssvc_priority", DBType: "text"},
		},
		NewBuf: func() Buf[p28Root] { return &p28RootBuf{} },
	}
}

// p35ParentQuery builds a LIMIT-ed parent query over p28_root, ORDERed BY
// the given mods (each expected to reference only p28_root's own columns
// via psql.Quote("p28_root", ...), the shape outerParentOrd requires).
func p35ParentQuery(limit int, orderMods ...bob.Mod[*dialect.SelectQuery]) bob.Query {
	mods := append([]bob.Mod[*dialect.SelectQuery]{
		sm.Columns(psql.Quote("p28_root", "id")),
		sm.From(psql.Quote("p28_root")),
	}, orderMods...)
	mods = append(mods, sm.Limit(limit))
	return psql.Select(mods...)
}

func TestP35OuterOrd_LimitUsesOuterWindow(t *testing.T) {
	tests := []struct {
		name           string
		orderMods      []bob.Mod[*dialect.SelectQuery]
		wantOuterOrder string // the ORDER BY clause inside the outer row_number() window and the parent's own trailing ORDER BY
	}{
		{
			name: "single_key_desc",
			orderMods: []bob.Mod[*dialect.SelectQuery]{
				sm.OrderBy(psql.Quote("p28_root", "id")).Desc(),
			},
			wantOuterOrder: `"p28_root"."id" DESC`,
		},
		{
			name: "single_key_desc_nulls_last",
			orderMods: []bob.Mod[*dialect.SelectQuery]{
				sm.OrderBy(psql.Quote("p28_root", "id")).Desc().NullsLast(),
			},
			wantOuterOrder: `"p28_root"."id" DESC NULLS LAST`,
		},
		{
			name: "two_keys",
			orderMods: []bob.Mod[*dialect.SelectQuery]{
				sm.OrderBy(psql.Quote("p28_root", "ssvc_priority")).Desc().NullsLast(),
				sm.OrderBy(psql.Quote("p28_root", "id")).Asc(),
			},
			wantOuterOrder: `"p28_root"."ssvc_priority" DESC NULLS LAST, "p28_root"."id" ASC`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			built, err := Build(
				context.Background(),
				p35ParentQuery(20, tt.orderMods...),
				p35RootParts(),
				NewNode(p28RootMidEdge()),
			)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			t.Logf("SQL:\n%s", built.SQL)

			wantWindow := `"p28_root".*, row_number()OVER ( ORDER BY ` + tt.wantOuterOrder + `) AS "_sl_ord"`
			if !strings.Contains(built.SQL, wantWindow) {
				t.Fatalf("missing outer row_number window %q:\n%s", wantWindow, built.SQL)
			}
			if !strings.Contains(built.SQL, `) AS "p28_root"`) {
				t.Fatalf(`missing outer wrapper alias ") AS \"p28_root\"":`+"\n%s", built.SQL)
			}
			if strings.Contains(built.SQL, "_sl_row_ord") {
				t.Fatalf("must not fall back to the inside installParentOrdColumn column when the outer window applies:\n%s", built.SQL)
			}
			wantParentOrderByAndLimit := "ORDER BY " + tt.wantOuterOrder + "\nLIMIT 20"
			if strings.Count(built.SQL, wantParentOrderByAndLimit) != 1 {
				t.Fatalf("parent's own ORDER BY + LIMIT must survive unchanged, exactly once:\n%s", built.SQL)
			}
		})
	}
}

func TestP35OuterOrd_NoLimitStaysInner(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("p28_root", "id")),
		sm.From(psql.Quote("p28_root")),
		sm.OrderBy(psql.Quote("p28_root", "id")).Desc(),
	)

	built, err := Build(
		context.Background(),
		parent,
		p35RootParts(),
		NewNode(p28RootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	wantInner := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantInner) {
		t.Fatalf("expected the inside installParentOrdColumn path without a LIMIT, missing %q:\n%s", wantInner, built.SQL)
	}
	if strings.Contains(built.SQL, `) AS "p28_root"`) {
		t.Fatalf(`must not wrap with the outer alias when the parent has no LIMIT:`+"\n%s", built.SQL)
	}
}

// TestP35OuterOrd_JoinColumnIsHoisted (P-39): an ORDER BY expression that
// references a table other than the parent's own no longer falls back to
// the inside installParentOrdColumn path. outerParentOrd hoists it onto the
// parent's own SELECT list as "_sl_row_sort0" (sortColPrefix) and the outer
// window's ORDER BY references that alias instead, so the parent's own
// LIMIT still keeps its top-N sort (issue P-39 §6.1). Before P-39, this
// same query fell back to the inside placement (installParentOrdColumn);
// this test used to be named TestP35OuterOrd_JoinColumnFallsBack and
// asserted that fallback — P-39 §5.3's "p35_outer_ord_test.go stays
// byte-identical" claim does not hold for this one case, because hoisting
// a joined table's ORDER BY column is exactly the behavior P-39 adds.
func TestP35OuterOrd_JoinColumnIsHoisted(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("p28_root", "id")),
		sm.From(psql.Quote("p28_root")),
		sm.OrderBy(psql.Quote("other_table", "name")).Desc(),
		sm.Limit(20),
	)

	built, err := Build(
		context.Background(),
		parent,
		p35RootParts(),
		NewNode(p28RootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `) AS "p28_root"`) {
		t.Fatalf(`an ORDER BY referencing a table other than the parent's own must still be hoisted onto the outer window (P-39), missing wrapper alias ") AS \"p28_root\"":`+"\n%s", built.SQL)
	}
	wantHoistedSelect := `"other_table"."name" AS "_sl_row_sort0"`
	if !strings.Contains(built.SQL, wantHoistedSelect) {
		t.Fatalf("expected the joined column hoisted onto the parent's own SELECT list, missing %q:\n%s", wantHoistedSelect, built.SQL)
	}
	wantOuterWindow := `row_number()OVER ( ORDER BY "p28_root"."_sl_row_sort0" DESC) AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantOuterWindow) {
		t.Fatalf("expected the outer window to ORDER BY the hoisted alias, missing %q:\n%s", wantOuterWindow, built.SQL)
	}
	if strings.Contains(built.SQL, "_sl_row_ord") {
		t.Fatalf("must not fall back to the inside installParentOrdColumn column when the joined column is hoisted:\n%s", built.SQL)
	}
}

func TestP35OuterOrd_DistinctSkipsOrd(t *testing.T) {
	parent := psql.Select(
		sm.Distinct(),
		sm.Columns(psql.Quote("p28_root", "id")),
		sm.From(psql.Quote("p28_root")),
		sm.OrderBy(psql.Quote("p28_root", "id")).Desc(),
		sm.Limit(20),
	)

	built, err := Build(
		context.Background(),
		parent,
		p35RootParts(),
		NewNode(p28RootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_ord") {
		t.Fatalf("DISTINCT must not trigger the inside installParentOrdColumn column either (same opt-out as before P-35):\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, `) AS "p28_root"`) {
		t.Fatalf("DISTINCT must not trigger the outer window:\n%s", built.SQL)
	}
	if !strings.Contains(built.SQL, `row_number() OVER () AS "_sl_ord"`) {
		t.Fatalf("DISTINCT must keep the no-ORDER-BY-guarantee fallback row_number() OVER ():\n%s", built.SQL)
	}
}
