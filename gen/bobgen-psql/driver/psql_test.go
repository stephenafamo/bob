package driver

import (
	"bytes"
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	embeddedpostgres "github.com/fergusstrange/embedded-postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testutils "github.com/stephenafamo/bob/test/utils"
)

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestDriver(t *testing.T) {
	port, err := helpers.GetFreePort()
	if err != nil {
		t.Fatalf("could not get a free port: %v", err)
	}

	dbConfig := embeddedpostgres.
		DefaultConfig().
		RuntimePath(filepath.Join(os.TempDir(), "bobgeb_psql")).
		Port(uint32(port)).
		Logger(&bytes.Buffer{})
	dsn := dbConfig.GetConnectionURL() + "?sslmode=disable"

	postgres := embeddedpostgres.NewDatabase(dbConfig)
	if err := postgres.Start(); err != nil {
		t.Fatalf("starting embedded postgres: %v", err)
	}
	defer func() {
		if err := postgres.Stop(); err != nil {
			t.Fatalf("could not stop postgres on port %d: %v", port, err)
		}
	}()

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("could not connect to db: %v", err)
	}
	defer db.Close()

	fmt.Printf("migrating...")
	if err := helpers.Migrate(context.Background(), db, testfiles.PostgresSchema); err != nil {
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

			testutils.TestDriver(t, testutils.DriverTestConfig[any]{
				Root: out,
				GetDriver: func() drivers.Interface[any] {
					return New(tt.config)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
			})
		})
	}
}
