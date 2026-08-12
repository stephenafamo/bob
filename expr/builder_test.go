package expr

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
)

type builderTestExpression struct {
	bob.Expression
}

func (builderTestExpression) New(exp bob.Expression) builderTestExpression {
	return builderTestExpression{Expression: exp}
}

func TestQuantifiedExpressionParentheses(t *testing.T) {
	query := bob.BaseQuery[bob.Expression]{
		Expression: bob.ExpressionFunc(func(_ context.Context, w io.StringWriter, _ bob.Dialect, _ int) ([]any, error) {
			w.WriteString("SELECT 1")
			return nil, nil
		}),
		Dialect:   dialect{},
		QueryType: bob.QueryTypeSelect,
	}

	operators := map[string]struct {
		build func(bob.Expression) builderTestExpression
		want  string
	}{
		"exists": {
			build: Exists[builderTestExpression, builderTestExpression],
			want:  "EXISTS (SELECT 1)",
		},
		"any": {
			build: Any[builderTestExpression, builderTestExpression],
			want:  "ANY (SELECT 1)",
		},
		"all": {
			build: All[builderTestExpression, builderTestExpression],
			want:  "ALL (SELECT 1)",
		},
	}
	operands := map[string]bob.Expression{
		"query":      query,
		"expression": Raw("SELECT 1"),
	}

	for operatorName, operator := range operators {
		for operandName, operand := range operands {
			t.Run(operatorName+" "+operandName, func(t *testing.T) {
				var got strings.Builder
				_, err := operator.build(operand).WriteSQL(t.Context(), &got, dialect{}, 1)
				if err != nil {
					t.Fatalf("write expression: %v", err)
				}

				if got.String() != operator.want {
					t.Fatalf("expected %q, got %q", operator.want, got.String())
				}
			})
		}
	}
}
