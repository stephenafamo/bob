package driver

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

//go:embed testdb.sql
var testDB string

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestAssemble(t *testing.T) {
	ctx := context.Background()

	config := Config{
		DSN:    "./test.db",
		Attach: map[string]string{"1": "./test1.db"},
	}

	err := os.Remove(config.DSN) // delete the old DB
	if err != nil {
		t.Fatalf("could not delete existing db: %v", err)
	}
	db, err := sql.Open("sqlite", config.DSN)
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	for schema, conn := range config.Attach {
		err := os.Remove(conn) // delete the old DB
		if err != nil {
			t.Fatalf("could not delete existing db: %v", err)
		}

		_, err = db.ExecContext(ctx, fmt.Sprintf("attach database '%s' as %q", conn, schema))
		if err != nil {
			t.Fatalf("could not attach %q: %v", conn, err)
		}
	}

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
			name:       "default",
			config:     config,
			goldenJson: "sqlite.golden.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Driver{config: tt.config}
			info, err := p.Assemble(context.Background())
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
