package driver

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testutils "github.com/stephenafamo/bob/test_utils"
	_ "modernc.org/sqlite"
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

//go:embed testdb.sql
var testDB string

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

	for schema, conn := range config.Attach {
		_, err = db.ExecContext(ctx, fmt.Sprintf("attach database '%s' as %q", conn, schema))
		if err != nil {
			t.Fatalf("could not attach %q: %v", conn, err)
		}
	}

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
			name:       "default",
			config:     config,
			goldenJson: "sqlite.golden.json",
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

			testutils.TestDriver(t, testutils.DriverTestConfig[any]{
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
