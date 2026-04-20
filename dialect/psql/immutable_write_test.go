package psql

import (
	"testing"

	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

func TestUpdateWithDoesNotMutateOriginal(t *testing.T) {
	base := Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
	)

	derived := base.With(
		um.Where(Quote("kind").EQ(Arg("Drama"))),
		um.Returning("id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "UPDATE films SET\n\"kind\" = $1" {
		t.Fatalf("base update changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "UPDATE films SET\n\"kind\" = $1\nWHERE (\"kind\" = $2)\nRETURNING id" {
		t.Fatalf("derived update mismatch: %#v", derivedSQL)
	}
}

func TestDeleteWithDoesNotMutateOriginal(t *testing.T) {
	base := Delete(
		dm.From("films"),
	)

	derived := base.With(
		dm.Where(Quote("kind").EQ(Arg("Drama"))),
		dm.Returning("id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "DELETE FROM films" {
		t.Fatalf("base delete changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "DELETE FROM films\nWHERE (\"kind\" = $1)\nRETURNING id" {
		t.Fatalf("derived delete mismatch: %#v", derivedSQL)
	}
}

func TestInsertWithDoesNotMutateOriginal(t *testing.T) {
	base := Insert(
		im.Into("films"),
		im.Values(Arg("UA502", "Bananas")),
	)

	derived := base.With(
		im.Returning("id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "INSERT INTO films\nVALUES ($1, $2)\n" {
		t.Fatalf("base insert changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != "INSERT INTO films\nVALUES ($1, $2)\nRETURNING id\n" {
		t.Fatalf("derived insert mismatch: %#v", derivedSQL)
	}
}

func BenchmarkUpdateQueryApplyMain(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Update(
			um.Table("films"),
			um.SetCol("kind").ToArg("Dramatic"),
		)
		q.Apply(
			um.Where(Quote("kind").EQ(Arg("Drama"))),
			um.Returning("id"),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUpdateQueryImmutableNativeHotPath(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Update(
			um.Table("films"),
			um.SetCol("kind").ToArg("Dramatic"),
		)
		derived := q.With(
			um.Where(Quote("kind").EQ(Arg("Drama"))),
			um.Returning("id"),
		)

		if _, _, err := derived.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeleteQueryApplyMain(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Delete(
			dm.From("films"),
		)
		q.Apply(
			dm.Where(Quote("kind").EQ(Arg("Drama"))),
			dm.Returning("id"),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDeleteQueryImmutableNativeHotPath(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Delete(
			dm.From("films"),
		)
		derived := q.With(
			dm.Where(Quote("kind").EQ(Arg("Drama"))),
			dm.Returning("id"),
		)

		if _, _, err := derived.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertQueryApplyMain(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Insert(
			im.Into("films"),
			im.Values(Arg("UA502", "Bananas")),
		)
		q.Apply(
			im.Returning("id"),
		)

		if _, _, err := q.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInsertQueryImmutableNativeHotPath(b *testing.B) {
	ctx := b.Context()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		q := Insert(
			im.Into("films"),
			im.Values(Arg("UA502", "Bananas")),
		)
		derived := q.With(
			im.Returning("id"),
		)

		if _, _, err := derived.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
