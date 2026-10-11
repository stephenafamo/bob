package sectionload

// DB-free end-to-end / integration tests for column restriction
// (SelectOptions.Only/Except).
//
// The harness reuses thenKeyFakeExecutor + thenKeyFakeRows/thenKeyColsRows (defined in
// other test files of this package). The fixtures are new rather than
// reusing thenKeyRoot/thenKeyMid: thenKeyMidBuf.TargetsFor ignores its cols argument and
// always returns 3 columns, which cannot work with tests that narrow columns
// via ColumnPlan, so dedicated colRestrict*Buf types mimic the switch-per-column
// style of generated code.
//
// "Only/Except on a multi-hop edge is an explicit error" is covered by
// columnselect_test.go's TestResolveColumnPlans_MultiHopEdgeIsError
// (multiHopTwoHopEdge + resolveColumnPlans ->
// *MultiHopColumnRestrictionUnsupportedError), so it is not duplicated here.
// The other key checks are likewise covered elsewhere: an edge missing
// ChildKeyFromBuf by validate_test.go's
// TestValidatePlan_RestrictedEdgeMissingChildKeyFromBufIsError, and an
// Except that excludes a key column by columnplan_test.go's
// TestNewColumnPlan_ExceptExcludingKeyIsError and columnselect_test.go's
// TestResolveColumnPlans_ExceptExcludingKeyIsError.

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
)

// --- fixtures: colRestrictRoot -> colRestrictMid -> colRestrictLeaf (all single-hop) -----------

type (
	colRestrictRoot    struct{ ID int }
	colRestrictRootBuf struct{ id int }
)

func (b *colRestrictRootBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *colRestrictRootBuf) BuildFor(modelCols []Column) colRestrictRoot {
	return colRestrictRoot{ID: b.id}
}

type (
	colRestrictMid    struct{ ID, RootID, OtherKey int }
	colRestrictMidBuf struct{ id, rootID, otherKey int }
)

func (b *colRestrictMidBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.rootID, &b.otherKey}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "root_id":
			out[i] = &b.rootID
		case "other_key":
			out[i] = &b.otherKey
		default:
			panic("colRestrictMidBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *colRestrictMidBuf) BuildFor(modelCols []Column) colRestrictMid {
	if modelCols == nil {
		return colRestrictMid{ID: b.id, RootID: b.rootID, OtherKey: b.otherKey}
	}
	m := colRestrictMid{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "root_id":
			m.RootID = b.rootID
		case "other_key":
			m.OtherKey = b.otherKey
		}
	}
	return m
}

type (
	colRestrictLeaf    struct{ ID, OtherKey int }
	colRestrictLeafBuf struct{ id, otherKey int }
)

func (b *colRestrictLeafBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.otherKey}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "other_key":
			out[i] = &b.otherKey
		default:
			panic("colRestrictLeafBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *colRestrictLeafBuf) BuildFor(modelCols []Column) colRestrictLeaf {
	return colRestrictLeaf{ID: b.id, OtherKey: b.otherKey}
}

func colRestrictRootTableParts() TableParts[colRestrictRoot] {
	return TableParts[colRestrictRoot]{
		Table:   "cr_root",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[colRestrictRoot] { return &colRestrictRootBuf{} },
	}
}

func colRestrictRootParts() partsAdapter[colRestrictRoot] {
	return partsAdapter[colRestrictRoot]{colRestrictRootTableParts()}
}

// colRestrictRootMidEdge is the parent edge (grandparent -> parent): root -> mid, restricted via
// Options (Only/Except), with ChildKeyFromBuf set to simulate what
// 01_edges.go.tpl generates for every edge.
func colRestrictRootMidEdge(opts SelectOptions) Edge[colRestrictRoot, colRestrictMid] {
	return Edge[colRestrictRoot, colRestrictMid]{
		RelName:  "Mid",
		FromCols: []string{"id"},
		ToCols:   []string{"root_id"},
		ToMany:   false,
		Options:  opts,
		Child: TableParts[colRestrictMid]{
			Table: "cr_mid",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "root_id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[colRestrictMid] { return &colRestrictMidBuf{} },
		},
		ParentKey: func(p colRestrictRoot) []any { return []any{p.ID} },
		ChildKey:  func(c colRestrictMid) []any { return []any{c.RootID} },
		ChildKeyFromBuf: func(b Buf[colRestrictMid]) []any {
			return []any{b.(*colRestrictMidBuf).rootID}
		},
		Stitch: func(p colRestrictRoot, cs []colRestrictMid) error { return nil },
	}
}

// colRestrictMidLeafEdge is the child edge (parent -> child): mid -> leaf, unrestricted.
// FromCols=["other_key"] is a column that is NOT
// auto-protected by colRestrictRootMidEdge's own NewColumnPlan (which only knows
// about "root_id", its own toCols()).
func colRestrictMidLeafEdge() Edge[colRestrictMid, colRestrictLeaf] {
	return Edge[colRestrictMid, colRestrictLeaf]{
		RelName:  "Leaf",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		ToMany:   true,
		Child: TableParts[colRestrictLeaf]{
			Table: "cr_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[colRestrictLeaf] { return &colRestrictLeafBuf{} },
		},
		ParentKey: func(p colRestrictMid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c colRestrictLeaf) []any { return []any{c.OtherKey} },
		ParentKeyFromBuf: func(b Buf[colRestrictMid]) []any {
			return []any{b.(*colRestrictMidBuf).otherKey}
		},
		Stitch: func(p colRestrictMid, cs []colRestrictLeaf) error { return nil },
	}
}

// colRestrictMidLeafCountEdge is colRestrictMidLeafEdge's count-node counterpart, for
// the restricted-parent-with-count-child test.
func colRestrictMidLeafCountEdge() CountEdge[colRestrictMid, colRestrictLeaf] {
	return CountEdge[colRestrictMid, colRestrictLeaf]{
		RelName:  "LeafCount",
		FromCols: []string{"other_key"},
		ToCols:   []string{"other_key"},
		Child: TableParts[colRestrictLeaf]{
			Table: "cr_leaf",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "other_key", DBType: "int4"},
			},
			NewBuf: func() Buf[colRestrictLeaf] { return &colRestrictLeafBuf{} },
		},
		ParentKey: func(p colRestrictMid) []any { return []any{p.OtherKey} },
		ChildKey:  func(c colRestrictLeaf) []any { return []any{c.OtherKey} },
		ParentKeyFromBuf: func(b Buf[colRestrictMid]) []any {
			return []any{b.(*colRestrictMidBuf).otherKey}
		},
		// ChildKeyFromBuf mirrors what the count template generates. Without
		// it, childKeyFromBufAny (node.go) falls back to
		// ChildKey(b.BuildFor(nil)), calling a full-column BuildFor(nil) on a
		// count-only scan buffer limited to keyCols; this fixture could then
		// no longer detect that path surviving in generated code. As in
		// generated code, it calls b.BuildFor(...) with an explicit column
		// list instead of reading fields through a type assertion (a direct
		// type-assertion read would not exercise the path through BuildFor
		// that generated code actually takes).
		ChildKeyFromBuf: func(b Buf[colRestrictLeaf]) []any {
			o := b.BuildFor([]Column{
				{Name: "other_key", DBType: "int4"},
			})
			return []any{o.OtherKey}
		},
		SetCount: func(p colRestrictMid, n int64) error { return nil },
	}
}

// colRestrictPanicOnFullBuildLeafBuf panics on BuildFor(nil) — the same sentinel
// technique columnplan_test.go's panicIfFullBuildBuf uses, applied to
// colRestrictLeaf. Used by TestCountEdge_ChildKeyFromBufAvoidsFullBuild to prove
// childKeyFromBufAny actually calls the ChildKeyFromBuf override instead
// of falling back to BuildFor(nil).
type colRestrictPanicOnFullBuildLeafBuf struct{ colRestrictLeafBuf }

// colRestrictFullBuildPanicMessage is the sentinel panic value BuildFor(nil) raises
// comparing the RECOVERED value
// against this exact string, on both the success and failure sides of
// TestCountEdge_ChildKeyFromBufAvoidsFullBuild, tells a genuine
// BuildFor(nil) call apart from an unrelated type-assertion panic that
// would otherwise also satisfy a bare `recover() != nil` check.
const colRestrictFullBuildPanicMessage = "BuildFor(nil) must not be called when ChildKeyFromBuf is set"

func (b *colRestrictPanicOnFullBuildLeafBuf) BuildFor(modelCols []Column) colRestrictLeaf {
	if modelCols == nil {
		panic(colRestrictFullBuildPanicMessage)
	}
	return b.colRestrictLeafBuf.BuildFor(modelCols)
}

// TestCountEdge_ChildKeyFromBufAvoidsFullBuild calls childKeyFromBufAny
// directly on colRestrictMidLeafCountEdge with ChildKeyFromBuf set, and checks without
// a DB that (a) the ChildKeyFromBuf override returns the correct value and
// never calls BuildFor(nil), and (b) removing the override makes
// childKeyFromBufAny actually fall back to BuildFor(nil) (a full-column
// build).
//
// Case (a) is given a buffer that panics on a full-column build, so a
// runtime that wrongly also called BuildFor(nil) around the override would
// be detected. Case (b) also checks that the recovered value equals
// colRestrictFullBuildPanicMessage, so an unrelated panic (such as a failed type
// assertion) is not mistaken for it.
func TestCountEdge_ChildKeyFromBufAvoidsFullBuild(t *testing.T) {
	edge := colRestrictMidLeafCountEdge()

	// (a): with the override set, a buffer that panics on a full-column
	// build still yields the correct value without going through
	// BuildFor(nil).
	panicBuf := &colRestrictPanicOnFullBuildLeafBuf{colRestrictLeafBuf{id: 10, otherKey: 999}}
	got := edge.childKeyFromBufAny(bufAdapter[colRestrictLeaf]{panicBuf})
	if len(got) != 1 || got[0].(int) != 999 {
		t.Fatalf("childKeyFromBufAny = %v, want [999]", got)
	}

	// (b): without the override, the actual fallback to BuildFor(nil) is
	// confirmed via the panic (reproducing the danger of a full-column build
	// on a count-only buffer). The recovered value must equal
	// colRestrictFullBuildPanicMessage so it is not confused with an unrelated
	// type-assertion panic.
	edge.ChildKeyFromBuf = nil
	panicBuf2 := &colRestrictPanicOnFullBuildLeafBuf{colRestrictLeafBuf{id: 10, otherKey: 999}}
	func() {
		defer func() {
			r := recover()
			if r != colRestrictFullBuildPanicMessage {
				t.Fatalf("panic = %v, want %q", r, colRestrictFullBuildPanicMessage)
			}
		}()
		edge.childKeyFromBufAny(bufAdapter[colRestrictLeaf]{panicBuf2})
	}()
}

// --- restriction on a nested (grandchild) level ------------------

// TestExecIsland_RestrictedGrandparentAllowsUnrestrictedGrandchild is
// checks, down to execIsland's return value, a 3-level tree: grandparent
// (root) -> parent (mid, restricted by Except("id")) -> child (leaf,
// unrestricted).
func TestExecIsland_RestrictedGrandparentAllowsUnrestrictedGrandchild(t *testing.T) {
	midEdge := colRestrictRootMidEdge(Except("id"))
	leafEdge := colRestrictMidLeafEdge()
	tree := NewNode(midEdge, NewNode(leafEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 2 {
		t.Fatalf("sections: got %d, want 2 (mid, leaf)", len(plan.sections))
	}

	root := colRestrictRootParts()
	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, mid.root_id, mid.other_key,
	//         leaf.id, leaf.other_key
	// (mid's childCols() is ColumnPlan.All()=[root_id, other_key]: Except("id")
	// excludes only "id", and "other_key" is not excluded so it stays in
	// Model automatically. leaf is unrestricted, so it is [id, other_key] in
	// table order.)
	exec := thenKeyFakeExecutor{rows: &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0, 0, 0},
			{int64(1), nil, 0, 100, 999, 0, 0},
			{int64(2), nil, 0, 0, 0, 300, 999},
		}},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}

	if len(res.items[0]) != 1 || res.items[0][0].(colRestrictRoot).ID != 100 {
		t.Fatalf("root items = %v, want [{ID:100}]", res.items[0])
	}
	midSet := plan.sections[0].set
	if len(res.items[midSet]) != 1 {
		t.Fatalf("mid items: got %d, want 1", len(res.items[midSet]))
	}
	mid := res.items[midSet][0].(colRestrictMid)
	if mid.RootID != 100 || mid.OtherKey != 999 {
		t.Fatalf("mid = %+v, want RootID=100 OtherKey=999", mid)
	}
	if mid.ID != 0 {
		t.Fatalf("mid.ID = %d, want 0 (excluded by Except(\"id\"), never scanned into the model)", mid.ID)
	}
	leafSet := plan.sections[1].set
	if len(res.items[leafSet]) != 1 {
		t.Fatalf("leaf items: got %d, want 1", len(res.items[leafSet]))
	}
	leaf := res.items[leafSet][0].(colRestrictLeaf)
	if leaf.ID != 300 || leaf.OtherKey != 999 {
		t.Fatalf("leaf = %+v, want ID=300 OtherKey=999", leaf)
	}
}

// --- restricted parent + count child ---------------------------------

// TestExecIsland_RestrictedParentWithCountChild checks that a tree whose
// parent edge has Except and whose child is a RelCount (CountEdge) passes
// validatePlan and that execIsland returns the correct count.
func TestExecIsland_RestrictedParentWithCountChild(t *testing.T) {
	midEdge := colRestrictRootMidEdge(Except("id"))
	countEdge := colRestrictMidLeafCountEdge()
	tree := NewNode(midEdge, NewCountNode(countEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	if len(plan.sections) != 1 || len(plan.counts) != 1 {
		t.Fatalf("plan = %+v, want 1 section (mid) + 1 count", plan)
	}

	root := colRestrictRootParts()
	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, mid.root_id, mid.other_key,
	//         count.other_key(keyCol), _sl_cnt<set>
	exec := thenKeyFakeExecutor{rows: &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0, 0, nil},
			{int64(1), nil, 0, 100, 999, 0, nil},
			{int64(2), nil, 0, 0, 0, 999, int64(3)},
		}},
	}}

	res, err := execIsland(context.Background(), exec, psql.RawQuery("SELECT 1"), root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}

	countSet := plan.counts[0].set
	got := res.counts[countSet]
	if len(got) != 1 || got[999] != 3 {
		t.Fatalf("counts[%d] = %v, want map[999:3]", countSet, got)
	}
}

// --- no automatic protection of the child edge's fromCols ------------

// TestValidatePlan_ChildFromColsNamedInOnlyPasses is the passing side of the
// child-fromCols rule: the path where Only names the child edge's fromCols
// explicitly (this exercises SelectOptions.Only being wired into
// resolveEdgeColumnPlan, which an Except-based test would not). other_key is
// named in Only explicitly, and root_id (the edge's own toCols) is added to
// Internal automatically.
func TestValidatePlan_ChildFromColsNamedInOnlyPasses(t *testing.T) {
	// other_key is named explicitly as the child edge's fromCols.
	// root_id is the edge's own toCols, so it is added to Internal
	// automatically.
	midEdge := colRestrictRootMidEdge(Only("other_key"))
	leafEdge := colRestrictMidLeafEdge()
	tree := NewNode(midEdge, NewNode(leafEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}

	columnPlan := resolved[0].edge.(Edge[colRestrictRoot, colRestrictMid]).ColumnPlan
	if len(columnPlan.Model) != 1 || columnPlan.Model[0].Name != "other_key" {
		t.Fatalf("Model = %v, want [other_key]", columnPlan.Model)
	}
	if len(columnPlan.Internal) != 1 || columnPlan.Internal[0].Name != "root_id" {
		t.Fatalf("Internal = %v, want [root_id]", columnPlan.Internal)
	}

	if err := validatePlan(resolved, nil); err != nil {
		t.Fatalf("validatePlan: %v", err)
	}

	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}
	root := colRestrictRootParts()
	parent := psql.RawQuery("SELECT id FROM cr_root")
	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	if !strings.Contains(sql, `"other_key"`) {
		t.Fatalf("other_key (the child edge's fromCols) is missing from the SQL:\n%s", sql)
	}
	if !strings.Contains(sql, `"root_id"`) {
		t.Fatalf("root_id (the edge's own toCols, auto-added to Internal) is missing from the SQL:\n%s", sql)
	}
}

// TestValidatePlan_ChildFromColsOmittedFromOnlyIsError is the rejecting side
// of the child-fromCols rule. When Only names just the edge's own join key
// (root_id) and deliberately omits the other_key the child edge needs, the
// result must be *MissingRequiredColumnError.
func TestValidatePlan_ChildFromColsOmittedFromOnlyIsError(t *testing.T) {
	// Name only the edge's own join key and deliberately omit the
	// other_key the child edge needs.
	midEdge := colRestrictRootMidEdge(Only("root_id"))
	leafEdge := colRestrictMidLeafEdge()
	tree := NewNode(midEdge, NewNode(leafEdge))

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	err = validatePlan(resolved, nil)
	var want *MissingRequiredColumnError
	if !errors.As(err, &want) {
		t.Fatalf("validatePlan err = %v, want *MissingRequiredColumnError", err)
	}
	if want.RelName != "Mid" || want.ChildRelName != "Leaf" || want.Column != "other_key" {
		t.Fatalf("err fields = %+v, want RelName=Mid ChildRelName=Leaf Column=other_key", want)
	}
}

// --- end-to-end: a join key not named by Except reaches the model ------

type (
	colRestrictSecretChild    struct{ ID, ParentID int }
	colRestrictSecretChildBuf struct {
		id, parentID int
		secret       string
	}
)

func (b *colRestrictSecretChildBuf) TargetsFor(cols []Column) []any {
	if len(cols) == 0 {
		return []any{&b.id, &b.parentID, &b.secret}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		case "id":
			out[i] = &b.id
		case "parent_id":
			out[i] = &b.parentID
		case "secret":
			out[i] = &b.secret
		default:
			panic("colRestrictSecretChildBuf: unknown column " + c.Name)
		}
	}
	return out
}

func (b *colRestrictSecretChildBuf) BuildFor(modelCols []Column) colRestrictSecretChild {
	if modelCols == nil {
		return colRestrictSecretChild{ID: b.id, ParentID: b.parentID}
	}
	m := colRestrictSecretChild{}
	for _, c := range modelCols {
		switch c.Name {
		case "id":
			m.ID = b.id
		case "parent_id":
			m.ParentID = b.parentID
			// secret has no corresponding field: if Except("secret") is in
			// effect, secret never appears in modelCols.
		}
	}
	return m
}

// colRestrictSecretEdge takes opts explicitly so callers can exercise both Only and
// Except through the same fixture: TestExecIsland_ExceptUnmentionedKeyReachesModel
// passes Except("secret"), and the table-driven
// TestBuild_ResolvesOnlyExceptThroughProductionWiring passes both Only and
// Except.
func colRestrictSecretEdge(opts SelectOptions) Edge[colRestrictRoot, colRestrictSecretChild] {
	return Edge[colRestrictRoot, colRestrictSecretChild]{
		RelName:  "SecretChild",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Options:  opts,
		Child: TableParts[colRestrictSecretChild]{
			Table: "cr_secret_children",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "parent_id", DBType: "int4"},
				{Name: "secret", DBType: "text"},
			},
			NewBuf: func() Buf[colRestrictSecretChild] { return &colRestrictSecretChildBuf{} },
		},
		ParentKey: func(p colRestrictRoot) []any { return []any{p.ID} },
		ChildKey:  func(c colRestrictSecretChild) []any { return []any{c.ParentID} },
		ChildKeyFromBuf: func(b Buf[colRestrictSecretChild]) []any {
			return []any{b.(*colRestrictSecretChildBuf).parentID}
		},
		Stitch: func(p colRestrictRoot, cs []colRestrictSecretChild) error { return nil },
	}
}

// TestExecIsland_ExceptUnmentionedKeyReachesModel executes a single-hop edge
// with Except("secret") and checks that the join key column ("parent_id"),
// which is not excluded, (a) appears in the SQL, (b) is scanned, and (c)
// fills the child model's field.
func TestExecIsland_ExceptUnmentionedKeyReachesModel(t *testing.T) {
	e := colRestrictSecretEdge(Except("secret"))
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	root := colRestrictRootParts()
	parent := psql.RawQuery("SELECT id FROM cr_root")
	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	// (a) it appears in the SQL.
	if !strings.Contains(sql, `"parent_id"`) {
		t.Fatalf("(a) parent_id is missing from the SQL:\n%s", sql)
	}
	if strings.Contains(sql, `"secret"`) {
		t.Fatalf("secret, excluded by Except(\"secret\"), appears in the SQL:\n%s", sql)
	}

	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, child.id, child.parent_id
	// (ColumnPlan.Model=[id, parent_id]: everything except secret goes to Model
	// automatically)
	exec := thenKeyFakeExecutor{rows: &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0},
			{int64(1), nil, 0, 500, 100},
		}},
	}}
	res, err := execIsland(context.Background(), exec, q, root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	set := plan.sections[0].set
	if len(res.items[set]) != 1 {
		t.Fatalf("child items: got %d, want 1", len(res.items[set]))
	}
	child := res.items[set][0].(colRestrictSecretChild)
	// (b)(c): parent_id is scanned and lands in the model's field.
	if child.ParentID != 100 {
		t.Fatalf("child.ParentID = %d, want 100 (parent_id must be scanned AND copied into the model)", child.ParentID)
	}
	if child.ID != 500 {
		t.Fatalf("child.ID = %d, want 500", child.ID)
	}
}

// TestExecIsland_OnlyRestrictsModelToNamedColumns is the Only counterpart of
// the test above. It executes a single-hop edge with Only("id") and checks
// at the execIsland level that (a) id appears in the SQL and secret does not,
// (b) the named id is scanned into the model, and (c) the unnamed parent_id
// (the join key, auto-added to Internal) is scanned but does not reach the
// model and stays zero. Except and Only take different paths through
// NewColumnPlan (except mode vs only mode), so the only-mode behavior of
// "unnamed columns stay zero" is checked all the way through execIsland.
func TestExecIsland_OnlyRestrictsModelToNamedColumns(t *testing.T) {
	e := colRestrictSecretEdge(Only("id"))
	tree := NewNode(e)

	resolved, err := resolveColumnPlans([]Node{tree})
	if err != nil {
		t.Fatalf("resolveColumnPlans: %v", err)
	}
	plan, err := planIsland(resolved)
	if err != nil {
		t.Fatalf("planIsland: %v", err)
	}

	root := colRestrictRootParts()
	parent := psql.RawQuery("SELECT id FROM cr_root")
	q, err := composeIsland(parent, root, plan, "")
	if err != nil {
		t.Fatalf("composeIsland: %v", err)
	}
	sql, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("bob.Build: %v", err)
	}
	// (a) SQL check: id appears and secret does not.
	if !strings.Contains(sql, `."id"`) {
		t.Fatalf("(a) id is missing from the SQL:\n%s", sql)
	}
	if strings.Contains(sql, `"secret"`) {
		t.Fatalf("secret, not named by Only(\"id\"), appears in the SQL:\n%s", sql)
	}

	cols := expectedColumns(root, plan)
	// layout: _sl_set, _sl_ord, root.id, child.id, child.parent_id
	// (ColumnPlan.Model=[id], Internal=[parent_id]: the same two columns as
	// the except-mode test are scanned, but the Model/Internal assignment is
	// reversed)
	exec := thenKeyFakeExecutor{rows: &thenKeyColsRows{
		cols: cols,
		thenKeyFakeRows: &thenKeyFakeRows{rowsData: [][]any{
			{int64(0), int64(1), 100, 0, 0},
			{int64(1), nil, 0, 500, 100},
		}},
	}}
	res, err := execIsland(context.Background(), exec, q, root, plan, nil)
	if err != nil {
		t.Fatalf("execIsland: %v", err)
	}
	set := plan.sections[0].set
	if len(res.items[set]) != 1 {
		t.Fatalf("child items: got %d, want 1", len(res.items[set]))
	}
	child := res.items[set][0].(colRestrictSecretChild)
	// (b): the id named by Only("id") is scanned into the model.
	if child.ID != 500 {
		t.Fatalf("child.ID = %d, want 500 (named by Only(\"id\"), so it must be set)", child.ID)
	}
	// (c): parent_id (Internal), not named, is scanned but does not reach the
	// model and stays zero.
	if child.ParentID != 0 {
		t.Fatalf("child.ParentID = %d, want 0 (an Internal column does not reach the model)", child.ParentID)
	}
}

// --- verification through run.go's own wiring ---------------------------
//
// The tests above all call resolveColumnPlans by hand and pass the result
// straight to planIsland/composeIsland/execIsland, so they never exercise the
// wiring by which run.go's All/Build actually call resolveColumnPlans.
// Removing those two call sites (in All and at the top of Build) would leave
// the tests above green, a false green, so the tests below call Build and
// All directly.

// TestBuild_ResolvesOnlyExceptThroughProductionWiring verifies that run.go's
// Build actually calls resolveColumnPlans: without calling
// resolveColumnPlans by hand, it calls only Build(ctx, parentQuery, parts,
// nodes...) and checks that the Options are reflected in the SQL. It needs
// no DB connection or fake executor (Build only composes SQL and does not
// execute it). It is table-driven so that both the Only and the Except paths
// are checked through Build.
func TestBuild_ResolvesOnlyExceptThroughProductionWiring(t *testing.T) {
	tests := []struct {
		name string
		opts SelectOptions
	}{
		{name: "Only", opts: Only("id")},
		{name: "Except", opts: Except("secret")},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := colRestrictSecretEdge(tc.opts)
			parent := psql.RawQuery("SELECT id FROM cr_root")

			built, err := Build(context.Background(), parent, colRestrictRootTableParts(), NewNode(e))
			if err != nil {
				t.Fatalf("Build: %v (if Build did not call resolveColumnPlans, the Options would have no effect and composeIsland could still succeed with a zero-value ColumnPlan and unrestricted SQL, so the absence of an error does not distinguish that case; the SQL content checks below are the actual test)", err)
			}
			// Matching a bare `"id"` would also match `"parent_id"` as a
			// substring (strings.Contains), so the check could pass on
			// parent_id alone even if id itself is missing from the SELECT.
			// Match the table-qualified form `."id"` (for example
			// `"cr_root"."id"` or `_sl_t"."id"`), which is distinct from
			// `."parent_id"`.
			if !strings.Contains(built.SQL, `."id"`) {
				t.Fatalf("Build().SQL has no table-qualified \"id\"; resolveColumnPlans may not be called inside Build:\n%s", built.SQL)
			}
			if !strings.Contains(built.SQL, `"parent_id"`) {
				t.Fatalf("Build().SQL has no parent_id; resolveColumnPlans may not be called inside Build:\n%s", built.SQL)
			}
			if strings.Contains(built.SQL, `"secret"`) {
				t.Fatalf("secret must never appear in the SQL when Only/Except is in effect (if resolveColumnPlans is not called inside Build, ColumnPlan stays zero and the SQL is unrestricted, including secret):\n%s", built.SQL)
			}
		})
	}
}

// TestAll_RejectsMultiHopColumnRestrictionThroughProductionWiring verifies
// from the error path that run.go's All actually calls resolveColumnPlans: a
// tree whose multiHopTwoHopEdge (a multi-hop edge) has Only is passed directly to
// All(...) without calling resolveColumnPlans by hand, and
// *MultiHopColumnRestrictionUnsupportedError must be returned. If that call
// in All were removed, this multi-hop edge (with ColumnPlan left at zero)
// would pass validatePlan without an error, because validateMultiHopEdge
// accepts the tree regardless of the ColumnPlan state.
func TestAll_RejectsMultiHopColumnRestrictionThroughProductionWiring(t *testing.T) {
	e := multiHopTwoHopEdge()
	e.Options = Only("label")
	parent := psql.RawQuery("SELECT id FROM mh_parents")

	_, err := All[multiHopParent](context.Background(), thenKeyFakeExecutor{rows: &thenKeyFakeRows{}}, parent, multiHopParentParts(), NewNode(e))
	var want *MultiHopColumnRestrictionUnsupportedError
	if !errors.As(err, &want) {
		t.Fatalf("All(...) err = %v, want *MultiHopColumnRestrictionUnsupportedError (resolveColumnPlans may not be called inside All)", err)
	}
}
