package psql

import (
	"context"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func TestBaseQueryWithDoesNotMutateOriginal(t *testing.T) {
	base := Select(
		sm.Columns("id"),
		sm.From("users"),
	)

	derived := base.With(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	if derived.Type() != bob.QueryTypeSelect {
		t.Fatalf("expected derived query type %q, got %q", bob.QueryTypeSelect, derived.Type())
	}

	baseSQL := selectToString(t, base, 0)
	if baseSQL != "SELECT \nid\nFROM users\n" {
		t.Fatalf("base query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL := selectToString(t, derived, 0)
	if derivedSQL != "SELECT \nid\nFROM users\nORDER BY id DESC\nLIMIT 10\nOFFSET 20\n" {
		t.Fatalf("derived query mismatch: %#v", derivedSQL)
	}
}

func TestViewQueryWithDoesNotMutateOriginal(t *testing.T) {
	base := someStructView.Query(
		sm.Where(Quote("id").GT(Arg(0))),
	)

	derived := base.With(
		sm.OrderBy("id").Desc(),
		sm.Limit(10),
		sm.Offset(20),
	)

	if derived.Scanner == nil {
		t.Fatal("expected derived view query to preserve scanner")
	}

	baseSQL := selectToString(t, base.BaseQuery, 1)
	if baseSQL != "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" > $0)\n" {
		t.Fatalf("base view query changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL := selectToString(t, derived.BaseQuery, 1)
	if derivedSQL != "SELECT \n\"some_struct\".\"id\" AS \"id\", \"some_struct\".\"name\" AS \"name\", \"some_struct\".\"email\" AS \"email\"\nFROM \"public\".\"some_struct\" AS \"public.some_struct\"\nWHERE (\"id\" > $0)\nORDER BY id DESC\nLIMIT 10\nOFFSET 20\n" {
		t.Fatalf("derived view query mismatch: %#v", derivedSQL)
	}
}

func BenchmarkBaseQueryApplyMain(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Select(
			sm.Columns("id", "name"),
			sm.From("users"),
			sm.Where(Quote("tenant_id").EQ(Arg(42))),
		)
		q.Apply(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkBaseQueryWithImmutableWithHelpers(b *testing.B) {
	ctx := context.Background()

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
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := someStructView.Query(
			sm.Where(Quote("id").GT(Arg(0))),
		)

		if _, _, err := asCountQuery(q.BaseQuery).Build(ctx); err != nil {
			b.Fatal(err)
		}

		q.Apply(
			sm.OrderBy("id").Desc(),
			sm.Limit(10),
			sm.Offset(20),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkViewQueryCountThenPaginateImmutableWithHelpers(b *testing.B) {
	ctx := context.Background()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := someStructView.Query(
			sm.Where(Quote("id").GT(Arg(0))),
		)

		if _, _, err := asCountQuery(q.BaseQuery).Build(ctx); err != nil {
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
