package sectionload

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/scan"
)

// sectionload's relation SELECTs must respect a runtime
// psql.UseSchema(ctx, ...) override exactly the way bob's own generated
// views do (dialect/psql/view.go's NameExpr): a generated schema that is
// non-empty is always fixed and never overridden; a generated schema of ""
// delegates to orm.SchemaTable, which reads the context value at WriteSQL
// time. Physical relation table sources in
// routes 1, 2, and 4 pass through tableExpr. Route 3 intentionally reads a
// query-local CTE through psql.Quote and must remain schema-unqualified.
// The tests below verify the rendered SQL and production wiring without a
// database; schema_e2e_test.go separately verifies that PostgreSQL returns
// parent and relation rows from the same runtime-selected schema.

func TestTableExprRespectsUseSchemaContext(t *testing.T) {
	renderExpr := func(ctx context.Context, e bob.Expression) string {
		t.Helper()
		var sb strings.Builder
		if _, err := e.WriteSQL(ctx, &sb, pdialect.Dialect, 1); err != nil {
			t.Fatalf("WriteSQL: %v", err)
		}
		return sb.String()
	}

	t.Run("generated schema empty: no UseSchema -> unqualified", func(t *testing.T) {
		got := renderExpr(context.Background(), tableExpr("", "posts"))
		if got != `"posts"` {
			t.Fatalf("got %q, want %q", got, `"posts"`)
		}
	})

	t.Run("generated schema empty: UseSchema(tenant_a) -> qualified", func(t *testing.T) {
		ctx := psql.UseSchema(context.Background(), "tenant_a")
		got := renderExpr(ctx, tableExpr("", "posts"))
		if got != `"tenant_a"."posts"` {
			t.Fatalf("got %q, want %q", got, `"tenant_a"."posts"`)
		}
	})

	t.Run("generated schema non-empty: fixed, UseSchema does not override", func(t *testing.T) {
		ctx := psql.UseSchema(context.Background(), "tenant_a")
		got := renderExpr(ctx, tableExpr("fixed_schema", "posts"))
		if got != `"fixed_schema"."posts"` {
			t.Fatalf("got %q, want %q (a non-empty generated schema must never be overridden by ctx)", got, `"fixed_schema"."posts"`)
		}
	})

	t.Run("generated schema non-empty: no UseSchema -> still fixed", func(t *testing.T) {
		got := renderExpr(context.Background(), tableExpr("fixed_schema", "posts"))
		if got != `"fixed_schema"."posts"` {
			t.Fatalf("got %q, want %q", got, `"fixed_schema"."posts"`)
		}
	})
}

// Same-island path (route 2: a leaf node's semi-join UNION branch).
func TestSameIslandRespectsUseSchemaContext(t *testing.T) {
	child := TableParts[builderCapChild]{
		Schema: "",
		Table:  "bld_children",
		Columns: []Column{
			{Name: "id", DBType: "int4"},
			{Name: "parent_id", DBType: "int4"},
		},
		NewBuf: func() Buf[builderCapChild] { return &builderCapChildBuf{} },
	}
	spec := edgeSelectSpec{
		edge:        builderCapEdge(),
		cols:        child.Columns,
		from:        tableExpr(child.Schema, child.Table),
		alias:       srcAlias,
		semiFromCTE: cteSrc,
	}
	q, err := buildEdgeSelect(spec)
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}

	sqlDefault, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("build (default ctx): %v", err)
	}
	if !strings.Contains(sqlDefault, `FROM "bld_children"`) {
		t.Fatalf("default-ctx SQL should reference the unqualified table; got:\n%s", sqlDefault)
	}

	ctxTenant := psql.UseSchema(context.Background(), "tenant_a")
	sqlTenant, _, err := bob.Build(ctxTenant, q)
	if err != nil {
		t.Fatalf("build (tenant ctx): %v", err)
	}
	if !strings.Contains(sqlTenant, `FROM "tenant_a"."bld_children"`) {
		t.Fatalf("tenant-ctx SQL should reference the schema-qualified table; got:\n%s", sqlTenant)
	}
}

// Then(...) boundary path (route 4), mirroring TestBuilderRoute4ThenBoundaryParentQuery.
func TestThenBoundaryRespectsUseSchemaContext(t *testing.T) {
	e := builderCapEdge()
	childRoot := e.child()
	q, err := buildEdgeSelect(edgeSelectSpec{
		edge:  e,
		cols:  childRoot.cols(),
		from:  tableExpr(childRoot.schemaName(), childRoot.tableName()),
		alias: srcAlias,
		keys:  [][]any{{1}, {2}, {3}},
	})
	if err != nil {
		t.Fatalf("buildEdgeSelect: %v", err)
	}

	sqlDefault, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("build (default ctx): %v", err)
	}
	if !strings.Contains(sqlDefault, `FROM "bld_children"`) {
		t.Fatalf("default-ctx SQL should reference the unqualified table; got:\n%s", sqlDefault)
	}

	ctxTenant := psql.UseSchema(context.Background(), "tenant_b")
	sqlTenant, _, err := bob.Build(ctxTenant, q)
	if err != nil {
		t.Fatalf("build (tenant ctx): %v", err)
	}
	if !strings.Contains(sqlTenant, `FROM "tenant_b"."bld_children"`) {
		t.Fatalf("tenant-ctx SQL should reference the schema-qualified table; got:\n%s", sqlTenant)
	}
}

// Route 1 (a with-children node's own MATERIALIZED CTE, composeIsland):
// this third tableExpr call site needs dedicated coverage
// through a fully composed query with nesting, only via the isolated
// TestTableExprRespectsUseSchemaContext. Build a 3-level tree (parent ->
// builderCapChild [has a grandchild, so it becomes a with-kids node] ->
// grandchild) so builderCapChild's own CTE body goes through route 1.

type useSchemaCapGrandchild struct {
	ID      int
	ChildID int
}

type useSchemaCapGrandchildBuf struct{ id, childID int }

func (b *useSchemaCapGrandchildBuf) TargetsFor(cols []Column) []any { return []any{&b.id, &b.childID} }

func (b *useSchemaCapGrandchildBuf) BuildFor(modelCols []Column) useSchemaCapGrandchild {
	return useSchemaCapGrandchild{ID: b.id, ChildID: b.childID}
}

type useSchemaCapParentBuf struct{ id int }

func (b *useSchemaCapParentBuf) TargetsFor(cols []Column) []any { return []any{&b.id} }
func (b *useSchemaCapParentBuf) BuildFor(modelCols []Column) builderCapParent {
	return builderCapParent{ID: b.id}
}

func TestNestedNodeCTERespectsUseSchemaContext(t *testing.T) {
	grandchildEdge := Edge[builderCapChild, useSchemaCapGrandchild]{
		RelName:  "Grandchild",
		FromCols: []string{"id"},
		ToCols:   []string{"child_id"},
		ToMany:   true,
		Child: TableParts[useSchemaCapGrandchild]{
			Schema: "",
			Table:  "sch_grandchildren",
			Columns: []Column{
				{Name: "id", DBType: "int4"},
				{Name: "child_id", DBType: "int4"},
			},
			NewBuf: func() Buf[useSchemaCapGrandchild] { return &useSchemaCapGrandchildBuf{} },
		},
		ParentKey: func(p builderCapChild) []any { return []any{p.ID} },
		ChildKey:  func(c useSchemaCapGrandchild) []any { return []any{c.ChildID} },
		Stitch:    func(p builderCapChild, cs []useSchemaCapGrandchild) error { return nil },
	}
	// builderCapChild now has a nested child of its own, so planIsland marks it
	// hasKids -> its section is emitted as route 1's own MATERIALIZED CTE.
	tree := NewNode(builderCapEdge(), NewNode(grandchildEdge))

	root := partsAdapter[builderCapParent]{TableParts[builderCapParent]{
		Schema:  "",
		Table:   "bld_parents",
		Columns: []Column{{Name: "id", DBType: "int4"}},
		NewBuf:  func() Buf[builderCapParent] { return &useSchemaCapParentBuf{} },
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

	sqlDefault, _, err := bob.Build(context.Background(), q)
	if err != nil {
		t.Fatalf("build (default ctx): %v", err)
	}
	if !strings.Contains(sqlDefault, `FROM "bld_children"`) {
		t.Fatalf("default-ctx SQL should reference the unqualified table in the with-kids node's CTE; got:\n%s", sqlDefault)
	}
	// Route 2: the grandchild edge is
	// a leaf under builderCapChild's own CTE, so its UNION branch goes through
	// unionBranch's real route-2 semi-join path — the same production
	// wiring TestSameIslandRespectsUseSchemaContext exercises in
	// isolation. Asserting it here, inside a fully composed query, is what
	// that isolated test cannot prove by itself.
	if !strings.Contains(sqlDefault, `FROM "sch_grandchildren"`) {
		t.Fatalf("default-ctx SQL should reference the grandchild's unqualified table (route 2, via the real unionBranch path); got:\n%s", sqlDefault)
	}

	ctxTenant := psql.UseSchema(context.Background(), "tenant_c")
	sqlTenant, _, err := bob.Build(ctxTenant, q)
	if err != nil {
		t.Fatalf("build (tenant ctx): %v", err)
	}
	if !strings.Contains(sqlTenant, `FROM "tenant_c"."bld_children"`) {
		t.Fatalf("tenant-ctx SQL should reference the schema-qualified table in the with-kids node's CTE; got:\n%s", sqlTenant)
	}
	// Route 2 under UseSchema, same production path as above.
	if !strings.Contains(sqlTenant, `FROM "tenant_c"."sch_grandchildren"`) {
		t.Fatalf("tenant-ctx SQL should reference the grandchild's schema-qualified table (route 2, via the real unionBranch path); got:\n%s", sqlTenant)
	}
	// Route 3: builderCapChild's own UNION branch reads its
	// already-filtered CTE back by name (unionBranch's route-3 path,
	// compose.go), never the physical table — a CTE reference is never
	// schema-qualified (CTEs are query-local, not a database object a
	// tenant schema could own). This must hold regardless of ctx.
	//
	// Route 3 must actually read the with-kids node back from its own CTE
	// (unionBranch's route-3 path) — asserting only the ABSENCE of a
	// schema-qualified CTE reference would keep passing if route 3
	// regressed to reading the physical table directly.
	const route3From = `FROM "_sl_n1" AS "_sl_t"`
	if !strings.Contains(sqlTenant, route3From) {
		t.Fatalf("tenant-ctx SQL should read the with-kids node from its own unqualified CTE (route 3); missing %q:\n%s", route3From, sqlTenant)
	}
	if strings.Contains(sqlTenant, `"tenant_c"."_sl_n`) {
		t.Fatalf("tenant-ctx SQL schema-qualifies a CTE self-reference (route 3); a CTE name must never be schema-prefixed:\n%s", sqlTenant)
	}
}

// Route 4 (Then(...) boundary, runThen in hooks.go):
// TestThenBoundaryRespectsUseSchemaContext calls buildEdgeSelect directly,
// never runThen itself, so a schema-context regression in runThen's own query construction would not be caught. This
// test drives runThen through a fake bob.Executor that captures the SQL it
// is actually asked to run, with no real DB.
type useSchemaCapturingExecutor struct{ capturedSQL string }

func (e *useSchemaCapturingExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	e.capturedSQL = query
	return nil, errUseSchemaStopAfterCapture
}

func (e *useSchemaCapturingExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errUseSchemaStopAfterCapture
}

var errUseSchemaStopAfterCapture = errors.New("useSchema test: stop after capturing the query")

func TestThenBoundaryRunThenRespectsUseSchemaContext(t *testing.T) {
	e := builderCapEdge()
	tp := thenPlan{edge: e}

	run := func(ctx context.Context) string {
		t.Helper()
		exec := &useSchemaCapturingExecutor{}
		parent := builderCapParent{ID: 1}
		err := runThen(ctx, exec, tp, []any{parent}, nil)
		if !errors.Is(err, errUseSchemaStopAfterCapture) {
			t.Fatalf("runThen: expected to reach the capturing executor, got: %v", err)
		}
		return exec.capturedSQL
	}

	sqlDefault := run(context.Background())
	if !strings.Contains(sqlDefault, `FROM "bld_children"`) {
		t.Fatalf("default-ctx runThen SQL should reference the unqualified table; got:\n%s", sqlDefault)
	}

	sqlTenant := run(psql.UseSchema(context.Background(), "tenant_d"))
	if !strings.Contains(sqlTenant, `FROM "tenant_d"."bld_children"`) {
		t.Fatalf("tenant-ctx runThen SQL should reference the schema-qualified table; got:\n%s", sqlTenant)
	}
}
