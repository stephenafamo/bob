package driver

import (
	"context"
	"database/sql"
	_ "embed"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

const columnOrderNameStarQuery = `-- GetUsersWithVideos
SELECT
  u.*,
  v.id AS video_id
FROM users AS u
INNER JOIN videos AS v ON v.user_id = u.id;`

func TestDriver(t *testing.T) {
	postgresContainer, err := postgres.Run(
		t.Context(), "pgvector/pgvector:0.8.0-pg16",
		postgres.BasicWaitStrategies(),
		testcontainers.WithLogger(log.New(io.Discard, "", log.LstdFlags)),
	)
	if err != nil {
		fmt.Printf("could not start postgres container: %v\n", err)
		return
	}
	defer func() {
		if err := testcontainers.TerminateContainer(postgresContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	dsn, err := postgresContainer.ConnectionString(t.Context(), "sslmode=disable")
	if err != nil {
		fmt.Printf("could not get connection string: %v\n", err)
		return
	}

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
	t.Run("column_order_name_star_types", func(t *testing.T) { testPostgresColumnOrderStarTypes(t, dsn) })
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
				Templates: gen.PSQLTemplates,
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
				Dialect:         "psql",
			})
		})
	}
}

func testPostgresColumnOrderStarTypes(t *testing.T, dsn string) {
	t.Helper()

	queryDir := t.TempDir()
	if err := os.WriteFile(queryDir+"/joined.sql", []byte(columnOrderNameStarQuery), 0o600); err != nil {
		t.Fatalf("write query file: %v", err)
	}

	drv := New(Config{
		Config: helpers.Config{
			Dsn:         dsn,
			Queries:     []string{queryDir},
			ColumnOrder: "name",
		},
		Schemas: []string{"public"},
	})

	info, err := drv.Assemble(context.Background())
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}

	var query drivers.Query
	found := false
	for _, folder := range info.QueryFolders {
		for _, file := range folder.Files {
			for _, q := range file.Queries {
				if q.Name == "GetUsersWithVideos" {
					query = q
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("query GetUsersWithVideos not found in assembled output")
	}

	colTypes := make(map[string]string, len(query.Columns))
	for _, col := range query.Columns {
		colTypes[col.Name] = col.TypeName
	}

	wantTypes := map[string]string{
		"id":              "int32",
		"email_validated": "bool",
		"primary_email":   "string",
		"parent_id":       "int32",
		"party_id":        "int32",
		"referrer":        "int32",
		"video_id":        "int32",
	}
	for name, want := range wantTypes {
		if got := colTypes[name]; got != want {
			t.Errorf("column %s: type = %q, want %q", name, got, want)
		}
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
					Templates: gen.PSQLTemplates,
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
				Templates: gen.PSQLTemplates,
				GetDriver: func() drivers.Interface[any, any, IndexExtra] {
					return New(testConfig)
				},
				GoldenFile:      tt.goldenJson,
				OverwriteGolden: *flagOverwriteGolden,
				Dialect:         "psql",
			})
		})
	}
}
