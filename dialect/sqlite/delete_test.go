package sqlite_test

import (
	"testing"

	d "github.com/stephenafamo/bob/dialect"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/delete/qm"
)

func TestDelete(t *testing.T) {
	examples := d.Testcases{
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
