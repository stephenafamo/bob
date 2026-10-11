package sectionload

// The inner placement (installParentOrdColumn) puts a row_number() window
// between Limit and Sort (WindowAgg -> Sort -> Limit) when the parent query
// has a LIMIT and the ORDER BY columns have no index, so PostgreSQL cannot use
// a top-N heapsort and sorts every row. For a LIMIT-ed parent query whose
// ORDER BY references only the parent table's own columns, placing
// row_number() outside the parent subquery (outerParentOrd/run.go) lets LIMIT
// apply first so the window only sees the surviving rows.
//
// Like parent_orderby_test.go, these are Build()-only (no DB) SQL-text
// assertions on fragments of the bob.Build output.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

// outerOrdRootParts is multiHopBaselineRootParts's 2-column variant (id,
// ssvc_priority): outerParentOrd requires every ORDER BY expression to
// name a column present in TableParts.Columns, so the multi-key case
// below needs a second column beyond multiHopBaselineRootParts's single "id".
// NewBuf is never invoked by these Build()-only tests, so reusing
// thenKeyRootBuf (which only has an id field) is safe.
func outerOrdRootParts() TableParts[thenKeyRoot] {
	return TableParts[thenKeyRoot]{
		Table: "tk_root",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "ssvc_priority", DBType: "text"},
		},
		NewBuf: func() Buf[thenKeyRoot] { return &thenKeyRootBuf{} },
	}
}

// outerOrdParentQuery builds a LIMIT-ed parent query over tk_root, ORDERed BY
// the given mods (each expected to reference only tk_root's own columns
// via psql.Quote("tk_root", ...), the shape outerParentOrd requires).
func outerOrdParentQuery(limit int, orderMods ...bob.Mod[*dialect.SelectQuery]) bob.Query {
	mods := append([]bob.Mod[*dialect.SelectQuery]{
		sm.Columns(psql.Quote("tk_root", "id")),
		sm.From(psql.Quote("tk_root")),
	}, orderMods...)
	mods = append(mods, sm.Limit(limit))
	return psql.Select(mods...)
}

func TestOuterOrd_LimitUsesOuterWindow(t *testing.T) {
	tests := []struct {
		name           string
		orderMods      []bob.Mod[*dialect.SelectQuery]
		wantOuterOrder string // the ORDER BY clause inside the outer row_number() window and the parent's own trailing ORDER BY
	}{
		{
			name: "single_key_desc",
			orderMods: []bob.Mod[*dialect.SelectQuery]{
				sm.OrderBy(psql.Quote("tk_root", "id")).Desc(),
			},
			wantOuterOrder: `"tk_root"."id" DESC`,
		},
		{
			name: "single_key_desc_nulls_last",
			orderMods: []bob.Mod[*dialect.SelectQuery]{
				sm.OrderBy(psql.Quote("tk_root", "id")).Desc().NullsLast(),
			},
			wantOuterOrder: `"tk_root"."id" DESC NULLS LAST`,
		},
		{
			name: "two_keys",
			orderMods: []bob.Mod[*dialect.SelectQuery]{
				sm.OrderBy(psql.Quote("tk_root", "ssvc_priority")).Desc().NullsLast(),
				sm.OrderBy(psql.Quote("tk_root", "id")).Asc(),
			},
			wantOuterOrder: `"tk_root"."ssvc_priority" DESC NULLS LAST, "tk_root"."id" ASC`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			built, err := Build(
				context.Background(),
				outerOrdParentQuery(20, tt.orderMods...),
				outerOrdRootParts(),
				NewNode(thenKeyRootMidEdge()),
			)
			if err != nil {
				t.Fatalf("Build: %v", err)
			}
			t.Logf("SQL:\n%s", built.SQL)

			wantWindow := `"tk_root".*, row_number()OVER ( ORDER BY ` + tt.wantOuterOrder + `) AS "_sl_ord"`
			if !strings.Contains(built.SQL, wantWindow) {
				t.Fatalf("missing outer row_number window %q:\n%s", wantWindow, built.SQL)
			}
			if !strings.Contains(built.SQL, `) AS "tk_root"`) {
				t.Fatalf(`missing outer wrapper alias ") AS \"tk_root\"":`+"\n%s", built.SQL)
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

func TestOuterOrd_NoLimitStaysInner(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("tk_root", "id")),
		sm.From(psql.Quote("tk_root")),
		sm.OrderBy(psql.Quote("tk_root", "id")).Desc(),
	)

	built, err := Build(
		context.Background(),
		parent,
		outerOrdRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	wantInner := `row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantInner) {
		t.Fatalf("expected the inside installParentOrdColumn path without a LIMIT, missing %q:\n%s", wantInner, built.SQL)
	}
	if strings.Contains(built.SQL, `) AS "tk_root"`) {
		t.Fatalf(`must not wrap with the outer alias when the parent has no LIMIT:`+"\n%s", built.SQL)
	}
}

// TestOuterOrd_JoinColumnIsHoisted: an ORDER BY expression that
// references a table other than the parent's own no longer falls back to
// the inside installParentOrdColumn path. outerParentOrd hoists it onto the
// parent's own SELECT list as "_sl_row_sort0" (sortColPrefix) and the outer
// window's ORDER BY references that alias instead, so the parent's own
// LIMIT still keeps its top-N sort. Such a query previously fell back to the
// inside placement (installParentOrdColumn); hoisting a joined table's
// ORDER BY column is the behavior this test pins.
func TestOuterOrd_JoinColumnIsHoisted(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("tk_root", "id")),
		sm.From(psql.Quote("tk_root")),
		sm.OrderBy(psql.Quote("other_table", "name")).Desc(),
		sm.Limit(20),
	)

	built, err := Build(
		context.Background(),
		parent,
		outerOrdRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if !strings.Contains(built.SQL, `) AS "tk_root"`) {
		t.Fatalf(`an ORDER BY referencing a table other than the parent's own must still be hoisted onto the outer window, missing wrapper alias ") AS \"tk_root\"":`+"\n%s", built.SQL)
	}
	wantHoistedSelect := `"other_table"."name" AS "_sl_row_sort0"`
	if !strings.Contains(built.SQL, wantHoistedSelect) {
		t.Fatalf("expected the joined column hoisted onto the parent's own SELECT list, missing %q:\n%s", wantHoistedSelect, built.SQL)
	}
	wantOuterWindow := `row_number()OVER ( ORDER BY "tk_root"."_sl_row_sort0" DESC) AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantOuterWindow) {
		t.Fatalf("expected the outer window to ORDER BY the hoisted alias, missing %q:\n%s", wantOuterWindow, built.SQL)
	}
	if strings.Contains(built.SQL, "_sl_row_ord") {
		t.Fatalf("must not fall back to the inside installParentOrdColumn column when the joined column is hoisted:\n%s", built.SQL)
	}
}

func TestOuterOrd_DistinctSkipsOrd(t *testing.T) {
	parent := psql.Select(
		sm.Distinct(),
		sm.Columns(psql.Quote("tk_root", "id")),
		sm.From(psql.Quote("tk_root")),
		sm.OrderBy(psql.Quote("tk_root", "id")).Desc(),
		sm.Limit(20),
	)

	built, err := Build(
		context.Background(),
		parent,
		outerOrdRootParts(),
		NewNode(thenKeyRootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	if strings.Contains(built.SQL, "_sl_row_ord") {
		t.Fatalf("DISTINCT must not trigger the inside installParentOrdColumn column either (same opt-out as the outer placement):\n%s", built.SQL)
	}
	if strings.Contains(built.SQL, `) AS "tk_root"`) {
		t.Fatalf("DISTINCT must not trigger the outer window:\n%s", built.SQL)
	}
	if !strings.Contains(built.SQL, `row_number() OVER () AS "_sl_ord"`) {
		t.Fatalf("DISTINCT must keep the no-ORDER-BY-guarantee fallback row_number() OVER ():\n%s", built.SQL)
	}
}
