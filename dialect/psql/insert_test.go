package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	testutils "github.com/stephenafamo/bob/test/utils"
)

func TestInsert(t *testing.T) {
	examples := testutils.Testcases{
		"simple insert": {
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL:  "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6)",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"insert from select": {
			Query: psql.Insert(
				im.Into("films"),
				im.Query(psql.Select(
					sm.From("tmp_films"),
					sm.Where(psql.Quote("date_prod").LT(psql.Arg("1971-07-13"))),
				)),
			),
			ExpectedSQL:  `INSERT INTO films SELECT * FROM tmp_films WHERE "date_prod" < $1`,
			ExpectedArgs: []any{"1971-07-13"},
		},
		"insert with cte": {
			Query: psql.Insert(
				im.With("src").As(psql.Select(
					sm.From("tmp_films"),
				)),
				im.Into("films"),
				im.Query(psql.Select(
					sm.From("src"),
				)),
			),
			ExpectedSQL: `WITH src AS (SELECT * FROM tmp_films)
				INSERT INTO films SELECT * FROM src`,
		},
		"insert with recursive cte": {
			Query: psql.Insert(
				im.Recursive(true),
				im.With("src").As(psql.Select(
					sm.From("tmp_films"),
				)),
				im.Into("films"),
				im.Query(psql.Select(
					sm.From("src"),
				)),
			),
			ExpectedSQL: `WITH RECURSIVE src AS (SELECT * FROM tmp_films)
				INSERT INTO films SELECT * FROM src`,
		},
		"bulk insert": {
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
			),
			ExpectedSQL: `INSERT INTO films VALUES
				($1, $2, $3, $4, $5, $6),
				($7, $8, $9, $10, $11, $12)`,
			ExpectedArgs: []any{
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
				"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins",
			},
		},
		"upsert": {
			Query: psql.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(psql.Arg(8, "Anvil Distribution")),
				im.Values(psql.Arg(9, "Sentry Distribution")),
				im.OnConflict("did").DoUpdate(
					im.SetCol("dname").To(psql.Concat(
						psql.Raw("EXCLUDED.dname"), psql.S(" (formerly "),
						psql.Quote("d", "dname"), psql.S(")"),
					)),
					im.Where(psql.Quote("d", "zipcode").NE(psql.S("21201"))),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES ($1, $2), ($3, $4)
				ON CONFLICT (did) DO UPDATE
				SET dname = (EXCLUDED.dname || ' (formerly ' || d.dname || ')')
				WHERE (d.zipcode <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"upsert on constraint": {
			Query: psql.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(psql.Arg(8, "Anvil Distribution")),
				im.Values(psql.Arg(9, "Sentry Distribution")),
				im.OnConflictOnConstraint("distributors_pkey").DoUpdate(
					im.SetExcluded("dname"),
					im.Where(psql.Quote("d", "zipcode").NE(psql.S("21201"))),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
				VALUES ($1, $2), ($3, $4)
				ON CONFLICT ON CONSTRAINT distributors_pkey DO UPDATE
				SET dname = EXCLUDED.dname
				WHERE (d.zipcode <> '21201')`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"upsert using excluded helper in set": {
			Query: psql.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(psql.Arg(8, "Anvil Distribution")),
				im.Values(psql.Arg(9, "Sentry Distribution")),
				im.OnConflict("did").DoUpdate(
					im.SetCol("dname").To(im.Excluded("dname")),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
			   VALUES ($1, $2), ($3, $4)
			   ON CONFLICT (did) DO UPDATE
			   SET dname = EXCLUDED.dname`,
			ExpectedArgs: []any{8, "Anvil Distribution", 9, "Sentry Distribution"},
		},
		"upsert setCol and setExpr via Set helper": {
			Query: psql.Insert(
				im.IntoAs("distributors", "d", "did", "dname"),
				im.Values(psql.Arg(8, "Anvil Distribution")),
				im.OnConflict("did").DoUpdate(
					im.Set(
						im.SetCol("dname").To(im.Excluded("dname")),
						im.SetExpr(psql.Quote("d", "did")).To(im.Excluded("did")),
					),
				),
			),
			ExpectedSQL: `INSERT INTO distributors AS "d" ("did", "dname")
			   VALUES ($1, $2)
			   ON CONFLICT (did) DO UPDATE
			   SET dname = EXCLUDED.dname,
			   "d"."did" = EXCLUDED.did`,
			ExpectedArgs: []any{8, "Anvil Distribution"},
		},
		"insert overriding system value": {
			Query: psql.Insert(
				im.Into("users", "id", "name"),
				im.OverridingSystem(),
				im.Values(psql.Arg(1, "Neo")),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "name") OVERRIDING SYSTEM VALUE VALUES ($1, $2)`,
			ExpectedArgs: []any{1, "Neo"},
		},
		"insert overriding user value": {
			Query: psql.Insert(
				im.Into("users", "id", "name"),
				im.OverridingUser(),
				im.Values(psql.Arg(1, "Neo")),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "name") OVERRIDING USER VALUE VALUES ($1, $2)`,
			ExpectedArgs: []any{1, "Neo"},
		},
		"on conflict do nothing": {
			Doc: "Upsert DO NOTHING",
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins")),
				im.OnConflict().DoNothing(),
			),
			ExpectedSQL:  "INSERT INTO films VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING",
			ExpectedArgs: []any{"UA502", "Bananas", 105, "1971-07-13", "Comedy", "82 mins"},
		},
		"on conflict with target where": {
			Query: psql.Insert(
				im.Into("distributors", "did", "is_active"),
				im.Values(psql.Arg(10, true)),
				im.OnConflict("did").Where(psql.Raw("is_active")).DoNothing(),
			),
			ExpectedSQL:  `INSERT INTO distributors ("did", "is_active") VALUES ($1, $2) ON CONFLICT (did) WHERE is_active DO NOTHING`,
			ExpectedArgs: []any{10, true},
		},
		"on conflict with target expression": {
			Query: psql.Insert(
				im.Into("users", "email"),
				im.Values(psql.Arg("neo@example.com")),
				im.OnConflict(
					im.ConflictTarget(psql.Raw("lower(email)")),
				).DoNothing(),
			),
			ExpectedSQL:  `INSERT INTO users ("email") VALUES ($1) ON CONFLICT (lower(email)) DO NOTHING`,
			ExpectedArgs: []any{"neo@example.com"},
		},
		"on conflict with collate and opclass": {
			Query: psql.Insert(
				im.Into("users", "email"),
				im.Values(psql.Arg("neo@example.com")),
				im.OnConflict(
					im.ConflictTarget("email").Collate("en_US").OpClass("text_pattern_ops"),
				).DoNothing(),
			),
			ExpectedSQL:  `INSERT INTO users ("email") VALUES ($1) ON CONFLICT (email COLLATE en_US text_pattern_ops) DO NOTHING`,
			ExpectedArgs: []any{"neo@example.com"},
		},
		"on conflict with expression collation and opclass": {
			Query: psql.Insert(
				im.Into("users", "email"),
				im.Values(psql.Arg("neo@example.com")),
				im.OnConflict(
					im.ConflictTarget("email").Collate(psql.Quote("public", "en_US.UTF-8")).OpClass(psql.Quote("public", "my opclass")),
				).DoNothing(),
			),
			ExpectedSQL:  `INSERT INTO users ("email") VALUES ($1) ON CONFLICT (email COLLATE "public"."en_US.UTF-8" "public"."my opclass") DO NOTHING`,
			ExpectedArgs: []any{"neo@example.com"},
		},
		"on conflict with schema qualified opclass": {
			Query: psql.Insert(
				im.Into("users", "email"),
				im.Values(psql.Arg("neo@example.com")),
				im.OnConflict(
					im.ConflictTarget("email").OpClass("public.text_pattern_ops"),
				).DoNothing(),
			),
			ExpectedSQL:  `INSERT INTO users ("email") VALUES ($1) ON CONFLICT (email public.text_pattern_ops) DO NOTHING`,
			ExpectedArgs: []any{"neo@example.com"},
		},
		"on conflict mixed target items": {
			Query: psql.Insert(
				im.Into("users", "tenant_id", "email"),
				im.Values(psql.Arg(1, "neo@example.com")),
				im.OnConflict(
					im.ConflictTarget("tenant_id").OpClass("int4_ops"),
					im.ConflictTarget(psql.Raw("lower(email)")).Collate("en_US"),
				).DoNothing(),
			),
			ExpectedSQL:  `INSERT INTO users ("tenant_id", "email") VALUES ($1, $2) ON CONFLICT (tenant_id int4_ops, lower(email) COLLATE en_US) DO NOTHING`,
			ExpectedArgs: []any{1, "neo@example.com"},
		},
		"on conflict do update set tuple to exprs": {
			Query: psql.Insert(
				im.Into("users", "id", "first_name", "last_name"),
				im.Values(psql.Arg(1, "Thomas", "Anderson")),
				im.OnConflict("id").DoUpdate(
					im.SetCols("first_name", "last_name").ToExprs(
						psql.Raw("EXCLUDED.first_name"),
						psql.Raw("EXCLUDED.last_name"),
					),
				),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "first_name", "last_name") VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET ("first_name", "last_name") = (EXCLUDED.first_name, EXCLUDED.last_name)`,
			ExpectedArgs: []any{1, "Thomas", "Anderson"},
		},
		"on conflict do update set tuple to row": {
			Query: psql.Insert(
				im.Into("users", "id", "first_name", "last_name"),
				im.Values(psql.Arg(1, "Thomas", "Anderson")),
				im.OnConflict("id").DoUpdate(
					im.SetCols("first_name", "last_name").ToRow(
						psql.Arg("Neo"),
						psql.Arg("Anderson"),
					),
					im.Where(psql.Raw("users.deleted_at IS NULL")),
				),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "first_name", "last_name") VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET ("first_name", "last_name") = ROW ($4, $5) WHERE users.deleted_at IS NULL`,
			ExpectedArgs: []any{1, "Thomas", "Anderson", "Neo", "Anderson"},
		},
		"on conflict do update set tuple to query": {
			Query: psql.Insert(
				im.Into("users", "id", "first_name", "last_name"),
				im.Values(psql.Arg(1, "Thomas", "Anderson")),
				im.OnConflict("id").DoUpdate(
					im.SetCols("first_name", "last_name").ToQuery(psql.Select(
						sm.Columns("first_name", "last_name"),
						sm.From("archived_users"),
						sm.Where(psql.Raw("archived_users.id = EXCLUDED.id")),
					)),
				),
			),
			ExpectedSQL:  `INSERT INTO users ("id", "first_name", "last_name") VALUES ($1, $2, $3) ON CONFLICT (id) DO UPDATE SET ("first_name", "last_name") = (SELECT first_name, last_name FROM archived_users WHERE archived_users.id = EXCLUDED.id)`,
			ExpectedArgs: []any{1, "Thomas", "Anderson"},
		},
		"insert with excluded in where": {
			Query: psql.Insert(
				im.Into("films"),
				im.Values(psql.Arg("UA502", "Bananas")),
				im.OnConflict("id").DoUpdate(
					im.SetExcluded("title"),
					im.Where(im.Excluded("id").EQ(psql.Arg(1))),
				),
			),
			ExpectedSQL:  `INSERT INTO films VALUES ($1, $2) ON CONFLICT (id) DO UPDATE SET title = EXCLUDED.title WHERE EXCLUDED.id = $3`,
			ExpectedArgs: []any{"UA502", "Bananas", 1},
		},
	}

	testutils.RunTests(t, examples, formatter)
}

func TestInsertReturningWith(t *testing.T) {
	examples := testutils.Testcases{
		"returning with old and new aliases": {
			Query: psql.Insert(
				im.Into("users"),
				im.Values(psql.Arg(1, "neo@example.com")),
				im.Returning(
					psql.Quote("before", "id"),
					psql.Quote("after", "primary_email"),
				).WithOldAs("before").WithNewAs("after"),
			),
			ExpectedSQL:  `INSERT INTO users VALUES ($1, $2) RETURNING WITH (OLD AS "before", NEW AS "after") "before"."id", "after"."primary_email"`,
			ExpectedArgs: []any{1, "neo@example.com"},
		},
	}

	testutils.RunTests(t, examples, nil)
}
