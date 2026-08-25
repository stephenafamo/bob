package driver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	driver "github.com/stephenafamo/bob/gen/bobgen-psql/driver"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/gen/plugins"
)

// stubMods is a minimal drivers.TemplateInclude for a query with no mods.
type stubMods struct{}

func (stubMods) IncludeInTemplate(language.Importer) string { return "_ = q" }

// Mirrors issue #602: the generated struct for a configured query should carry
// the tags set in the config (e.g. json), the same way model structs do. Before
// the fix the query row struct only ever had a db tag.
func TestQueryRowStructHasConfiguredTags(t *testing.T) {
	nullable := false

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	gomod := "module scratch.local/repro\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}
	queryDir := filepath.Join(tmp, "queries")
	if err := os.MkdirAll(queryDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	info := &drivers.DBInfo[any, any, driver.IndexExtra]{
		Driver: "github.com/jackc/pgx/v5/stdlib",
		Tables: drivers.Tables[any, driver.IndexExtra]{
			{
				Key:  "widgets",
				Name: "widgets",
				Columns: []drivers.Column{
					{Name: "widget_id", DBType: "integer", Type: "int32"},
					{Name: "widget_name", DBType: "text", Type: "string"},
				},
				Constraints: drivers.Constraints[any]{
					Primary: &drivers.Constraint[any]{Name: "widgets_pkey", Columns: []string{"widget_id"}},
				},
			},
		},
		QueryFolders: []drivers.QueryFolder{
			{
				Path: queryDir,
				Files: []drivers.QueryFile{
					{
						Path: filepath.Join(queryDir, "get_widget.sql"),
						Queries: []drivers.Query{
							{
								Name: "GetWidget",
								SQL:  "SELECT widget_id, widget_name FROM widgets",
								Type: bob.QueryTypeSelect,
								Columns: drivers.QueryCols{
									{Name: "widget_id", DBName: "widget_id", TypeName: "int32", Nullable: &nullable},
									{Name: "widget_name", DBName: "widget_name", TypeName: "string", Nullable: &nullable},
								},
								Mods: stubMods{},
							},
						},
					},
				},
			},
		},
	}

	outputPlugins := plugins.Setup[any, any, driver.IndexExtra](
		plugins.PresetAll, gen.PSQLTemplates,
	)

	state := &gen.State[any]{Config: gen.Config[any]{
		NoTests: true,
		Tags:    []string{"json"},
	}}
	if err := gen.Run[any, any, driver.IndexExtra](
		context.Background(), state, stubDriver{types: helpers.Types(), info: info}, outputPlugins...,
	); err != nil {
		t.Fatalf("gen.Run: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(queryDir, "get_widget.bob.go"))
	if err != nil {
		t.Fatalf("reading generated query file: %v", err)
	}

	for _, want := range []string{
		"`db:\"widget_id\" json:\"widget_id\"`",
		"`db:\"widget_name\" json:\"widget_name\"`",
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("generated query struct missing tag %s\n---\n%s", want, got)
		}
	}
}
