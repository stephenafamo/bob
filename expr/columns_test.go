package expr

import (
	"testing"

	"github.com/stephenafamo/bob"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestColumnsExprExpressions(t *testing.T) {
	examples := testutils.ExpressionTestcases{
		"with parent": {
			Expression:  NewColumnsExpr("id", "name").WithParent("t").Expressions(),
			ExpectedSQL: `"t"."id" AS "id", "t"."name" AS "name"`,
		},
		"with agg and alias": {
			Expression:  NewColumnsExpr("id", "name").WithParent("t").WithAggFunc("count(", ")").WithPrefix("x_").Expressions(),
			ExpectedSQL: `count("t"."id") AS "x_id", count("t"."name") AS "x_name"`,
		},
		"with agg and alias disabled": {
			Expression:  NewColumnsExpr("id", "name").WithParent("t").WithAggFunc("count(", ")").WithPrefix("x_").DisableAlias().Expressions(),
			ExpectedSQL: `count("t"."id"), count("t"."name")`,
		},
		"without parent": {
			Expression:  NewColumnsExpr("id", "name").Expressions(),
			ExpectedSQL: `"id", "name"`,
		},
		"skips empty parent parts": {
			Expression:  NewColumnsExpr("id", "name").WithParent("", "t", "").Expressions(),
			ExpectedSQL: `"t"."id" AS "id", "t"."name" AS "name"`,
		},
		"only empty parent parts": {
			Expression:  NewColumnsExpr("id", "name").WithParent("", "").Expressions(),
			ExpectedSQL: `"id", "name"`,
		},
	}

	testutils.RunExpressionTests(t, dialect{}, examples)
}

func TestColumnsExprExpressionsAny(t *testing.T) {
	cols := NewColumnsExpr("id", "name").WithParent("t").Expressions()
	anyCols := cols.Any()

	if len(anyCols) != 2 {
		t.Fatalf("expected 2 expressions, got %d", len(anyCols))
	}

	for i, col := range anyCols {
		if _, ok := col.(bob.Expression); !ok {
			t.Fatalf("expected item %d to implement bob.Expression, got %T", i, col)
		}
	}
}
