package gen_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/plugins"
	"github.com/stephenafamo/bob/internal"
)

type fkStubDriver struct {
	types drivers.Types
	info  *drivers.DBInfo[any, any, any]
}

func (s fkStubDriver) Dialect() string      { return "psql" }
func (s fkStubDriver) Types() drivers.Types { return s.types }
func (s fkStubDriver) Assemble(context.Context) (*drivers.DBInfo[any, any, any], error) {
	return s.info, nil
}

func TestRun_ForeignKeyTypeInferredFromReferencedColumn(t *testing.T) {
	types := helpers.Types()
	types.Register("uuid.UUID", drivers.Type{
		Imports:    []string{`"github.com/google/uuid"`},
		RandomExpr: `return uuid.New()`,
	})

	info := &drivers.DBInfo[any, any, any]{
		Driver: "github.com/jackc/pgx/v5/stdlib",
		Tables: drivers.Tables[any, any]{
			{
				Key:  "tax",
				Name: "tax",
				Columns: []drivers.Column{
					{Name: "id", DBType: "varchar(255)", Type: "string"},
					{Name: "cohort_id", DBType: "bigint", Type: "int64"},
				},
				Constraints: drivers.Constraints[any]{
					Primary: &drivers.Constraint[any]{Name: "tax_pkey", Columns: []string{"id"}},
					Uniques: []drivers.Constraint[any]{
						{Name: "tax_cohort_id_id_key", Columns: []string{"cohort_id", "id"}},
					},
				},
			},
			{
				Key:  "user_tax",
				Name: "user_tax",
				Columns: []drivers.Column{
					{Name: "id", DBType: "varchar(255)", Type: "string"},
					{Name: "cohort_id", DBType: "bigint", Type: "int64"},
					{Name: "tax_id", DBType: "text", Type: "string", Nullable: true},
				},
				Constraints: drivers.Constraints[any]{
					Primary: &drivers.Constraint[any]{Name: "user_tax_pkey", Columns: []string{"id"}},
					Foreign: []drivers.ForeignKey[any]{
						{
							Constraint:     drivers.Constraint[any]{Name: "user_tax_cohort_id_tax_id_fkey", Columns: []string{"cohort_id", "tax_id"}},
							ForeignTable:   "tax",
							ForeignColumns: []string{"cohort_id", "id"},
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

	outputPlugins := plugins.Setup[any, any, any](
		plugins.PresetAll, gen.BaseTemplates,
	)

	state := &gen.State[any]{
		Config: gen.Config[any]{
			Replacements: []gen.Replace{
				{
					Tables: []string{"tax"},
					Match: gen.ColumnFilter{
						Name: internal.Pointer("id"),
					},
					Replace: "uuid.UUID",
				},
			},
		},
	}

	if err := gen.Run[any, any, any](
		context.Background(), state, fkStubDriver{types: types, info: info}, outputPlugins...,
	); err != nil {
		t.Fatalf("gen.Run: %v", err)
	}

	userTaxModel, err := os.ReadFile(filepath.Join(tmp, "models", "user_tax.bob.go"))
	if err != nil {
		t.Fatalf("read models/user_tax.bob.go: %v", err)
	}
	content := string(userTaxModel)
	if !strings.Contains(content, `"github.com/google/uuid"`) {
		t.Errorf("models/user_tax.bob.go missing uuid import:\n%s", content)
	}
	if !strings.Contains(content, "null.Val[uuid.UUID]") {
		t.Errorf("models/user_tax.bob.go missing null.Val[uuid.UUID] for inferred FK column:\n%s", content)
	}
}
