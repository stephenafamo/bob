package sectionload

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/orm"
)

type useSchemaParent struct {
	ID       int64
	Label    string
	Children []*useSchemaChild
}

type useSchemaChild struct {
	ID       int64
	ParentID int64
	Label    string
}

type useSchemaParentBuf struct {
	id    sql.NullInt64
	label sql.NullString
}

func (b *useSchemaParentBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.label}
}

func (b *useSchemaParentBuf) BuildFor(modelCols []Column) *useSchemaParent {
	return &useSchemaParent{
		ID:    b.id.Int64,
		Label: b.label.String,
	}
}

type useSchemaChildBuf struct {
	id       sql.NullInt64
	parentID sql.NullInt64
	label    sql.NullString
}

func (b *useSchemaChildBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.parentID, &b.label}
}

func (b *useSchemaChildBuf) BuildFor(modelCols []Column) *useSchemaChild {
	return &useSchemaChild{
		ID:       b.id.Int64,
		ParentID: b.parentID.Int64,
		Label:    b.label.String,
	}
}

func TestUseSchemaSelectsParentAndRelationsFromSameTenant(t *testing.T) {
	ctx := context.Background()

	dsn := os.Getenv("SECTIONLOAD_TEST_DSN")
	if dsn == "" {
		t.Skip("SECTIONLOAD_TEST_DSN is not set")
	}

	sqlDB, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open testdb: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if err := sqlDB.PingContext(ctx); err != nil {
		t.Skipf("testdb not reachable: %v", err)
	}

	db := bob.NewDB(sqlDB)
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("BeginTx: %v", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	fixtureSQL := []string{
		`CREATE SCHEMA sch_use_schema_a`,
		`CREATE SCHEMA sch_use_schema_b`,
		`CREATE TABLE sch_use_schema_a.sch_parents (
			id bigint PRIMARY KEY,
			label text NOT NULL
		)`,
		`CREATE TABLE sch_use_schema_a.sch_children (
			id bigint PRIMARY KEY,
			parent_id bigint NOT NULL,
			label text NOT NULL
		)`,
		`CREATE TABLE sch_use_schema_b.sch_parents (
			id bigint PRIMARY KEY,
			label text NOT NULL
		)`,
		`CREATE TABLE sch_use_schema_b.sch_children (
			id bigint PRIMARY KEY,
			parent_id bigint NOT NULL,
			label text NOT NULL
		)`,
		`INSERT INTO sch_use_schema_a.sch_parents VALUES (1, 'parent-a')`,
		`INSERT INTO sch_use_schema_a.sch_children VALUES (11, 1, 'child-a')`,
		`INSERT INTO sch_use_schema_b.sch_parents VALUES (1, 'parent-b')`,
		`INSERT INTO sch_use_schema_b.sch_children VALUES (21, 1, 'child-b')`,
	}
	for _, stmt := range fixtureSQL {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("fixture statement failed:\n%s\nerror: %v", stmt, err)
		}
	}

	parentParts := TableParts[*useSchemaParent]{
		Table: "sch_parents",
		Columns: []Column{
			{Name: "id", DBType: "int8"},
			{Name: "label", DBType: "text"},
		},
		NewBuf: func() Buf[*useSchemaParent] {
			return &useSchemaParentBuf{}
		},
	}
	childParts := TableParts[*useSchemaChild]{
		Table: "sch_children",
		Columns: []Column{
			{Name: "id", DBType: "int8"},
			{Name: "parent_id", DBType: "int8"},
			{Name: "label", DBType: "text"},
		},
		NewBuf: func() Buf[*useSchemaChild] {
			return &useSchemaChildBuf{}
		},
	}
	edge := Edge[*useSchemaParent, *useSchemaChild]{
		RelName:  "Children",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Child:    childParts,
		ParentKey: func(parent *useSchemaParent) []any {
			return []any{parent.ID}
		},
		ChildKey: func(child *useSchemaChild) []any {
			return []any{child.ParentID}
		},
		Stitch: func(parent *useSchemaParent, children []*useSchemaChild) error {
			parent.Children = children
			return nil
		},
	}

	// Reuse this exact query object for both runtime schemas.
	parentQuery := psql.Select(
		sm.Columns(psql.Quote("id"), psql.Quote("label")),
		sm.From(orm.SchemaTable("sch_parents")),
		sm.OrderBy("id"),
	)

	routes := []struct {
		name string
		node Node
	}{
		{name: "same_island", node: NewNode(edge)},
		{name: "then_boundary", node: Then(NewNode(edge))},
	}
	tenants := []struct {
		schema      string
		otherSchema string
		parentLabel string
		childLabel  string
	}{
		{
			schema:      "sch_use_schema_a",
			otherSchema: "sch_use_schema_b",
			parentLabel: "parent-a",
			childLabel:  "child-a",
		},
		{
			schema:      "sch_use_schema_b",
			otherSchema: "sch_use_schema_a",
			parentLabel: "parent-b",
			childLabel:  "child-b",
		},
	}

	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			for _, tenant := range tenants {
				t.Run(tenant.schema, func(t *testing.T) {
					tenantCtx := psql.UseSchema(ctx, tenant.schema)

					built, err := Build(
						tenantCtx,
						parentQuery,
						parentParts,
						route.node,
					)
					if err != nil {
						t.Fatalf("Build: %v", err)
					}
					// NOTE: Build renders only the FIRST island (run.go:130-132),
					// so for the then_boundary route these two assertions cover
					// the parent query alone. The relation side of that route is
					// covered by the All(...) assertions below.
					if !strings.Contains(built.SQL, `"`+tenant.schema+`"."sch_parents"`) {
						t.Fatalf("parent table is not schema-qualified with %q:\n%s", tenant.schema, built.SQL)
					}
					if strings.Contains(built.SQL, `"`+tenant.otherSchema+`".`) {
						t.Fatalf("SQL unexpectedly references the other schema %q:\n%s", tenant.otherSchema, built.SQL)
					}

					rows, err := All(
						tenantCtx,
						tx,
						parentQuery,
						parentParts,
						route.node,
					)
					if err != nil {
						t.Fatalf("All: %v", err)
					}
					if len(rows) != 1 {
						t.Fatalf("parent count: got %d, want 1", len(rows))
					}
					if rows[0].Label != tenant.parentLabel {
						t.Fatalf("parent label: got %q, want %q", rows[0].Label, tenant.parentLabel)
					}
					if len(rows[0].Children) != 1 {
						t.Fatalf("child count: got %d, want 1", len(rows[0].Children))
					}
					if rows[0].Children[0].Label != tenant.childLabel {
						t.Fatalf("child label: got %q, want %q", rows[0].Children[0].Label, tenant.childLabel)
					}
				})
			}
		})
	}
}
