package psql

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/expr"
)

func TestImmutableSelectQueryWithDoesNotMutateOriginal(t *testing.T) {
	base := Select(
		sm.Columns("id"),
		sm.From("users"),
	).With()

	derived := base.With(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	if derived.Type() != bob.QueryTypeSelect {
		t.Fatalf("expected derived query type %q, got %q", bob.QueryTypeSelect, derived.Type())
	}

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "SELECT \nid\nFROM users\n" {
		t.Fatalf("base query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "SELECT \nid\nFROM users\nORDER BY id DESC\nLIMIT 10\nOFFSET 20\n" {
		t.Fatalf("derived query mismatch: %#v", derivedSQL)
	}
}

func TestImmutableViewQueryWithDoesNotMutateOriginal(t *testing.T) {
	base := someStructView.Query(
		sm.Where(Quote("id").GT(Arg(0))),
	).With()

	derived := base.With(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	if derived.Scanner == nil {
		t.Fatal("expected derived view query to preserve scanner")
	}

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" > $1)\n" {
		t.Fatalf("base view query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" > $1)\nORDER BY id DESC\nLIMIT 10\nOFFSET 20\n" {
		t.Fatalf("derived view query mismatch: %#v", derivedSQL)
	}
}

func TestImmutableSelectQueryApplyDoesNotMutateOriginal(t *testing.T) {
	base := Select(
		sm.Columns("id"),
		sm.From("users"),
	)

	derived := base.Apply(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "SELECT \nid\nFROM users\n" {
		t.Fatalf("base query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "SELECT \nid\nFROM users\nORDER BY id DESC\nLIMIT 10\nOFFSET 20\n" {
		t.Fatalf("derived query mismatch: %#v", derivedSQL)
	}
}

func TestImmutableSelectQueryApplyFallbackDoesNotMutateOriginal(t *testing.T) {
	base := Select(
		sm.Columns("id", "name"),
		sm.From("users"),
	)

	derived := base.Apply(
		sm.Distinct(),
		sm.Union(Select(
			sm.Columns("id", "name"),
			sm.From("admins"),
		)),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "SELECT \nid, name\nFROM users\n" {
		t.Fatalf("base query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, derivedArgs, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expected := Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Distinct(),
		sm.Union(Select(
			sm.Columns("id", "name"),
			sm.From("admins"),
		)),
	)

	expectedSQL, expectedArgs, err := expected.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != expectedSQL {
		t.Fatalf("derived fallback query mismatch: got %#v want %#v", derivedSQL, expectedSQL)
	}
	if len(derivedArgs) != len(expectedArgs) {
		t.Fatalf("derived fallback args mismatch: got %d want %d", len(derivedArgs), len(expectedArgs))
	}
}

func TestImmutableSelectQueryApplyCombinedDoesNotMutateOriginal(t *testing.T) {
	base := Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Limit(100),
		sm.OrderBy("id"),
	)

	derived := base.Apply(
		sm.Union(Select(
			sm.Columns("id", "name"),
			sm.From("admins"),
			sm.Limit(10),
			sm.OrderBy("id"),
		)),
		sm.OrderCombined("id"),
		sm.LimitCombined(1000),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "SELECT \nid, name\nFROM users\nORDER BY id\nLIMIT 100\n" {
		t.Fatalf("base query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, derivedArgs, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expected := Select(
		sm.Columns("id", "name"),
		sm.From("users"),
		sm.Limit(100),
		sm.OrderBy("id"),
		sm.Union(Select(
			sm.Columns("id", "name"),
			sm.From("admins"),
			sm.Limit(10),
			sm.OrderBy("id"),
		)),
		sm.OrderCombined("id"),
		sm.LimitCombined(1000),
	)

	expectedSQL, expectedArgs, err := expected.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != expectedSQL {
		t.Fatalf("derived combined query mismatch: got %#v want %#v", derivedSQL, expectedSQL)
	}
	if len(derivedArgs) != len(expectedArgs) {
		t.Fatalf("derived combined args mismatch: got %d want %d", len(derivedArgs), len(expectedArgs))
	}
}

func TestImmutableViewQueryApplyDoesNotMutateOriginal(t *testing.T) {
	base := someStructView.Query(
		sm.Where(Quote("id").GT(Arg(0))),
	)

	derived := base.Apply(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" > $1)\n" {
		t.Fatalf("base view query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" > $1)\nORDER BY id DESC\nLIMIT 10\nOFFSET 20\n" {
		t.Fatalf("derived view query mismatch: %#v", derivedSQL)
	}
}

func TestViewSelectQueryHooksUseImmutableSelectQuery(t *testing.T) {
	view := NewView[*someStruct, bob.Expression]("public", "some_struct", expr.ColsForStruct[someStruct]("some_struct"))

	var hookSQL string
	view.SelectQueryHooks.AppendHooks(func(ctx context.Context, exec bob.Executor, q *SelectQuery) (context.Context, error) {
		sql, _, err := q.Build(ctx)
		if err != nil {
			return ctx, err
		}
		hookSQL = sql
		return context.WithValue(ctx, "view-hook-ran", true), nil
	})

	query := view.Query(
		sm.Where(Quote("id").EQ(Arg(1))),
		sm.OrderBy("name"),
	)

	ctx, err := query.RunHooks(t.Context(), nil)
	if err != nil {
		t.Fatal(err)
	}

	if got := ctx.Value("view-hook-ran"); got != true {
		t.Fatalf("expected hook marker in context, got %#v", got)
	}

	expected := "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" = $1)\nORDER BY name\n"
	if hookSQL != expected {
		t.Fatalf("unexpected hook SQL: %#v", hookSQL)
	}
}

func BenchmarkBaseQueryApplyMain(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Select(
			sm.Columns("id", "name"),
			sm.From("users"),
			sm.Where(Quote("tenant_id").EQ(Arg(42))),
		)
		q = q.Apply(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBaseQueryImmutableNativeHotPath(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Select(
			sm.Columns("id", "name"),
			sm.From("users"),
			sm.Where(Quote("tenant_id").EQ(Arg(42))),
		)
		derived := q.With(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		if _, _, err := derived.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkViewQueryCountThenPaginateApplyMain(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := someStructView.Query(
			sm.Where(Quote("id").GT(Arg(0))),
		)

		if _, _, err := q.Query.derivedSelectQuery.AsCount().Build(ctx); err != nil {
			b.Fatal(err)
		}

		q = q.Apply(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkViewQueryCountThenPaginateImmutableNativeHotPath(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := someStructView.Query(
			sm.Where(Quote("id").GT(Arg(0))),
		)

		if _, _, err := q.Query.derivedSelectQuery.AsCount().Build(ctx); err != nil {
			b.Fatal(err)
		}

		derived := q.With(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		if _, _, err := derived.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
