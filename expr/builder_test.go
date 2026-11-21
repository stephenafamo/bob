package expr

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
)

type chain struct {
	Chain[chain, chain]
}

func (chain) New(exp bob.Expression) chain {
	var c chain
	c.Base = exp
	return c
}

// A custom query may render its expression SQL without parentheses.
type unwrappedQuery struct{ bob.Query }

func (q unwrappedQuery) WriteSQL(ctx context.Context, w io.StringWriter, _ bob.Dialect, start int) ([]any, error) {
	return q.WriteQuery(ctx, w, start)
}

// The expected SQL is compared exactly, without normalization, because
// the doubled parentheses that https://github.com/stephenafamo/bob/issues/742
// guards against would be collapsed by testutils.Clean.
func TestPrefixedSubqueryParens(t *testing.T) {
	query := RawQuery(dialect{}, "SELECT ?", 42)

	cases := map[string]struct {
		expr bob.Expression
		want string
		args []any
	}{
		"exists query":        {Exists[chain, chain](query), "EXISTS (SELECT ?3)", []any{42}},
		"exists custom query": {Exists[chain, chain](unwrappedQuery{query}), "EXISTS (SELECT ?3)", []any{42}},
		"any query":           {Any[chain, chain](query), "ANY (SELECT ?3)", []any{42}},
		"all query":           {All[chain, chain](query), "ALL (SELECT ?3)", []any{42}},
		"exists non-query":    {Exists[chain, chain](Raw("SELECT 1")), "EXISTS (SELECT 1)", nil},
		"any arg":             {Any[chain, chain](Arg(1)), "ANY (?3)", []any{1}},
		"all arg":             {All[chain, chain](Arg(1)), "ALL (?3)", []any{1}},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := &strings.Builder{}
			args, err := tc.expr.WriteSQL(t.Context(), w, dialect{}, 3)
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(tc.args, args) {
				t.Fatalf("got args %v, want %v", args, tc.args)
			}
			if got := w.String(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
