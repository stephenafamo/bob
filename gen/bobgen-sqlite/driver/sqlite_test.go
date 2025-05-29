package driver

import (
	"context"
	"database/sql"
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func connect(t *testing.T, driver, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driver, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	return db
}

func migrate(t *testing.T, db *sql.DB, schema embed.FS, pattern string) {
	t.Helper()
	if err := helpers.Migrate(context.Background(), db, schema, pattern); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleLibSQL(t *testing.T) {
	ctx := context.Background()

	libsqlServer, err := testcontainers.Run(
		ctx,
		"ghcr.io/tursodatabase/libsql-server:c6e4e09",
		testcontainers.WithExposedPorts("7070:8080", "9000:8000"),
		testcontainers.WithWaitStrategy(wait.ForListeningPort("8080/tcp").WithStartupTimeout(time.Second*5)),
	)
	t.Cleanup(func() {
		if err := testcontainers.TerminateContainer(libsqlServer); err != nil {
			fmt.Printf("failed to terminate libsql container: %v\n", err)
		}
	})
	if err != nil {
		t.Fatalf("failed to start libsql container: %v", err)
	}

	config := Config{
		Config: helpers.Config{
			Dsn: "http://localhost:7070",
		},
	}

	os.Setenv("LIBSQL_TEST_DSN", config.Dsn)
	os.Setenv("BOB_SQLITE_ATTACH_QUERIES", strings.Join(config.AttachQueries(), ";"))

	db := connect(t, "libsql", config.Dsn)
	if err := attach(ctx, db, config); err != nil {
		t.Fatalf("attaching: %v", err)
	}

	fmt.Printf("migrating...")
	migrate(t, db, testfiles.LibSQLDefaultSchema, "libsql/default/*.sql")
	fmt.Printf(" DONE\n")

	out, err := os.MkdirTemp("", "bobgen_libsql_")
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
		Root: out,
		GetDriver: func() drivers.Interface[any, any, IndexExtra] {
			return New(config)
		},
		GoldenFile: "libsql.golden.json",
		GoldenFileMod: func(b []byte) []byte {
			return []byte(strings.ReplaceAll(
				string(b),
				defaultDriver,
				"github.com/tursodatabase/libsql-client-go/libsql",
			))
		},
		OverwriteGolden: *flagOverwriteGolden,
		Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
	})
}

func TestAssembleSQLite(t *testing.T) {
	ctx := context.Background()

	dir, err := os.MkdirTemp("", "bobgen_sqlite_*")
	if err != nil {
		log.Fatal(err)
	}
	t.Cleanup(func() {
		if t.Failed() {
			t.Log("db files directory:", dir)
			return
		}
		os.RemoveAll(dir)
	})

	mainDB, err := os.Create(filepath.Join(dir, "main.db"))
	if err != nil {
		t.Fatalf("unable to create main.db: %s", err)
	}

	oneDB, err := os.Create(filepath.Join(dir, "one.db"))
	if err != nil {
		t.Fatalf("unable to create one.db: %s", err)
	}

	config := Config{
		Config: helpers.Config{
			Dsn: "file:" + mainDB.Name() + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)",
		},
		Attach: map[string]string{"one": oneDB.Name()},
	}
	os.Setenv("SQLITE_TEST_DSN", config.Dsn)
	os.Setenv("BOB_SQLITE_ATTACH_QUERIES", strings.Join(config.AttachQueries(), ";"))

	db := connect(t, "sqlite", config.Dsn)
	defer db.Close()

	if err := attach(ctx, db, config); err != nil {
		t.Fatalf("attaching: %v", err)
	}

	fmt.Printf("migrating...")
	migrate(t, db, testfiles.SQLiteSchema, "sqlite/*.sql")
	fmt.Printf(" DONE\n")

	t.Run("driver", func(t *testing.T) { testSQLiteDriver(t, config) })
	t.Run("assemble", func(t *testing.T) { testSQLiteAssemble(t, config) })
}

func testSQLiteDriver(t *testing.T, config Config) {
	t.Helper()

	tests := []struct {
		name   string
		driver string
	}{
		{
			name:   "mattn",
			driver: "github.com/mattn/go-sqlite3",
		},
		{
			name:   "modernc",
			driver: "modernc.org/sqlite",
		},
		{
			name:   "ncruces",
			driver: "github.com/ncruces/go-sqlite3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := os.MkdirTemp("", "bobgen_sqlite_")
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
				Root: out,
				GetDriver: func() drivers.Interface[any, any, IndexExtra] {
					return New(testConfig)
				},
				GoldenFile: "sqlite.golden.json",
				GoldenFileMod: func(b []byte) []byte {
					return []byte(strings.ReplaceAll(
						string(b), defaultDriver, tt.driver,
					))
				},
				OverwriteGolden: overwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
			})
		})
	}
}

func testSQLiteAssemble(t *testing.T, config Config) {
	t.Helper()

	tests := []struct {
		name       string
		only       map[string][]string
		except     map[string][]string
		goldenJson string
	}{
		{
			name: "include tables",
			only: map[string][]string{
				"foo_bar":     nil,
				"foo_baz":     nil,
				"one.foo_bar": nil,
				"one.foo_baz": nil,
			},
			goldenJson: "include-tables.golden.json",
		},
		{
			name: "exclude tables",
			except: map[string][]string{
				"foo_bar":     nil,
				"foo_baz":     nil,
				"one.foo_bar": nil,
				"one.foo_baz": nil,
				"*":           {"secret_col"},
			},
			goldenJson: "exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables",
			only: map[string][]string{
				"foo_bar":     nil,
				"foo_baz":     nil,
				"one.foo_bar": nil,
				"one.foo_baz": nil,
			},
			except: map[string][]string{
				"foo_bar":     nil,
				"bar_baz":     nil,
				"one.foo_bar": nil,
				"one.bar_baz": nil,
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
				"/^foo/":      nil,
				"bar_baz":     nil,
				"bar_qux":     nil,
				"one.bar_baz": nil,
				"one.bar_qux": nil,
			},
			except: map[string][]string{
				"/bar$/":      nil,
				"foo_baz":     nil,
				"foo_qux":     nil,
				"one.foo_baz": nil,
				"one.foo_qux": nil,
			},
			goldenJson: "include-exclude-tables-mixed.golden.json",
		},
	}

	for _, tt := range tests {
		testConfig := config
		testConfig.Only = tt.only
		testConfig.Except = tt.except

		overwriteGolden := *flagOverwriteGolden
		if testConfig.Driver != "" && testConfig.Driver != defaultDriver {
			// If not using the default driver, we do not overwrite the golden file
			overwriteGolden = false
		}

		t.Run(tt.name, func(t *testing.T) {
			testgen.TestAssemble(t, testgen.AssembleTestConfig[any, any, IndexExtra]{
				GetDriver: func() drivers.Interface[any, any, IndexExtra] {
					return New(testConfig)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: overwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
			})
		})
	}
}
