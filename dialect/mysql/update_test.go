package mysql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/mysql"
	"github.com/stephenafamo/bob/dialect/mysql/sm"
	"github.com/stephenafamo/bob/dialect/mysql/um"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestUpdate(t *testing.T) {
	examples := testutils.Testcases{
		"simple": {
			Query: mysql.Update(
				um.Table("films"),
				um.SetCol("kind").ToArg("Dramatic"),
				um.Where(mysql.Quote("kind").EQ(mysql.Arg("Drama"))),
			),
			ExpectedSQL:  "UPDATE films SET `kind` = ? WHERE (`kind` = ?)",
			ExpectedArgs: []any{"Dramatic", "Drama"},
		},
		"update multiple tables": {
			Query: mysql.Update(
				um.Table("employees, accounts"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.Where(mysql.Quote("accounts", "name").EQ(mysql.Arg("Acme Corporation"))),
				um.Where(mysql.Quote("employees", "id").EQ(mysql.Quote("accounts", "sales_person"))),
			),
			ExpectedSQL:  "UPDATE employees, accounts SET `sales_count` = sales_count + 1 WHERE (`accounts`.`name` = ?) AND (`employees`.`id` = `accounts`.`sales_person`)",
			ExpectedArgs: []any{"Acme Corporation"},
		},
		"setExpr as mod": {
			Query: mysql.Update(
				um.Table(mysql.Quote("table1").As("T1")),
				um.LeftJoin(mysql.Quote("table2").As("T2")).
					OnEQ(mysql.Quote("T1", "some_id"), mysql.Quote("T2", "id")),
				um.SetExpr(mysql.Quote("T1", "some_value")).ToArg("test"),
				um.Where(mysql.Quote("T1", "id").EQ(mysql.Arg(1))),
				um.Where(mysql.Quote("T2", "other_value").EQ(mysql.Arg("something"))),
			),
			ExpectedSQL:  "UPDATE `table1` AS `T1` LEFT JOIN `table2` AS `T2` ON (`T1`.`some_id` = `T2`.`id`) SET `T1`.`some_value` = ? WHERE (`T1`.`id` = ?) AND (`T2`.`other_value` = ?)",
			ExpectedArgs: []any{"test", 1, "something"},
		},
		"setCol and setExpr via Set helper": {
			Query: mysql.Update(
				um.Table(mysql.Quote("table1").As("T1")),
				um.LeftJoin(mysql.Quote("table2").As("T2")).
					OnEQ(mysql.Quote("T1", "some_id"), mysql.Quote("T2", "id")),
				um.Set(
					um.SetCol("sales_count").To("sales_count + 1"),
					um.SetExpr(mysql.Quote("T1", "some_value")).ToArg("test"),
				),
				um.Where(mysql.Quote("T1", "id").EQ(mysql.Arg(1))),
			),
			ExpectedSQL:  "UPDATE `table1` AS `T1` LEFT JOIN `table2` AS `T2` ON (`T1`.`some_id` = `T2`.`id`) SET `sales_count` = sales_count + 1, `T1`.`some_value` = ? WHERE (`T1`.`id` = ?)",
			ExpectedArgs: []any{"test", 1},
		},
		"assign via Set helper": {
			Doc: "Assign via Set",
			Query: mysql.Update(
				um.Table(mysql.Quote("table1").As("T1")),
				um.LeftJoin(mysql.Quote("table2").As("T2")).
					OnEQ(mysql.Quote("T1", "some_id"), mysql.Quote("T2", "id")),
				um.Set(
					mysql.Quote("sales_count").Assign(mysql.Raw("sales_count + 1")),
					mysql.Quote("T1", "some_value").Assign(mysql.Arg("test")),
				),
				um.Where(mysql.Quote("T1", "id").EQ(mysql.Arg(1))),
			),
			ExpectedSQL:  "UPDATE `table1` AS `T1` LEFT JOIN `table2` AS `T2` ON (`T1`.`some_id` = `T2`.`id`) SET `sales_count` = sales_count + 1, `T1`.`some_value` = ? WHERE (`T1`.`id` = ?)",
			ExpectedArgs: []any{"test", 1},
		},
		"with sub-select": {
			ExpectedSQL:  "UPDATE employees SET `sales_count` = sales_count + 1 WHERE (`id` = (SELECT sales_person FROM accounts WHERE (`name` = ?)))",
			ExpectedArgs: []any{"Acme Corporation"},
			Query: mysql.Update(
				um.Table("employees"),
				um.SetCol("sales_count").To("sales_count + 1"),
				um.Where(mysql.Quote("id").EQ(mysql.Select(
					sm.Columns("sales_person"),
					sm.From("accounts"),
					sm.Where(mysql.Quote("name").EQ(mysql.Arg("Acme Corporation"))),
				))),
			),
		},
	}

	testutils.RunTests(t, examples, formatter)
}
