package parser

import (
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	pg "github.com/pganalyze/pg_query_go/v6"
	pgparse "github.com/wasilibs/go-pgquery"
)

var subExprRe = regexp.MustCompile(`q\.(\w+)\(EXPR\.subExpr\((\d+), (\d+)\)\)`)

func TestCaseExprEnd(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  map[string]string
	}{
		{
			name: "update set case",
			input: `UPDATE item i
SET
    name = CASE
        WHEN $2 THEN $3
        ELSE i.name
    END
WHERE i.id = $1`,
			want: map[string]string{
				"AppendSet": `name = CASE
        WHEN $1 THEN $2
        ELSE i.name
    END`,
				"AppendWhere": `i.id = $3`,
			},
		},
		{
			name:  "update set nested case",
			input: `UPDATE item SET name = CASE WHEN $1 THEN CASE WHEN $2 THEN 'a' ELSE 'b' END END WHERE id = $3`,
			want: map[string]string{
				"AppendSet":   `name = CASE WHEN $1 THEN CASE WHEN $2 THEN 'a' ELSE 'b' END END`,
				"AppendWhere": `id = $3`,
			},
		},
		{
			name:  "update where case",
			input: `UPDATE item SET price = 1 WHERE CASE WHEN $1 THEN id = $2 ELSE false END`,
			want: map[string]string{
				"AppendSet":   `price = 1`,
				"AppendWhere": `CASE WHEN $1 THEN id = $2 ELSE false END`,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scanResult, err := pgparse.Scan(tc.input)
			if err != nil {
				t.Fatalf("scan: %v", err)
			}

			parseResult, err := pgparse.Parse(tc.input)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}

			w := walker{
				input:       tc.input,
				tokens:      scanResult.GetTokens(),
				mods:        &strings.Builder{},
				names:       make(map[position]string),
				nullability: make(map[position]nullable),
				groups:      make(map[argPos]struct{}),
				multiple:    make(map[[2]int]struct{}),
				atom:        &atomic.Int64{},
				paramIdxMap: make(map[int64]int64),
			}

			stmt := parseResult.Stmts[0].Stmt
			info := w.walk(stmt)
			w.modUpdateStatement(stmt.Node.(*pg.Node_UpdateStmt), info.children["UpdateStmt"])

			formatted, err := w.formattedQuery()
			if err != nil {
				t.Fatalf("formattedQuery: %v", err)
			}

			got := make(map[string]string)
			for _, m := range subExprRe.FindAllStringSubmatch(w.mods.String(), -1) {
				start, _ := strconv.Atoi(m[2])
				end, _ := strconv.Atoi(m[3])
				got[m[1]] = formatted[start:end]
			}

			for method, want := range tc.want {
				if got[method] != want {
					t.Errorf("%s: got %q, want %q", method, got[method], want)
				}
			}
		})
	}
}
