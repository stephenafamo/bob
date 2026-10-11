package sectionload

// P-41: composeIsland's inner-ordering branch (parentOrdCol != "", the
// "default" case in the switch at compose.go:721) used to list only
// root.cols() in _sl_src's own SELECT, dropping any declared PreloadCount
// alias (plan.declaredCounts) that the parent query also carries.
// section0List (compose.go:897-901) then read "_sl_src"."__count_<Rel>"
// back off a column _sl_src never actually output. This test fixes that in
// place: a parent query with both an ORDER BY (forcing the inner-ordering
// path, same fixture style as p34_parent_orderby_test.go) and a declared
// PreloadCount column (the AppendPreloadSelect technique
// mappermod_buildtime_test.go already uses to fake one without a real
// generated model — Build() only extracts the declarations off the
// parentQuery's own SELECT list and sets them into plan.declaredCounts, it
// never needs a countPreloadable root type) must carry the count column
// through _sl_src by name.
import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func TestP41_DeclaredCountSurvivesInnerOrdering(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Quote("id")),
		sm.From(psql.Quote("p28_root")),
		sm.OrderBy("id"),
	)

	var inner *dialect.SelectQuery
	parent.Apply(bob.ModFunc[*dialect.SelectQuery](
		func(q *dialect.SelectQuery) { inner = q },
	))
	// declaredCountPreloadsShim only reads the alias off the rendered SQL
	// text's trailing `AS "__count_<Rel>"` (countpreload_shim.go), so a raw
	// literal ending that way is indistinguishable, for this test's
	// purposes, from bob's own generated correlated-subquery count column.
	inner.AppendPreloadSelect(psql.Raw(
		`(SELECT count(*) FROM p28_mid WHERE p28_mid.root_id = p28_root.id) AS "__count_Whatever"`,
	))

	built, err := Build(
		context.Background(),
		parent,
		p05BaselineRootParts(),
		NewNode(p28RootMidEdge()),
	)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	t.Logf("SQL:\n%s", built.SQL)

	// This checks that _sl_src places the declared count column right after
	// the parent columns, and does not expose _sl_row_ord (the
	// inner-ordering helper). section0List references the declared count
	// column by name — the final result's column order is _sl_ord, root
	// cols, declared count cols.
	wantSrcCols := `"q"."id", "q"."__count_Whatever", row_number() OVER (ORDER BY "q"."_sl_row_ord") AS "_sl_ord"`
	if !strings.Contains(built.SQL, wantSrcCols) {
		t.Fatalf("missing declared count column in _sl_src's SELECT list %q:\n%s", wantSrcCols, built.SQL)
	}

	// section 0 must read the same column back off _sl_src by name — this
	// is the reference the bug left dangling (previously _sl_src had no
	// "__count_Whatever" output column for this to point at).
	wantSection0 := `"_sl_src"."__count_Whatever"`
	if !strings.Contains(built.SQL, wantSection0) {
		t.Fatalf("missing section 0 reference to the declared count column %q:\n%s", wantSection0, built.SQL)
	}
}
