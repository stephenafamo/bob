package expr

import (
	"context"
	"io"
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

// Mimics bob.BaseQuery, which writes its own surrounding parentheses when
// rendered as an expression.
type selfWrappingQuery struct{ sql string }

func (q selfWrappingQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("(")
	w.WriteString(q.sql)
	w.WriteString(")")
	return nil, nil
}

func (q selfWrappingQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	return q.WriteSQL(ctx, w, dialect{}, start)
}

func (q selfWrappingQuery) Type() bob.QueryType { return bob.QueryTypeSelect }

// The expected SQL is compared exactly, without normalization, because
// the doubled parentheses that https://github.com/stephenafamo/bob/issues/742
// guards against would be collapsed by testutils.Clean.
func TestPrefixedSubqueryParens(t *testing.T) {
	query := selfWrappingQuery{sql: "SELECT 1 FROM users"}

	cases := map[string]struct {
		expr bob.Expression
		want string
	}{
		"exists query":     {Exists[chain, chain](query), "EXISTS (SELECT 1 FROM users)"},
		"any query":        {Any[chain, chain](query), "ANY (SELECT 1 FROM users)"},
		"all query":        {All[chain, chain](query), "ALL (SELECT 1 FROM users)"},
		"exists non-query": {Exists[chain, chain](Raw("SELECT 1")), "EXISTS (SELECT 1)"},
		"any arg":          {Any[chain, chain](Arg(1)), "ANY (?1)"},
		"all arg":          {All[chain, chain](Arg(1)), "ALL (?1)"},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			w := &strings.Builder{}
			if _, err := tc.expr.WriteSQL(context.Background(), w, dialect{}, 1); err != nil {
				t.Fatal(err)
			}
			if got := w.String(); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
