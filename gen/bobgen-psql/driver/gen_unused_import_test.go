package driver_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	driver "github.com/stephenafamo/bob/gen/bobgen-psql/driver"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/plugins"
)

type stubDriver struct {
	types drivers.Types
	info  *drivers.DBInfo[any, any, driver.IndexExtra]
}

func (s stubDriver) Dialect() string      { return "psql" }
func (s stubDriver) Types() drivers.Types { return s.types }
func (s stubDriver) Assemble(context.Context) (*drivers.DBInfo[any, any, driver.IndexExtra], error) {
	return s.info, nil
}

func col(name, dbType, goType string) drivers.Column {
	return drivers.Column{Name: name, DBType: dbType, Type: goType}
}

// Mirrors issue #730: a table with no UUID columns must not import the uuid
// package in its generated model file, even when uuid_pkg is configured and
// other tables use UUIDs.
func TestScalarTableModelHasNoUUIDImport(t *testing.T) {
	types := helpers.Types()
	types.Register("uuid.UUID", drivers.Type{
		Imports:    []string{`"github.com/google/uuid"`},
		RandomExpr: `return uuid.New()`,
	})

	info := &drivers.DBInfo[any, any, driver.IndexExtra]{
		Driver: "github.com/jackc/pgx/v5/stdlib",
		Tables: drivers.Tables[any, driver.IndexExtra]{
			{
				Key:  "sample_table",
				Name: "sample_table",
				Columns: []drivers.Column{
					col("id", "integer", "int32"),
					col("parent_id", "integer", "int32"),
					col("label_id", "integer", "int32"),
					col("enabled", "boolean", "bool"),
				},
				Constraints: drivers.Constraints[any]{
					Primary: &drivers.Constraint[any]{Name: "sample_table_pkey", Columns: []string{"id"}},
				},
			},
			{
				Key:  "tags",
				Name: "tags",
				Columns: []drivers.Column{
					col("id", "uuid", "uuid.UUID"),
					col("name", "text", "string"),
				},
				Constraints: drivers.Constraints[any]{
					Primary: &drivers.Constraint[any]{Name: "tags_pkey", Columns: []string{"id"}},
				},
			},
			{
				Key:  "sample_tags",
				Name: "sample_tags",
				Columns: []drivers.Column{
					col("sample_id", "integer", "int32"),
					col("tag_id", "uuid", "uuid.UUID"),
				},
				Constraints: drivers.Constraints[any]{
					Primary: &drivers.Constraint[any]{Name: "sample_tags_pkey", Columns: []string{"sample_id", "tag_id"}},
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Name: "sample_tags_sample_id_fkey", Columns: []string{"sample_id"}},
							ForeignTable:   "sample_table",
							ForeignColumns: []string{"id"},
						},
						{
							Constraint:     drivers.Constraint[any]{Name: "sample_tags_tag_id_fkey", Columns: []string{"tag_id"}},
							ForeignTable:   "tags",
							ForeignColumns: []string{"id"},
						},
					},
				},
			},
		},
	}

	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	gomod := "module scratch.local/repro\n\ngo 1.24\n"
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	outputPlugins := plugins.Setup[any, any, driver.IndexExtra](
		plugins.PresetAll, gen.PSQLTemplates,
	)

	state := &gen.State[any]{Config: gen.Config[any]{}}
	if err := gen.Run[any, any, driver.IndexExtra](
		context.Background(), state, stubDriver{types: types, info: info}, outputPlugins...,
	); err != nil {
		t.Fatalf("gen.Run: %v", err)
	}

	var checked int
	err = filepath.WalkDir(tmp, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
			return err
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(tmp, path)
		if strings.Contains(rel, "sample_table") {
			checked++
			if strings.Contains(string(b), "github.com/google/uuid") &&
				!strings.Contains(string(b), "uuid.") {
				t.Errorf("%s imports github.com/google/uuid but never uses it", rel)
			}
			t.Logf("checked %s (%d bytes)", rel, len(b))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked == 0 {
		t.Fatal("no sample_table files were generated/checked")
	}
}
