package psql_test

import (
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
	"github.com/stephenafamo/bob/expr"
	testutils "github.com/stephenafamo/bob/test/utils"
)

type withTestStruct struct {
	ID    int64  `db:"id,pk"`
	Name  string `db:"name"`
	Email string `db:"email"`
}

var withTestStructView = psql.NewView[*withTestStruct, bob.Expression](
	"public",
	"with_test_struct",
	expr.ColsForStruct[withTestStruct]("with_test_struct"),
)

func TestSelectWithRegression(t *testing.T) {
	t.Run("native path matches direct construction", func(t *testing.T) {
		base := psql.Select(
			sm.Columns("id", "name"),
			sm.From("users"),
			sm.Where(psql.Quote("tenant_id").EQ(psql.Arg(42))),
		)

		derived := base.With(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		assertQueriesEqual(t, base, psql.Select(
			sm.Columns("id", "name"),
			sm.From("users"),
			sm.Where(psql.Quote("tenant_id").EQ(psql.Arg(42))),
		))
		assertQueriesEqual(t, derived, psql.Select(
			sm.Columns("id", "name"),
			sm.From("users"),
			sm.Where(psql.Quote("tenant_id").EQ(psql.Arg(42))),
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		))
	})

	t.Run("fallback path matches direct construction", func(t *testing.T) {
		base := psql.Select(
			sm.Columns("users.id", "users.name"),
			sm.From("users"),
		)

		derived := base.With(
			sm.LeftJoin("teams").Using("id"),
			sm.ForUpdate("users").SkipLocked(),
		)

		assertQueriesEqual(t, base, psql.Select(
			sm.Columns("users.id", "users.name"),
			sm.From("users"),
		))
		assertQueriesEqual(t, derived, psql.Select(
			sm.Columns("users.id", "users.name"),
			sm.From("users"),
			sm.LeftJoin("teams").Using("id"),
			sm.ForUpdate("users").SkipLocked(),
		))
	})
}

func TestViewQueryWithRegression(t *testing.T) {
	base := withTestStructView.Query(
		sm.Where(psql.Quote("id").GT(psql.Arg(0))),
	)

	derived := base.With(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	assertQueriesEqual(t, base, withTestStructView.Query(
		sm.Where(psql.Quote("id").GT(psql.Arg(0))),
	))
	assertQueriesEqual(t, derived.Query, withTestStructView.Query(
		sm.Where(psql.Quote("id").GT(psql.Arg(0))),
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	))
}

func TestUpdateWithRegression(t *testing.T) {
	base := psql.Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
	)

	derived := base.With(
		um.SetCol("updated_at").To("NOW()"),
		um.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
		um.Returning("id"),
	)

	assertQueriesEqual(t, base, psql.Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
	))
	assertQueriesEqual(t, derived, psql.Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
		um.SetCol("updated_at").To("NOW()"),
		um.Where(psql.Quote("kind").EQ(psql.Arg("Drama"))),
		um.Returning("id"),
	))
}

func TestDeleteWithRegression(t *testing.T) {
	base := psql.Delete(
		dm.From("employees"),
	)

	derived := base.With(
		dm.Using("accounts"),
		dm.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
		dm.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
		dm.Returning("id"),
	)

	assertQueriesEqual(t, base, psql.Delete(
		dm.From("employees"),
	))
	assertQueriesEqual(t, derived, psql.Delete(
		dm.From("employees"),
		dm.Using("accounts"),
		dm.Where(psql.Quote("accounts", "name").EQ(psql.Arg("Acme Corporation"))),
		dm.Where(psql.Quote("employees", "id").EQ(psql.Quote("accounts", "sales_person"))),
		dm.Returning("id"),
	))
}

func TestInsertWithRegression(t *testing.T) {
	t.Run("native path matches direct construction", func(t *testing.T) {
		base := psql.Insert(
			im.Into("films"),
			im.Values(psql.Arg("UA502", "Bananas")),
		)

		derived := base.With(
			im.Returning("id"),
		)

		assertQueriesEqual(t, base, psql.Insert(
			im.Into("films"),
			im.Values(psql.Arg("UA502", "Bananas")),
		))
		assertQueriesEqual(t, derived, psql.Insert(
			im.Into("films"),
			im.Values(psql.Arg("UA502", "Bananas")),
			im.Returning("id"),
		))
	})

	t.Run("fallback path matches direct construction", func(t *testing.T) {
		base := psql.Insert(
			im.IntoAs("distributors", "d", "did", "dname"),
			im.Values(psql.Arg(8, "Anvil Distribution")),
		)

		derived := base.With(
			im.OnConflict("did").DoUpdate(
				im.SetExcluded("dname"),
				im.Where(psql.Quote("d", "zipcode").NE(psql.S("21201"))),
			),
		)

		assertQueriesEqual(t, base, psql.Insert(
			im.IntoAs("distributors", "d", "did", "dname"),
			im.Values(psql.Arg(8, "Anvil Distribution")),
		))
		assertQueriesEqual(t, derived, psql.Insert(
			im.IntoAs("distributors", "d", "did", "dname"),
			im.Values(psql.Arg(8, "Anvil Distribution")),
			im.OnConflict("did").DoUpdate(
				im.SetExcluded("dname"),
				im.Where(psql.Quote("d", "zipcode").NE(psql.S("21201"))),
			),
		))
	})
}

func assertQueriesEqual(t *testing.T, got bob.Query, want bob.Query) {
	t.Helper()

	gotSQL, gotArgs, err := bob.Build(t.Context(), got)
	if err != nil {
		t.Fatalf("build got: %v", err)
	}

	wantSQL, wantArgs, err := bob.Build(t.Context(), want)
	if err != nil {
		t.Fatalf("build want: %v", err)
	}

	diff, err := testutils.QueryDiff(wantSQL, gotSQL, formatter)
	if err != nil {
		t.Fatalf("query diff error: %v", err)
	}
	if diff != "" {
		t.Fatalf("sql diff: %s", diff)
	}

	if diff := testutils.ArgsDiff(wantArgs, gotArgs); diff != "" {
		t.Fatalf("args diff: %s", diff)
	}
}
