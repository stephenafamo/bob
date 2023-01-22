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

	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
)

//go:embed testdatabase.sql
var testDB string

var (
	flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

	dsn = os.Getenv("MYSQL_TEST_DSN")
)

func TestAssemble(t *testing.T) {
	if dsn == "" {
		t.Fatalf("No environment variable MYSQL_TEST_DSN")
	}
	// somehow create the DB
	config, err := mysql.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("could not parse dsn: %v", err)
	}

	if !strings.Contains(config.DBName, "droppable") {
		t.Fatalf("database name %q must contain %q to ensure that data is not lost", config.DBName, "droppable")
	}

	if !config.MultiStatements {
		t.Fatalf("multi statements MUST be turned on")
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatalf("could not connect to db: %v", err)
	}

	fmt.Printf("dropping database...")
	_, err = db.Exec(fmt.Sprintf(`DROP DATABASE %s;`, config.DBName))
	if err != nil {
		t.Fatalf("could not drop database: %v", err)
	}
	fmt.Printf(" DONE\n")

	fmt.Printf("creating database...")
	_, err = db.Exec(fmt.Sprintf(`CREATE DATABASE %s;`, config.DBName))
	if err != nil {
		t.Fatalf("could not recreate database: %v", err)
	}
	fmt.Printf(" DONE\n")

	fmt.Printf("selecting database...")
	_, err = db.Exec(fmt.Sprintf(`USE %s;`, config.DBName))
	if err != nil {
		t.Fatalf("could not select database: %v", err)
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
				Dsn: dsn,
			},
			goldenJson: "mysql.golden.json",
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

			require.JSONEq(t, string(want), string(got))
		})
	}
}
