package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/dm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestDelete(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: mysql.Delete(
				dm.From("films"),
				dm.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  "DELETE FROM films WHERE (`kind` = ?)",
			ExpectedArgs: []any{"Drama"},
		},
		"multiple tables": {
			Query: mysql.Delete(
				dm.From("films"),
				dm.From("actors"),
				dm.Using("films"),
				dm.InnerJoin("film_actors").OnEQ(mysql.Raw("films.id"), mysql.Raw("film_actors.film_id")),
				dm.InnerJoin("actors").OnEQ(mysql.Raw("film_actors.actor_id"), mysql.Raw("actors.id")),
				dm.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL: `DELETE FROM films, actors USING films
			` + "INNER JOIN film_actors ON (films.id = film_actors.film_id)" + `
			` + "INNER JOIN actors ON (film_actors.actor_id = actors.id) WHERE (`kind` = ?)",
			ExpectedArgs: []any{"Drama"},
		},
		"with limit and offest": {
			Query: mysql.Delete(
				dm.From("films"),
				dm.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
				dm.Limit(10),
				dm.OrderBy("producer").Desc(),
			),
			ExpectedSQL:  "DELETE FROM films WHERE (`kind` = ?) ORDER BY producer DESC LIMIT 10",
			ExpectedArgs: []any{"Drama"},
		},
		"with using": {
			Query: mysql.Delete(
				dm.From("employees"),
				dm.Using("accounts"),
				dm.Where(mysql.Quote("accounts", "name").EQ(mysql.Arg("Acme Corporation"))),
				dm.Where(mysql.Quote("employees", "id").EQ(mysql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL:  "DELETE FROM employees USING accounts WHERE (`accounts`.`name` = ?) AND (`employees`.`id` = `accounts`.`sales_person`)",
			ExpectedArgs: []any{"Acme Corporation"},
		},
	}

	testutils.RunTests(t, examples, formatter)
}
