package expr

import (
	"testing"

	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestStatement(t *testing.T) {
	examples := testutils.ExpressionTestcases{
		"plain": {
			Expression: Clause{
				query: "SELECT a, b FROM alphabet",
			},
			ExpectedSQL: `SELECT a, b FROM alphabet`,
		},
		"escaped args": {
			Expression: Clause{
				query: `SELECT a, b FROM "alphabet\?" WHERE c = ? AND d <= ?`,
				args:  []any{1, 2},
			},
			ExpectedSQL:  `SELECT a, b FROM "alphabet?" WHERE c = ?1 AND d <= ?2`,
			ExpectedArgs: []any{1, 2},
		},
		"mismatched args and placeholders": {
			Expression: Clause{
				query: "SELECT a, b FROM alphabet WHERE c = ? AND d <= ?",
			},
			ExpectedSQL:   `SELECT a, b FROM alphabet WHERE c = ?1 AND d <= ?2`,
			ExpectedError: &rawError{args: 0, placeholders: 2},
		},
		"numbered args": {
			Expression: Clause{
				query: "SELECT a, b FROM alphabet WHERE c = ? AND d <= ?",
				args:  []any{1, 2},
			},
			ExpectedSQL:  `SELECT a, b FROM alphabet WHERE c = ?1 AND d <= ?2`,
			ExpectedArgs: []any{1, 2},
		},
	}

	testutils.RunExpressionTests(t, dialect{}, examples)
}
