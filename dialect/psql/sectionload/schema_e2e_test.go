package sectionload

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/orm"
)

const p12SchemaTestDSN = "host=127.0.0.1 port=5432 user=vuls dbname=testdb sslmode=disable password=test application_name=p12_use_schema"

type p12SchemaParent struct {
	ID       int64
	Label    string
	Children []*p12SchemaChild
}

type p12SchemaChild struct {
	ID       int64
	ParentID int64
	Label    string
}

type p12SchemaParentBuf struct {
	id    sql.NullInt64
	label sql.NullString
}

func (b *p12SchemaParentBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.label}
}

func (b *p12SchemaParentBuf) BuildFor(modelCols []Column) *p12SchemaParent {
	return &p12SchemaParent{
		ID:    b.id.Int64,
		Label: b.label.String,
	}
}

type p12SchemaChildBuf struct {
	id       sql.NullInt64
	parentID sql.NullInt64
	label    sql.NullString
}

func (b *p12SchemaChildBuf) TargetsFor(cols []Column) []any {
	return []any{&b.id, &b.parentID, &b.label}
}

func (b *p12SchemaChildBuf) BuildFor(modelCols []Column) *p12SchemaChild {
	return &p12SchemaChild{
		ID:       b.id.Int64,
		ParentID: b.parentID.Int64,
		Label:    b.label.String,
	}
}

func TestUseSchemaSelectsParentAndRelationsFromSameTenant(t *testing.T) {
	ctx := context.Background()

	sqlDB, err := sql.Open("postgres", p12SchemaTestDSN)
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
		`CREATE SCHEMA p12_use_schema_a`,
		`CREATE SCHEMA p12_use_schema_b`,
		`CREATE TABLE p12_use_schema_a.p12_parents (
			id bigint PRIMARY KEY,
			label text NOT NULL
		)`,
		`CREATE TABLE p12_use_schema_a.p12_children (
			id bigint PRIMARY KEY,
			parent_id bigint NOT NULL,
			label text NOT NULL
		)`,
		`CREATE TABLE p12_use_schema_b.p12_parents (
			id bigint PRIMARY KEY,
			label text NOT NULL
		)`,
		`CREATE TABLE p12_use_schema_b.p12_children (
			id bigint PRIMARY KEY,
			parent_id bigint NOT NULL,
			label text NOT NULL
		)`,
		`INSERT INTO p12_use_schema_a.p12_parents VALUES (1, 'parent-a')`,
		`INSERT INTO p12_use_schema_a.p12_children VALUES (11, 1, 'child-a')`,
		`INSERT INTO p12_use_schema_b.p12_parents VALUES (1, 'parent-b')`,
		`INSERT INTO p12_use_schema_b.p12_children VALUES (21, 1, 'child-b')`,
	}
	for _, stmt := range fixtureSQL {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("fixture statement failed:\n%s\nerror: %v", stmt, err)
		}
	}

	parentParts := TableParts[*p12SchemaParent]{
		Table: "p12_parents",
		Columns: []Column{
			{Name: "id", DBType: "int8"},
			{Name: "label", DBType: "text"},
		},
		NewBuf: func() Buf[*p12SchemaParent] {
			return &p12SchemaParentBuf{}
		},
	}
	childParts := TableParts[*p12SchemaChild]{
		Table: "p12_children",
		Columns: []Column{
			{Name: "id", DBType: "int8"},
			{Name: "parent_id", DBType: "int8"},
			{Name: "label", DBType: "text"},
		},
		NewBuf: func() Buf[*p12SchemaChild] {
			return &p12SchemaChildBuf{}
		},
	}
	edge := Edge[*p12SchemaParent, *p12SchemaChild]{
		RelName:  "Children",
		FromCols: []string{"id"},
		ToCols:   []string{"parent_id"},
		ToMany:   true,
		Child:    childParts,
		ParentKey: func(parent *p12SchemaParent) []any {
			return []any{parent.ID}
		},
		ChildKey: func(child *p12SchemaChild) []any {
			return []any{child.ParentID}
		},
		Stitch: func(parent *p12SchemaParent, children []*p12SchemaChild) error {
			parent.Children = children
			return nil
		},
	}

	// Reuse this exact query object for both runtime schemas.
	parentQuery := psql.Select(
		sm.Columns(psql.Quote("id"), psql.Quote("label")),
		sm.From(orm.SchemaTable("p12_parents")),
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
			schema:      "p12_use_schema_a",
			otherSchema: "p12_use_schema_b",
			parentLabel: "parent-a",
			childLabel:  "child-a",
		},
		{
			schema:      "p12_use_schema_b",
			otherSchema: "p12_use_schema_a",
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
					if !strings.Contains(built.SQL, `"`+tenant.schema+`"."p12_parents"`) {
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
