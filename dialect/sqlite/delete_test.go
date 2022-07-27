package sqlite

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
)

func TestDelete(t *testing.T) {
	var qm = DeleteQM{}
	var examples = d.Testcases{
		"simple": {
			Query: Delete(
				qm.From("films"),
				qm.Where(qm.X("kind").EQ(qm.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = ?1)`,
			ExpectedArgs: []any{"Drama"},
		},
	}

	d.RunTests(t, examples)
}
