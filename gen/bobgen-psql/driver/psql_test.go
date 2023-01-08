package driver

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

//go:embed testdatabase.sql
var testDB string

var (
	flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

	dsn = os.Getenv("DRIVER_TEST_DSN")
)

func TestAssemble(t *testing.T) {
	if dsn == "" {
		t.Fatalf("No environment variable DRIVER_TEST_DSN")
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
	for _, statement := range strings.Split(testDB, ";") {
		_, err = db.Exec(statement)
		if err != nil {
			t.Fatal(err)
		}
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
			p := &Driver{config: tt.config}
			info, err := p.Assemble()
			if err != nil {
				t.Fatal(err)
			}

			sort.Slice(info.Tables, func(i, j int) bool {
				return info.Tables[i].Key < info.Tables[j].Key
			})

			got, err := json.MarshalIndent(info, "", "\t")
			if err != nil {
				t.Fatal(err)
			}

			if *flagOverwriteGolden {
				if err = os.WriteFile(tt.goldenJson, got, 0o664); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(tt.goldenJson)
			if err != nil {
				t.Fatal(err)
			}

			// if diff := cmp.Diff(exp, spp); diff != "" {
			// t.Fatal(diff)
			// }
			require.JSONEq(t, string(want), string(got))
		})
	}
}
