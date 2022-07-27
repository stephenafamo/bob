package sqlite_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite"
)

func TestDelete(t *testing.T) {
	var qm = sqlite.DeleteQM{}
	var examples = d.Testcases{
		"simple": {
			Query: sqlite.Delete(
				qm.From("films"),
				qm.Where(sqlite.X("kind").EQ(sqlite.Arg("Drama"))),
			),
			ExpectedSQL:  `DELETE FROM films WHERE (kind = ?1)`,
			ExpectedArgs: []any{"Drama"},
		},
	}

	d.RunTests(t, examples)
}
