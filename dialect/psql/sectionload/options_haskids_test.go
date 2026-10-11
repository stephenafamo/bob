package sectionload

// P-02: a with-children ("hasKids") node's ORDER BY option must be applied
// once, at its CTE body (route 1), and the resulting ordinal column must
// carry through to that node's own UNION ALL branch (route 3, which reads
// the CTE back) without recomputing it — options are never applied twice
// (issue P-02 §10.4 item 5 / A-02-2). A sibling leaf edge with no options
// of its own must be completely unaffected.
//
// Tree: p11CapParent -[Child, ORDER BY id DESC]-> p11CapChild (hasKids,
// because it has its own child) -[Grandchild, no options]-> p02Grandchild.

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

type p02Grandchild struct {
	ID      int
	ChildID int
}

type p02GrandchildBuf struct{ id, childID int }

func (b *p02GrandchildBuf) TargetsFor(cols []Column) []any {
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
			panic("p02GrandchildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p02GrandchildBuf) BuildFor(modelCols []Column) p02Grandchild {
	if len(modelCols) == 0 {
		return p02Grandchild{ID: b.id, ChildID: b.childID}
	}
	m := p02Grandchild{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "child_id":
			m.ChildID = b.childID
		default:
			panic("p02GrandchildBuf: unknown column " + c.Name)
		}
	}
	return m
}

func p02GrandchildEdge() Edge[p11CapChild, p02Grandchild] {
	return Edge[p11CapChild, p02Grandchild]{
		RelName:  "Grandchild",
		FromCols: []string{"id"},
		ToCols:   []string{"child_id"},
		ToMany:   true,
		Child: TableParts[p02Grandchild]{
			Schema: "",
			Table:  "p02_grandchildren",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "child_id", DBType: "int4"},
			},
			NewBuf: func() Buf[p02Grandchild] { return &p02GrandchildBuf{} },
		},
		ParentKey: func(p p11CapChild) []any { return []any{p.ID} },
		ChildKey:  func(c p02Grandchild) []any { return []any{c.ChildID} },
		Stitch:    func(p p11CapChild, cs []p02Grandchild) error { return nil },
	}
}

type p02ParentBuf struct{ id int }

func (b *p02ParentBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		default:
			panic("p02ParentBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *p02ParentBuf) BuildFor(modelCols []Column) p11CapParent {
	return p11CapParent{ID: b.id}
}

func TestComposeIsland_HasKidsOrderBy_AppliedOnceAtCTE(t *testing.T) {
	orderedChild := p11CapEdgeWith(Mods(sm.OrderBy("id").Desc()))
	tree := NewNode(orderedChild, NewNode(p02GrandchildEdge()))

	root := partsAdapter[p11CapParent]{TableParts[p11CapParent]{
		Schema:  "",
		Table:   "p11_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[p11CapParent] { return &p02ParentBuf{} },
	}}
	plan, err := planIsland([]Node{tree})
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	parent := psql.RawQuery("SELECT id FROM p11_parents")

	q, err := composeIsland(parent, root, plan, DefaultMaxColumns, "")
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
	if strings.Contains(sql, "p02_grandchildren\") AS \"_sl_t\", row_number") {
		t.Fatalf("grandchild leaf must not gain an ordinal column; got:\n%s", sql)
	}
}
