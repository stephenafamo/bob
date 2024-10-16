package driver

import (
	"context"
	"database/sql"
	sqlDriver "database/sql/driver"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
	"modernc.org/sqlite"
)

func cleanup(t *testing.T, config Config) {
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

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestAssemble(t *testing.T) {
	ctx := context.Background()

	config := Config{
		DSN:    "./test.db",
		Attach: map[string]string{"one": "./test1.db"},
	}

	cleanup(t, config)
	t.Cleanup(func() { cleanup(t, config) })

	db, err := sql.Open("sqlite", config.DSN)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err = registerRegexpFunction(); err != nil {
		t.Fatal(err)
	}

	for schema, conn := range config.Attach {
		_, err = db.ExecContext(ctx, fmt.Sprintf("attach database '%s' as %q", conn, schema))
		if err != nil {
			t.Fatalf("could not attach %q: %v", conn, err)
		}
	}

	fmt.Printf("migrating...")
	if err := helpers.Migrate(context.Background(), db, testfiles.SQLiteSchema); err != nil {
		t.Fatal(err)
	}
	fmt.Printf(" DONE\n")

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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			testgen.TestDriver(t, testgen.DriverTestConfig[any]{
				Root: out,
				GetDriver: func() drivers.Interface[any] {
					return New(tt.config)
				},
				GoldenFile:      tt.goldenJson,
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
