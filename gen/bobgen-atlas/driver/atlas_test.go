package driver

import (
	"context"
	"embed"
	_ "embed"
	"encoding/json"
	"flag"
	"io/fs"
	"os"
	"sort"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"
)

//go:embed test_schema
var testSchema embed.FS

var flagOverwriteGolden = flag.Bool("overwrite-golden", false, "Overwrite the golden file with the current execution results")

func TestAssemble(t *testing.T) {
	psqlSchemas, _ := fs.Sub(testSchema, "test_schema")
	tests := []struct {
		name       string
		config     Config
		fs         fs.FS
		goldenJson string
	}{
		{
			name: "default",
			config: Config{
				Dialect: "psql",
			},
			fs:         psqlSchemas,
			goldenJson: "atlas.golden.json",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Driver{config: tt.config, fs: tt.fs}
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
