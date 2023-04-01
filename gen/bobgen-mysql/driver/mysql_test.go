package driver

import (
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testutils "github.com/stephenafamo/bob/test_utils"
)

//go:embed testdatabase.sql
var testDB string

var (
	flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

	dsn = os.Getenv("MYSQL_DRIVER_TEST_DSN")
)

func TestDriver(t *testing.T) {
	if dsn == "" {
		t.Fatalf("No environment variable MYSQL_DRIVER_TEST_DSN")
	}
	// somehow create the DB
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("could not parse dsn: %v", err)
	}

	if !strings.Contains(config.DBName, "droppable") {
		t.Fatalf("database name %q must contain %q to ensure that data is not lost", config.DBName, "droppable")
	}

	if !config.MultiStatements {
		t.Fatalf("multi statements MUST be turned on")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("could not connect to db: %v", err)
	}

	fmt.Printf("dropping database...")
	_, err = db.Exec(fmt.Sprintf(`DROP DATABASE %s;`, config.DBName))
	if err != nil {
		t.Fatalf("could not drop database: %v", err)
	}
	fmt.Printf(" DONE\n")

	fmt.Printf("creating database...")
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE %s;`, config.DBName))
	if err != nil {
		t.Fatalf("could not recreate database: %v", err)
	}
	fmt.Printf(" DONE\n")

	fmt.Printf("selecting database...")
	_, err = db.Exec(fmt.Sprintf(`USE %s;`, config.DBName))
	if err != nil {
		t.Fatalf("could not select database: %v", err)
	}
	fmt.Printf(" DONE\n")

	fmt.Printf("migrating...")
	_, err = db.Exec(testDB)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf(" DONE\n")

	tests := []struct {
		name       string
		config     Config
		goldenJson string
	}{
		{
			name: "default",
			config: Config{
				Dsn: dsn,
			},
			goldenJson: "mysql.golden.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := os.MkdirTemp("", "bobgen_mysql_")
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
					return New(tt.config)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
			})
		})
	}
}
