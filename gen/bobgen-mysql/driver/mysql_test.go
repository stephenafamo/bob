package driver

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/go-sql-driver/mysql"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
	"github.com/testcontainers/testcontainers-go"
	mysqltest "github.com/testcontainers/testcontainers-go/modules/mysql"
)

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestDriver(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	mysqlContainer, err := mysqltest.Run(context.Background(),
		"mysql:8.0.35",
		mysqltest.WithDatabase("bobgen"),
		mysqltest.WithUsername("root"),
		mysqltest.WithPassword("password"),
	)
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(mysqlContainer); err != nil {
			fmt.Printf("failed to terminate MySQL container: %v\n", err)
		}
	})
	if err != nil {
		t.Fatalf("failed to start container: %v", err)
	}

	dsn, err := mysqlContainer.ConnectionString(ctx, "tls=skip-verify", "multiStatements=true", "parseTime=true")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("could not parse dsn: %v", err)
	}

	if !config.MultiStatements {
		t.Fatalf("multi statements MUST be turned on")
	}

	os.Setenv("MYSQL_TEST_DSN", dsn)
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("could not connect to db: %v", err)
	}

	fmt.Printf("migrating...")
	if err := helpers.Migrate(context.Background(), db, testfiles.MySQLSchema, "mysql/*.sql"); err != nil {
		t.Fatal(err)
	}
	fmt.Printf(" DONE\n")

	tests := []struct {
		name       string
		only       map[string][]string
		except     map[string][]string
		goldenJson string
	}{
		{
			name:       "default",
			goldenJson: "mysql.golden.json",
		},
		{
			name: "include tables",
			only: map[string][]string{
				"foo_bar": nil,
				"foo_baz": nil,
			},
			goldenJson: "include-tables.golden.json",
		},
		{
			name: "exclude tables",
			except: map[string][]string{
				"foo_bar": nil,
				"foo_baz": nil,
				"*":       {"secret_col"},
			},
			goldenJson: "exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables",
			only: map[string][]string{
				"foo_bar": nil,
				"foo_baz": nil,
			},
			except: map[string][]string{
				"foo_bar": nil,
				"bar_baz": nil,
			},
			goldenJson: "include-exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables regex",
			only: map[string][]string{
				"/^foo/": nil,
				"/^bar/": nil,
			},
			except: map[string][]string{
				"/bar$/": nil,
				"/baz$/": nil,
			},
			goldenJson: "include-exclude-tables-regex.golden.json",
		},
		{
			name: "include + exclude tables mixed",
			only: map[string][]string{
				"/^foo/":  nil,
				"bar_baz": nil,
				"bar_qux": nil,
			},
			except: map[string][]string{
				"/bar$/":  nil,
				"foo_baz": nil,
				"foo_qux": nil,
			},
			goldenJson: "include-exclude-tables-mixed.golden.json",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := Config{
				Config: helpers.Config{
					Dsn:    dsn,
					Only:   tt.only,
					Except: tt.except,
				},
			}

			if i > 0 {
				testgen.TestAssemble(t, testgen.AssembleTestConfig[any, any, any]{
					GetDriver: func() drivers.Interface[any, any, any] {
						return New(testConfig)
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
			t.Cleanup(func() {
				if t.Failed() {
					t.Log("template test output:", out)
					return
				}
				os.RemoveAll(out)
			})

			testgen.TestDriver(t, testgen.DriverTestConfig[any, any, any]{
				Root: out,
				GetDriver: func() drivers.Interface[any, any, any] {
					return New(testConfig)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
			})
		})
	}
}
