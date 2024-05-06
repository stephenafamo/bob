package expr

import (
	"testing"

	testutils "github.com/stephenafamo/bob/test/utils"
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
		"expr args": {
			Expression: Clause{
				query: "SELECT a, b FROM alphabet WHERE c IN (?) AND d <= ?",
				args:  []any{Arg(5, 6, 7), 2},
			},
			ExpectedSQL:  `SELECT a, b FROM alphabet WHERE c IN (?1, ?2, ?3) AND d <= ?4`,
			ExpectedArgs: []any{5, 6, 7, 2},
		},
		"expr args group": {
			Expression: Clause{
				query: "SELECT a, b FROM alphabet WHERE c IN ? AND d <= ?",
				args:  []any{ArgGroup(5, 6, 7), 2},
			},
			ExpectedSQL:  `SELECT a, b FROM alphabet WHERE c IN (?1, ?2, ?3) AND d <= ?4`,
			ExpectedArgs: []any{5, 6, 7, 2},
		},
		"expr args quote": {
			Expression: Clause{
				query: "SELECT a, b FROM alphabet WHERE c = ? AND d <= ?",
				args:  []any{Quote("AA"), 2},
			},
			ExpectedSQL:  `SELECT a, b FROM alphabet WHERE c = "AA" AND d <= ?1`,
			ExpectedArgs: []any{2},
		},
	}

	testutils.RunExpressionTests(t, dialect{}, examples)
}
