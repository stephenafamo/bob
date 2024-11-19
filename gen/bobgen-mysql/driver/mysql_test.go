package driver

import (
	"context"
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
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
)

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
	if err := helpers.Migrate(context.Background(), db, testfiles.MySQLSchema); err != nil {
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
		{
			name: "include tables",
			config: Config{
				Dsn: dsn,
				Only: map[string][]string{
					"foo_bar": nil,
					"foo_baz": nil,
				},
			},
			goldenJson: "include-tables.golden.json",
		},
		{
			name: "exclude tables",
			config: Config{
				Dsn: dsn,
				Except: map[string][]string{
					"foo_bar": nil,
					"foo_baz": nil,
					"*":       {"secret_col"},
				},
			},
			goldenJson: "exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables",
			config: Config{
				Dsn: dsn,
				Only: map[string][]string{
					"foo_bar": nil,
					"foo_baz": nil,
				},
				Except: map[string][]string{
					"foo_bar": nil,
					"bar_baz": nil,
				},
			},
			goldenJson: "include-exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables regex",
			config: Config{
				Dsn: dsn,
				Only: map[string][]string{
					"/^foo/": nil,
					"/^bar/": nil,
				},
				Except: map[string][]string{
					"/bar$/": nil,
					"/baz$/": nil,
				},
			},
			goldenJson: "include-exclude-tables-regex.golden.json",
		},
		{
			name: "include + exclude tables mixed",
			config: Config{
				Dsn: dsn,
				Only: map[string][]string{
					"/^foo/":  nil,
					"bar_baz": nil,
					"bar_qux": nil,
				},
				Except: map[string][]string{
					"/bar$/":  nil,
					"foo_baz": nil,
					"foo_qux": nil,
				},
			},
			goldenJson: "include-exclude-tables-mixed.golden.json",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if i > 0 {
				testgen.TestAssemble(t, testgen.AssembleTestConfig[any, any, any]{
					GetDriver: func() drivers.Interface[any, any, any] {
						return New(tt.config)
					},
					GoldenFile:      tt.goldenJson,
					OverwriteGolden: *flagOverwriteGolden,
					Templates:       &helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
				})
				return
			}
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

			testgen.TestDriver(t, testgen.DriverTestConfig[any, any, any]{
				Root: out,
				GetDriver: func() drivers.Interface[any, any, any] {
					return New(tt.config)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
			})
		})
	}
}
