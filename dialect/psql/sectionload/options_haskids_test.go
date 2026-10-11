package sectionload

// A with-children ("hasKids") node's ORDER BY option must be applied
// once, at its CTE body (route 1), and the resulting ordinal column must
// carry through to that node's own UNION ALL branch (route 3, which reads
// the CTE back) without recomputing it — options are never applied twice
// A sibling leaf edge with no options
// of its own must be completely unaffected.
//
// Tree: builderCapParent -[Child, ORDER BY id DESC]-> builderCapChild (hasKids,
// because it has its own child) -[Grandchild, no options]-> hasKidsGrandchild.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type hasKidsGrandchild struct {
	ID      int
	ChildID int
}

type hasKidsGrandchildBuf struct{ id, childID int }

func (b *hasKidsGrandchildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.childID}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "child_id":
			out[i] = &b.childID
		default:
			panic("hasKidsGrandchildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *hasKidsGrandchildBuf) BuildFor(modelCols []Column) hasKidsGrandchild {
	if len(modelCols) == 0 {
		return hasKidsGrandchild{ID: b.id, ChildID: b.childID}
	}
	m := hasKidsGrandchild{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "child_id":
			m.ChildID = b.childID
		default:
			panic("hasKidsGrandchildBuf: unknown column " + c.Name)
		}
	}
	return m
}

func hasKidsGrandchildEdge() Edge[builderCapChild, hasKidsGrandchild] {
	return Edge[builderCapChild, hasKidsGrandchild]{
		RelName:  "Grandchild",
		FromCols: []string{"id"},
		ToCols:   []string{"child_id"},
		ToMany:   true,
		Child: TableParts[hasKidsGrandchild]{
			Schema: "",
			Table:  "hk_grandchildren",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "child_id", DBType: "int4"},
			},
			NewBuf: func() Buf[hasKidsGrandchild] { return &hasKidsGrandchildBuf{} },
		},
		ParentKey: func(p builderCapChild) []any { return []any{p.ID} },
		ChildKey:  func(c hasKidsGrandchild) []any { return []any{c.ChildID} },
		Stitch:    func(p builderCapChild, cs []hasKidsGrandchild) error { return nil },
	}
}

type hasKidsParentBuf struct{ id int }

func (b *hasKidsParentBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		default:
			panic("hasKidsParentBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *hasKidsParentBuf) BuildFor(modelCols []Column) builderCapParent {
	return builderCapParent{ID: b.id}
}

func TestComposeIsland_HasKidsOrderBy_AppliedOnceAtCTE(t *testing.T) {
	orderedChild := builderCapEdgeWith(Mods(sm.OrderBy("id").Desc()))
	tree := NewNode(orderedChild, NewNode(hasKidsGrandchildEdge()))

	root := partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema:  "",
		Table:   "bld_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[builderCapParent] { return &hasKidsParentBuf{} },
	}}
	plan, err := planIsland([]Node{tree})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM bld_parents")

	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	t.Logf("SQL:\n%s", sql)

	// Exactly two row_number() calls: the pre-existing root-restore one
	// (_sl_src's row_number() OVER ()) and the one this test adds for the
	// ordered child's CTE body. Not three: the grandchild leaf and the
	// child's own UNION-branch read-back must not recompute it.
	if got := strings.Count(sql, "row_number()"); got != 2 {
		t.Fatalf("want exactly 2 row_number() calls (root-restore + child CTE body), got %d:\n%s", got, sql)
	}

	cteIdx := strings.Index(sql, quoteIdent(cteName(1)))
	if cteIdx < 0 {
		t.Fatalf("expected CTE %s not found:\n%s", cteName(1), sql)
	}
	if !strings.Contains(sql[:strings.Index(sql, "UNION ALL")], ordCol) {
		t.Fatalf("expected the ordinal column %q inside the WITH block (CTE body); got:\n%s", ordCol, sql)
	}

	// The ordinal must be threaded through to colOrd (_sl_ord) somewhere
	// in the query (the child's own UNION ALL branch, route 3).
	if !strings.Contains(sql, `"`+ordCol+`" AS "`+colOrd+`"`) {
		t.Fatalf("expected %q read back into %q; got:\n%s", ordCol, colOrd, sql)
	}

	// The grandchild (leaf, no options) must show no trace of ordering.
	// Locate its own branch by its table name and check no row_number
	// reference sits near it — approximated here by confirming the
	// grandchild's table name never appears adjacent to "row_number".
	if strings.Contains(sql, "hk_grandchildren\") AS \"_sl_t\", row_number") {
		t.Fatalf("grandchild leaf must not gain an ordinal column; got:\n%s", sql)
	}
}
