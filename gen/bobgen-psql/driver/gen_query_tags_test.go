package driver_test

import (
	_ "embed"
	"fmt"
	"os"
	"os/exec"
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
	testgen "github.com/stephenafamo/bob/test/gen"
)

// stubMods is a minimal drivers.TemplateInclude for a query with no mods.
type stubMods struct{}

func (stubMods) IncludeInTemplate(language.Importer) string { return "_ = q" }

//go:embed testdata/query_tags_test.go
var queryTagsTest []byte

func TestQueryRowStructHasConfiguredTags(t *testing.T) {
	nullable := false
	canBeNull := true

	mod, err := exec.CommandContext(t.Context(), "go", "env", "GOMOD").Output()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	t.Chdir(tmp)
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "-mod=mod")
	root := filepath.Dir(strings.TrimSpace(string(mod)))
	gomod := fmt.Sprintf(
		"module scratch.local/repro\n\ngo 1.26.0\n\nrequire github.com/stephenafamo/bob v0.0.0\nreplace github.com/stephenafamo/bob => %q\n",
		filepath.ToSlash(root),
	)
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}
	queryDir := filepath.Join(tmp, "queries")
	if err := os.MkdirAll(queryDir, 0o700); err != nil {
		t.Fatal(err)
	}
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
								SQL:  "SELECT 11 AS widget_id, 22 AS widget_id, 'sample' AS widget_name, 'hidden' AS secret_token",
								Type: bob.QueryTypeSelect,
								Columns: drivers.QueryCols{
									{Name: "widget_id", DBName: "widget_id", TypeName: "int32", Nullable: &nullable},
									{Name: "widget_id", DBName: "widget_id", TypeName: "int32", Nullable: &nullable},
									{Name: "display_name", DBName: "widget_name", TypeName: "string", Nullable: &nullable},
									{Name: "secret_token", DBName: "secret_token", TypeName: "string", Nullable: &nullable},
								},
								Mods: stubMods{},
							},
							{
								Name: "GroupedWidgets",
								SQL:  `SELECT id, label, child AS "children.id" FROM sample ORDER BY seq`,
								Type: bob.QueryTypeSelect,
								Columns: drivers.QueryCols{
									{Name: "id", DBName: "id", TypeName: "int32", Nullable: &nullable},
									{Name: "label", DBName: "label", TypeName: "string", Nullable: &canBeNull},
									{Name: "children.id", DBName: "children.id", TypeName: "int32", Nullable: &canBeNull},
								},
								Mods: stubMods{},
							},
							{
								Name: "GroupedBytes",
								SQL:  `SELECT CAST(label AS BLOB) AS code, child AS "children.id" FROM sample WHERE label <> '' ORDER BY seq`,
								Type: bob.QueryTypeSelect,
								Columns: drivers.QueryCols{
									{Name: "code", DBName: "code", TypeName: "[]byte", Nullable: &nullable},
									{Name: "children.id", DBName: "children.id", TypeName: "int32", Nullable: &canBeNull},
								},
								Mods: stubMods{},
							},
							{
								Name: "GetNestedWidgets",
								SQL:  `SELECT 7 AS related_widgets, 42 AS "related_widgets.widget_id", 'hidden' AS "related_widgets.secret_token"`,
								Type: bob.QueryTypeSelect,
								Columns: drivers.QueryCols{
									{Name: "related_widgets", DBName: "related_widgets", TypeName: "int32", Nullable: &nullable},
									{Name: "related_widgets.widget_id", DBName: "related_widgets.widget_id", TypeName: "int32", Nullable: &nullable},
									{
										Name:     "related_widgets.secret_token",
										DBName:   "related_widgets.secret_token",
										TypeName: "string",
										Nullable: &nullable,
									},
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
		DisableBobcheck: testgen.SkipBobcheck,
		NoTests:         true,
		Tags:            []string{"json", "yaml"},
		TagIgnore:       []string{"secret_token"},
	}}
	if err := gen.Run[any, any, driver.IndexExtra](
		t.Context(), state, stubDriver{types: helpers.Types(), info: info}, outputPlugins...,
	); err != nil {
		t.Fatalf("gen.Run: %v", err)
	}

	if err := os.WriteFile(filepath.Join(queryDir, "tags_test.go"), queryTagsTest, 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.CommandContext(t.Context(), "go", "test", "./queries")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("testing generated query results: %v\n%s", err, out)
	}
}
