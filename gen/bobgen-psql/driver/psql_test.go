package driver

import (
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	testutils "github.com/stephenafamo/bob/test_utils"
)

//go:embed testdatabase.sql
var testDB string

var (
	flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

	dsn = os.Getenv("PSQL_TEST_DSN")
)

func TestDriver(t *testing.T) {
	if dsn == "" {
		t.Fatalf("No environment variable PSQL_TEST_DSN")
	}
	// somehow create the DB
	config, err := pgx.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("could not parse dsn: %v", err)
	}

	if !strings.Contains(config.Database, "droppable") {
		t.Fatalf("database name %q must contain %q to ensure that data is not lost", config.Database, "droppable")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("could not connect to db: %v", err)
	}
	defer db.Close()

	fmt.Printf("cleaning tables...")
	_, err = db.Exec(`DO $$ DECLARE
    r RECORD;
		BEGIN
			FOR r IN (SELECT tablename FROM pg_tables WHERE schemaname = current_schema()) LOOP
				EXECUTE 'DROP TABLE IF EXISTS ' || quote_ident(r.tablename) || ' CASCADE';
			END LOOP;
		END $$;`)
	if err != nil {
		t.Fatalf("could not connect drop all tables: %v", err)
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
				Dsn:     dsn,
				Schemas: []string{"public", "other", "shared"},
			},
			goldenJson: "psql.golden.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := os.MkdirTemp("", "bobgen_psql_")
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

			modelsFolder := filepath.Join(out, "models")
			err = os.Mkdir(modelsFolder, os.ModePerm)
			if err != nil {
				t.Fatalf("unable to create models folder: %s", err)
			}

			tt.config.Output = modelsFolder
			testutils.TestDriver(t, testutils.DriverTestConfig[any]{
				Root:            out,
				Driver:          New(tt.config),
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
			})
		})
	}
}
