package psql_test

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: psql.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
				um.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = $1 WHERE (kind = $2)`,
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"with from": {
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.From("accounts"),
				um.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
				um.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts
			  WHERE (accounts.name = $1)
			  AND (employees.id = accounts.sales_person)`,
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"with from append": {
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.FromAppend("accounts"),
				um.FromAppend("departments"),
				um.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts, departments
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"with from and from append": {
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.From("accounts"),
				um.FromAppend("departments"),
				um.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM accounts, departments
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"with from append then from override": {
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.FromAppend("accounts"),
				um.From("departments"),
				um.Where(psql.Quote("employees", "id").EQ(psql.Arg(1))),
			),
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 FROM departments
			  WHERE (employees.id = $1)`,
			ExpectedArgs: []any{1},
		},
		"set tuple columns from row": {
			Query: psql.Update(
				um.Table("weather"),
				um.SetCols("temp_lo", "temp_hi", "prcp").ToRow(
					psql.Raw("temp_lo + 1"),
					psql.Raw("temp_lo + 15"),
					psql.Raw("DEFAULT"),
				),
				um.Where(psql.Quote("city").EQ(psql.Arg("San Francisco"))),
			),
			ExpectedSQL:  `UPDATE weather SET (temp_lo, temp_hi, prcp) = ROW (temp_lo + 1, temp_lo + 15, DEFAULT) WHERE (city = $1)`,
			ExpectedArgs: []any{"San Francisco"},
		},
		"set tuple columns from sub-select": {
			Query: psql.Update(
				um.Table("accounts"),
				um.SetCols("contact_first_name", "contact_last_name").ToQuery(psql.Select(
					sm.Columns("first_name", "last_name"),
					sm.From("employees"),
					sm.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
				)),
			),
			ExpectedSQL: `UPDATE accounts SET (contact_first_name, contact_last_name) =
			  (SELECT first_name, last_name FROM employees WHERE (employees.id = accounts.sales_person))`,
		},
		"where current of": {
			Query: psql.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
				um.WhereCurrentOf("c_films"),
			),
			ExpectedSQL:  `UPDATE films SET "kind" = $1 WHERE CURRENT OF c_films`,
			ExpectedArgs: []any{"Dramatic"},
		},
		"with sub-select": {
			ExpectedSQL: `UPDATE employees SET "sales_count" = sales_count + 1 WHERE (id =
				  (SELECT sales_person FROM accounts WHERE (name = $1)))`,
			ExpectedArgs: []any{"Acme Corporation"},
			Query: psql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.Where(psql.Quote("id").EQ(psql.Group(psql.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(psql.Quote("name").EQ(psql.Arg("Acme Corporation"))),
				)))),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func TestUpdateReturningWith(t *testing.T) {
	examples := testutils.Testcases{
		"returning with old and new aliases": {
			Query: psql.Update(
				um.Table("users"),
				um.SetCol("primary_email").ToArg("new@example.com"),
				um.Where(psql.Quote("id").EQ(psql.Arg(1))),
				um.Returning(
					psql.Quote("before", "primary_email"),
					psql.Quote("after", "primary_email"),
				).WithOldAs("before").WithNewAs("after"),
			),
			ExpectedSQL:  `UPDATE users SET "primary_email" = $1 WHERE ("id" = $2) RETURNING WITH (OLD AS "before", NEW AS "after") "before"."primary_email", "after"."primary_email"`,
			ExpectedArgs: []any{"new@example.com", 1},
		},
	}

	testutils.RunTests(t, examples, nil)
}

func TestUpdateWhereCurrentOfConflict(t *testing.T) {
	_, _, err := bob.Build(context.Background(), psql.Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
		um.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
		um.WhereCurrentOf("c_films"),
	))

	if err == nil {
		t.Fatal("expected error when both WHERE and WHERE CURRENT OF are set")
	}
}
