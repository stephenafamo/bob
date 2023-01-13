package sqlite_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
	"github.com/stephenafamo/bob/dialect/sqlite/um"
	testutils "github.com/stephenafamo/bob/test_utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: sqlite.Update(
				um.Table("films"),
				um.SetArg("kind", "Dramatic"),
				um.Where(sqlite.X("kind").EQ(sqlite.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = ?1 WHERE (kind = ?2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: sqlite.Update(
				um.Table("employees"),
				um.Set("sales_count", "sales_count + 1"),
				um.From("accounts"),
				um.Where(sqlite.X("accounts.name").EQ(sqlite.Arg("Acme Corporation"))),
				um.Where(sqlite.X("employees.id").EQ("accounts.sales_person")),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
			  WHERE (accounts.name = ?1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees AS "e" NOT INDEXED
				SET "sales_count" = sales_count + 1
				WHERE (id = (SELECT sales_person FROM accounts WHERE (name = ?1)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: sqlite.Update(
				um.TableAs("employees", "e"),
				um.TableNotIndexed(),
				um.Set("sales_count", "sales_count + 1"),
				um.Where(sqlite.X("id").EQ(sqlite.P(sqlite.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(sqlite.X("name").EQ(sqlite.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}
