package driver

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
)

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestDriver(t *testing.T) {
	port, err := helpers.GetFreePort()
	if err != nil {
		t.Fatalf("could not get a free port: %v", err)
	}

	dbConfig := embeddedpostgres.
		DefaultConfig().
		RuntimePath(filepath.Join(os.TempDir(), "bobgen_psql")).
		Port(uint32(port)).
		Logger(&bytes.Buffer{})
	dsn := dbConfig.GetConnectionURL() + "?sslmode=disable"

	postgres := embeddedpostgres.NewDatabase(dbConfig)
	if err := postgres.Start(); err != nil {
		t.Fatalf("starting embedded postgres: %v", err)
	}
	t.Cleanup(func() {
		if err := postgres.Stop(); err != nil {
			t.Fatalf("could not stop postgres on port %d: %v", port, err)
		}
	})

	os.Setenv("PSQL_TEST_DSN", dsn)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("could not connect to db: %v", err)
	}
	defer db.Close()

	fmt.Printf("migrating...")
	if err := helpers.Migrate(context.Background(), db, testfiles.PostgresSchema, "psql/*.sql"); err != nil {
		t.Fatal(err)
	}
	fmt.Printf(" DONE\n")

	t.Run("driver", func(t *testing.T) { testPostgresDriver(t, dsn) })
	t.Run("assemble", func(t *testing.T) { testPostgresAssemble(t, dsn) })
}

func testPostgresDriver(t *testing.T, dsn string) {
	t.Helper()

	config := Config{
		Config: helpers.Config{
			Dsn:     dsn,
			Queries: []string{"./queries"},
		},
		Schemas: []string{"public", "other", "shared"},
	}

	tests := []struct {
		name   string
		driver string
	}{
		{
			name:   "pq",
			driver: "github.com/lib/pq",
		},
		// {
		// 	name:       "pgx-v5",
		// 	driver: "github.com/jackc/pgx/v5",
		// },
		{
			name:   "pgx-v5-std",
			driver: "github.com/jackc/pgx/v5/stdlib",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := os.MkdirTemp("", "bobgen_psql_")
			if err != nil {
				t.Fatalf("unable to create tempdir: %s", err)
			}

			t.Cleanup(func() {
				if t.Failed() {
					t.Log("template test output:", out)
					return
				}
				os.RemoveAll(out)
			})

			overwriteGolden := *flagOverwriteGolden
			if tt.driver != "" && tt.driver != defaultDriver {
				// If not using the default driver, we do not overwrite the golden file
				overwriteGolden = false
			}

			testConfig := config
			testConfig.Driver = tt.driver

			testgen.TestDriver(t, testgen.DriverTestConfig[any, any, IndexExtra]{
				Root:      out,
				Templates: &helpers.Templates{Models: []fs.FS{gen.PSQLModelTemplates}},
				GetDriver: func() drivers.Interface[any, any, IndexExtra] {
					return New(testConfig)
				},
				GoldenFile: "psql.golden.json",
				GoldenFileMod: func(b []byte) []byte {
					return []byte(strings.ReplaceAll(
						string(b), defaultDriver, tt.driver,
					))
				},
				OverwriteGolden: overwriteGolden,
			})
		})
	}
}

func testPostgresAssemble(t *testing.T, dsn string) {
	t.Helper()

	tests := []struct {
		name       string
		Only       map[string][]string
		Except     map[string][]string
		goldenJson string
	}{
		{
			name: "include tables",
			Only: map[string][]string{
				"foo_bar": nil,
				"foo_baz": nil,
			},
			goldenJson: "include-tables.golden.json",
		},
		{
			name: "exclude tables",
			Except: map[string][]string{
				"foo_bar": nil,
				"foo_baz": nil,
				"*":       {"secret_col"},
			},
			goldenJson: "exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables",
			Only: map[string][]string{
				"foo_bar": nil,
				"foo_baz": nil,
			},
			Except: map[string][]string{
				"foo_bar": nil,
				"bar_baz": nil,
			},
			goldenJson: "include-exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables regex",
			Only: map[string][]string{
				"/^foo/": nil,
				"/^bar/": nil,
			},
			Except: map[string][]string{
				"/bar$/": nil,
				"/baz$/": nil,
			},
			goldenJson: "include-exclude-tables-regex.golden.json",
		},
		{
			name: "include + exclude tables mixed",
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
			goldenJson: "include-exclude-tables-mixed.golden.json",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testConfig := Config{
				Config: helpers.Config{
					Dsn:    dsn,
					Only:   tt.Only,
					Except: tt.Except,
				},
			}

			if i > 0 {
				testgen.TestAssemble(t, testgen.AssembleTestConfig[any, any, IndexExtra]{
					Templates: &helpers.Templates{Models: []fs.FS{gen.PSQLModelTemplates}},
					GetDriver: func() drivers.Interface[any, any, IndexExtra] {
						return New(testConfig)
					},
					GoldenFile:      tt.goldenJson,
					OverwriteGolden: *flagOverwriteGolden,
				})
				return
			}

			out, err := os.MkdirTemp("", "bobgen_psql_")
			if err != nil {
				t.Fatalf("unable to create tempdir: %s", err)
			}

			t.Cleanup(func() {
				if t.Failed() {
					t.Log("template test output:", out)
					return
				}
				os.RemoveAll(out)
			})

			testgen.TestDriver(t, testgen.DriverTestConfig[any, any, IndexExtra]{
				Root:      out,
				Templates: &helpers.Templates{Models: []fs.FS{gen.PSQLModelTemplates}},
				GetDriver: func() drivers.Interface[any, any, IndexExtra] {
					return New(testConfig)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
			})
		})
	}
}
