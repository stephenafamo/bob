package driver

import (
	_ "embed"
	"encoding/json"
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

//go:embed test_data_model.json
var testDatamodel []byte

var (
	dataModel           Datamodel
	flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")
)

type testCase struct {
	name           string
	provider       Provider
	goldenJson     string
	modelTemplates fs.FS
}

func TestPostgres(t *testing.T) {
	testDialect(t, testCase{
		name:       "psql",
		goldenJson: "prisma.psql_golden.json",
		provider: Provider{
			DriverName: "pgx",
			DriverPkg:  "github.com/jackc/pgx/v5/stdlib",
		},
		modelTemplates: gen.PSQLModelTemplates,
	})
}

func TestMySQL(t *testing.T) {
	testDialect(t, testCase{
		name:       "mysql",
		goldenJson: "prisma.mysql_golden.json",
		provider: Provider{
			DriverName: "mysql",
			DriverPkg:  "github.com/go-sql-driver/mysql",
		},
		modelTemplates: gen.MySQLModelTemplates,
	})
}

func TestSQLite(t *testing.T) {
	testDialect(t, testCase{
		name:       "sqlite",
		goldenJson: "prisma.sqlite_golden.json",
		provider: Provider{
			DriverName: "sqlite",
			DriverPkg:  "modernc.org/sqlite",
		},
		modelTemplates: gen.SQLiteModelTemplates,
	})
}

func init() {
	err := json.Unmarshal(testDatamodel, &dataModel)
	if err != nil {
		panic(fmt.Sprintf("could not decode test_data_model.json: %v", err))
	}
}

func testDialect(t *testing.T, tt testCase) {
	t.Helper()
	out, err := os.MkdirTemp("", fmt.Sprintf("bobgen_prisma_%s_", tt.name))
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

	testgen.TestDriver(t, testgen.DriverTestConfig[Extra]{
		Root: out,
		GetDriver: func() drivers.Interface[Extra] {
			return New(Config{}, tt.name, tt.provider, dataModel)
		},
		GoldenFile:      tt.goldenJson,
		OverwriteGolden: *flagOverwriteGolden,
		Templates: &helpers.Templates{
			Models: append([]fs.FS{gen.PrismaModelTemplates}, tt.modelTemplates),
		},
	})
}
