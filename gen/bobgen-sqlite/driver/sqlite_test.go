package driver

import (
	"context"
	"database/sql"
	sqlDriver "database/sql/driver"
	"embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
	"modernc.org/sqlite"
)

var libSQLAddress = os.Getenv("LIBSQL_TEST_SERVER")

func cleanupSQLite(t *testing.T, config Config) {
	t.Helper()

	fmt.Printf("cleaning...")
	err := os.Remove(config.DSN) // delete the old DB
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("could not delete existing db: %v", err)
	}

	for _, conn := range config.Attach {
		err := os.Remove(conn) // delete the old DB
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("could not delete existing db: %v", err)
		}
	}

	fmt.Printf(" DONE\n")
}

func cleanupLibSQL(t *testing.T, db *sql.DB) {
	t.Helper()

	fmt.Printf("cleaning...")

	// Find all tables
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%';")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	// Drop each table
	for rows.Next() {
		var tableName string
		if err = rows.Scan(&tableName); err != nil {
			t.Fatalf("could not delete existing db: %v", err)
		}
		_, err = db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS %q;", tableName))
		if err != nil {
			t.Fatalf("could not delete %q table: %v", tableName, err)
		}
	}

	// Find all tables
	viewRows, err := db.Query("SELECT name FROM sqlite_master WHERE type='view' AND name NOT LIKE 'sqlite_%';")
	if err != nil {
		t.Fatal(err)
	}
	defer viewRows.Close()

	// Drop each table
	for viewRows.Next() {
		var viewName string
		if err = viewRows.Scan(&viewName); err != nil {
			t.Fatalf("could not delete existing db: %v", err)
		}
		_, err = db.Exec(fmt.Sprintf("DROP VIEW IF EXISTS %q;", viewName))
		if err != nil {
			t.Fatalf("could not delete %q view: %v", viewName, err)
		}
	}

	fmt.Printf(" DONE\n")
}

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestAssembleSQLite(t *testing.T) {
	ctx := context.Background()

	config := Config{
		DSN:    "./test.db",
		Attach: map[string]string{"one": "./test1.db"},
	}

	cleanupSQLite(t, config)
	t.Cleanup(func() { cleanupSQLite(t, config) })

	db := connect(t, "sqlite", config.DSN)
	defer db.Close()

	if err := registerRegexpFunction(); err != nil {
		t.Fatal(err)
	}

	attach(t, ctx, db, config)

	fmt.Printf("migrating...")
	migrate(t, db, testfiles.SQLiteSchema, "sqlite/*.sql")
	fmt.Printf(" DONE\n")

	assemble(t, config, nil)
}

func TestAssembleLibSQL(t *testing.T) {
	ctx := context.Background()

	config := Config{
		DSN:    "ws://" + libSQLAddress,
		Attach: map[string]string{"one": "one"},
	}

	db := connect(t, "libsql", config.DSN)

	attach(t, ctx, db, config)

	dbHttpDefault := connect(t, "libsql", "http://"+libSQLAddress)
	dbHttpOne := connect(t, "libsql", "http://one."+libSQLAddress)

	cleanupLibSQL(t, dbHttpDefault)
	cleanupLibSQL(t, dbHttpOne)

	fmt.Printf("migrating...")
	migrate(t, dbHttpDefault, testfiles.LibSQLDefaultSchema, "libsql/default/*.sql")
	migrate(t, dbHttpOne, testfiles.LibSQLOneSchema, "libsql/one/*.sql")
	fmt.Printf(" DONE\n")

	t.Cleanup(func() {
		cleanupLibSQL(t, dbHttpDefault)
		cleanupLibSQL(t, dbHttpOne)
		dbHttpDefault.Close()
		dbHttpOne.Close()
	})

	assemble(t, config, func(b []byte) []byte {
		return []byte(strings.ReplaceAll(
			string(b),
			"modernc.org/sqlite",
			"github.com/tursodatabase/libsql-client-go/libsql",
		))
	})
}

func connect(t *testing.T, driverName, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	return db
}

func attach(t *testing.T, ctx context.Context, db *sql.DB, config Config) {
	t.Helper()
	for schema, conn := range config.Attach {
		if strings.HasPrefix(conn, "./") {
			conn = strconv.Quote(conn)
		}
		_, err := db.ExecContext(ctx, fmt.Sprintf("attach database %s as %s", conn, schema))
		if err != nil {
			t.Fatalf("could not attach %q: %v", conn, err)
		}
	}
}

func migrate(t *testing.T, db *sql.DB, schema embed.FS, pattern string) {
	t.Helper()
	if err := helpers.Migrate(context.Background(), db, schema, pattern); err != nil {
		t.Fatal(err)
	}
}

func assemble(t *testing.T, config Config, mod func([]byte) []byte) {
	t.Helper()
	tests := []struct {
		name       string
		config     Config
		goldenJson string
	}{
		{
			name:       "default",
			config:     config,
			goldenJson: "sqlite.golden.json",
		},
		{
			name: "include tables",
			config: Config{
				DSN:    config.DSN,
				Attach: config.Attach,
				Only: map[string][]string{
					"foo_bar":     nil,
					"foo_baz":     nil,
					"one.foo_bar": nil,
					"one.foo_baz": nil,
				},
			},
			goldenJson: "include-tables.golden.json",
		},
		{
			name: "exclude tables",
			config: Config{
				DSN:    config.DSN,
				Attach: config.Attach,
				Except: map[string][]string{
					"foo_bar":     nil,
					"foo_baz":     nil,
					"one.foo_bar": nil,
					"one.foo_baz": nil,
					"*":           {"secret_col"},
				},
			},
			goldenJson: "exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables",
			config: Config{
				DSN:    config.DSN,
				Attach: config.Attach,
				Only: map[string][]string{
					"foo_bar":     nil,
					"foo_baz":     nil,
					"one.foo_bar": nil,
					"one.foo_baz": nil,
				},
				Except: map[string][]string{
					"foo_bar":     nil,
					"bar_baz":     nil,
					"one.foo_bar": nil,
					"one.bar_baz": nil,
				},
			},
			goldenJson: "include-exclude-tables.golden.json",
		},
		{
			name: "include + exclude tables regex",
			config: Config{
				DSN:    config.DSN,
				Attach: config.Attach,
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
				DSN:    config.DSN,
				Attach: config.Attach,
				Only: map[string][]string{
					"/^foo/":      nil,
					"bar_baz":     nil,
					"bar_qux":     nil,
					"one.bar_baz": nil,
					"one.bar_qux": nil,
				},
				Except: map[string][]string{
					"/bar$/":      nil,
					"foo_baz":     nil,
					"foo_qux":     nil,
					"one.foo_baz": nil,
					"one.foo_qux": nil,
				},
			},
			goldenJson: "include-exclude-tables-mixed.golden.json",
		},
	}

	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if i > 0 {
				testgen.TestAssemble(t, testgen.AssembleTestConfig[any, any, IndexExtra]{
					GetDriver: func() drivers.Interface[any, any, IndexExtra] {
						return New(tt.config)
					},
					GoldenFile:      tt.goldenJson,
					GoldenFileMod:   mod,
					OverwriteGolden: *flagOverwriteGolden,
					Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
				})
				return
			}

			out, err := os.MkdirTemp("", "bobgen_sqlite_")
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

			testgen.TestDriver(t, testgen.DriverTestConfig[any, any, IndexExtra]{
				Root: out,
				GetDriver: func() drivers.Interface[any, any, IndexExtra] {
					return New(tt.config)
				},
				GoldenFile:      tt.goldenJson,
				GoldenFileMod:   mod,
				OverwriteGolden: *flagOverwriteGolden,
				Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
			})
		})
	}
}

func registerRegexpFunction() error {
	return sqlite.RegisterScalarFunction("regexp", 2, func(
		ctx *sqlite.FunctionContext,
		args []sqlDriver.Value,
	) (sqlDriver.Value, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("expected 2 arguments, got %d", len(args))
		}

		re, ok := args[0].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[0])
		}

		s, ok := args[1].(string)
		if !ok {
			return nil, fmt.Errorf("expected string, got %T", args[1])
		}

		match, err := regexp.MatchString(re, s)
		if err != nil {
			return nil, err
		}

		return match, nil
	})
}
