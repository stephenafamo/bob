package driver

import (
	"embed"
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testutils "github.com/stephenafamo/bob/test_utils"
)

//go:embed test_schema
var testSchema embed.FS

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestDriver(t *testing.T) {
	psqlSchemas, _ := fs.Sub(testSchema, "test_schema/psql")
	mysqlSchemas, _ := fs.Sub(testSchema, "test_schema/mysql")
	sqliteSchemas, _ := fs.Sub(testSchema, "test_schema/sqlite")
	tests := []struct {
		name           string
		config         Config
		goldenJson     string
		schema         fs.FS
		modelTemplates fs.FS
	}{
		{
			name: "psql",
			config: Config{
				Dialect: "psql",
			},
			schema:     psqlSchemas,
			goldenJson: "atlas.psql_golden.json",
		},
		{
			name: "mysql",
			config: Config{
				Dialect: "mysql",
			},
			schema:         mysqlSchemas,
			goldenJson:     "atlas.mysql_golden.json",
			modelTemplates: gen.MySQLModelTemplates,
		},
		{
			name: "sqlite",
			config: Config{
				Dialect: "sqlite",
			},
			schema:         sqliteSchemas,
			goldenJson:     "atlas.sqlite_golden.json",
			modelTemplates: gen.SQLiteModelTemplates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := os.MkdirTemp("", fmt.Sprintf("bobgen_atlas_%s_", tt.name))
			if err != nil {
				t.Fatalf("unable to create tempdir: %s", err)
			}

			// Defer cleanup of the tmp folder
			defer func() {
				if t.Failed() {
					t.Log("template test output:", out)
					return
				}
				os.RemoveAll(out)
			}()

			testutils.TestDriver(t, testutils.DriverTestConfig[any]{
				Root: out,
				GetDriver: func() drivers.Interface[any] {
					return New(tt.config, tt.schema)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{tt.modelTemplates}},
			})
		})
	}
}
