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
	testgen "github.com/stephenafamo/bob/test/gen"
)

//go:embed test_schema
var testSchema embed.FS

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

type testCase struct {
	name           string
	config         Config
	goldenJson     string
	schema         fs.FS
	modelTemplates fs.FS
}

func TestPostgres(t *testing.T) {
	psqlSchemas, _ := fs.Sub(testSchema, "test_schema/psql")
	psqlCase := testCase{
		name: "psql",
		config: Config{
			Dialect: "psql",
		},
		schema:         psqlSchemas,
		goldenJson:     "atlas.psql_golden.json",
		modelTemplates: gen.PSQLModelTemplates,
	}
	testDialect(t, psqlCase)
}

func TestMySQL(t *testing.T) {
	mysqlSchemas, _ := fs.Sub(testSchema, "test_schema/mysql")
	mysqlCase := testCase{
		name: "mysql",
		config: Config{
			Dialect: "mysql",
		},
		schema:         mysqlSchemas,
		goldenJson:     "atlas.mysql_golden.json",
		modelTemplates: gen.MySQLModelTemplates,
	}
	testDialect(t, mysqlCase)
}

func TestSQLite(t *testing.T) {
	sqliteSchemas, _ := fs.Sub(testSchema, "test_schema/sqlite")
	sqliteCase := testCase{
		name: "sqlite",
		config: Config{
			Dialect: "sqlite",
		},
		schema:         sqliteSchemas,
		goldenJson:     "atlas.sqlite_golden.json",
		modelTemplates: gen.SQLiteModelTemplates,
	}
	testDialect(t, sqliteCase)
}

func testDialect(t *testing.T, tt testCase) {
	t.Helper()
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

	testgen.TestDriver(t, testgen.DriverTestConfig[any]{
		Root: out,
		GetDriver: func() drivers.Interface[any] {
			return New(tt.config, tt.schema)
		},
		GoldenFile:      tt.goldenJson,
		OverwriteGolden: *flagOverwriteGolden,
		Templates:       &helpers.Templates{Models: []fs.FS{tt.modelTemplates}},
	})
}
