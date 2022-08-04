package expr

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
)

func TestStatement(t *testing.T) {
	examples := d.ExpressionTestcases{
		"plain": {
			Expression: Raw{
				query: "SELECT a, b FROM alphabet",
			},
			ExpectedSQL: `SELECT a, b FROM alphabet`,
		},
		"escaped args": {
			Expression: Raw{
				query: `SELECT a, b FROM "alphabet\?" WHERE c = ? AND d <= ?`,
				args:  []any{1, 2},
			},
			ExpectedSQL:  `SELECT a, b FROM "alphabet?" WHERE c = ?1 AND d <= ?2`,
			ExpectedArgs: []any{1, 2},
		},
		"mismatched args and placeholders": {
			Expression: Raw{
				query: "SELECT a, b FROM alphabet WHERE c = ? AND d <= ?",
			},
			ExpectedSQL:   `SELECT a, b FROM alphabet WHERE c = ?1 AND d <= ?2`,
			ExpectedError: &rawError{args: 0, placeholders: 2},
		},
		"numbered args": {
			Expression: Raw{
				query: "SELECT a, b FROM alphabet WHERE c = ? AND d <= ?",
				args:  []any{1, 2},
			},
			ExpectedSQL:  `SELECT a, b FROM alphabet WHERE c = ?1 AND d <= ?2`,
			ExpectedArgs: []any{1, 2},
		},
	}

	d.RunExpressionTests(t, dialect{}, examples)
}
