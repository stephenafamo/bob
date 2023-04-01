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
	testutils "github.com/stephenafamo/bob/test_utils"
)

//go:embed test_data_model.json
var testDatamodel []byte

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestAssemble(t *testing.T) {
	var dataModel Datamodel
	err := json.Unmarshal(testDatamodel, &dataModel)
	if err != nil {
		t.Fatalf("could not decode test_data_model.json: %v", err)
	}

	tests := []struct {
		name           string
		config         Config
		provider       Provider
		datamodel      Datamodel
		goldenJson     string
		modelTemplates fs.FS
	}{
		{
			name:       "psql",
			config:     Config{},
			datamodel:  dataModel,
			goldenJson: "prisma.psql_golden.json",
			provider: Provider{
				DriverName: "pgx",
				DriverPkg:  "github.com/jackc/pgx/v5/stdlib",
			},
		},
		{
			name:       "mysql",
			config:     Config{},
			datamodel:  dataModel,
			goldenJson: "prisma.mysql_golden.json",
			provider: Provider{
				DriverName: "mysql",
				DriverPkg:  "github.com/go-sql-driver/mysql",
			},
			modelTemplates: gen.MySQLModelTemplates,
		},
		{
			name:       "sqlite",
			config:     Config{},
			datamodel:  dataModel,
			goldenJson: "prisma.sqlite_golden.json",
			provider: Provider{
				DriverName: "sqlite",
				DriverPkg:  "modernc.org/sqlite",
			},
			modelTemplates: gen.SQLiteModelTemplates,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			testutils.TestDriver(t, testutils.DriverTestConfig[Extra]{
				Root: out,
				GetDriver: func() drivers.Interface[Extra] {
					return New(tt.config, tt.name, tt.provider, tt.datamodel)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
				Templates: &helpers.Templates{
					Models: append([]fs.FS{gen.PrismaModelTemplates}, tt.modelTemplates),
				},
			})
		})
	}
}
