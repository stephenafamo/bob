package psql

import (
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql/dm"
	"github.com/stephenafamo/bob/dialect/psql/im"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/bob/dialect/psql/um"
)

func TestUpdateApplyDoesNotMutateOriginalFromLegacyWithCase(t *testing.T) {
	base := Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
	)

	derived := base.Apply(
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

func TestDeleteApplyDoesNotMutateOriginalFromLegacyWithCase(t *testing.T) {
	base := Delete(
		dm.From("films"),
	)

	derived := base.Apply(
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

func TestInsertApplyDoesNotMutateOriginalFromLegacyWithCase(t *testing.T) {
	base := Insert(
		im.Into("films"),
		im.Values(Arg("UA502", "Bananas")),
	)

	derived := base.Apply(
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

func TestUpdateApplyDoesNotMutateOriginal(t *testing.T) {
	base := Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Dramatic"),
	)

	derived := base.Apply(
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

func TestDeleteApplyDoesNotMutateOriginal(t *testing.T) {
	base := Delete(
		dm.From("films"),
	)

	derived := base.Apply(
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

func TestUpdateApplyDoesNotDuplicateOnly(t *testing.T) {
	base := Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Drama"),
		um.Only(),
	)

	derived := base.Apply(um.Only())

	sql, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sql != "UPDATE ONLY films SET\n\"kind\" = $1" {
		t.Fatalf("unexpected update SQL: %#v", sql)
	}
}

func TestDeleteApplyDoesNotDuplicateOnly(t *testing.T) {
	base := Delete(
		dm.From("films"),
		dm.Only(),
	)

	derived := base.Apply(dm.Only())

	sql, _, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if sql != "DELETE FROM ONLY films" {
		t.Fatalf("unexpected delete SQL: %#v", sql)
	}
}

func TestInsertApplyDoesNotMutateOriginal(t *testing.T) {
	base := Insert(
		im.Into("films"),
		im.Values(Arg("UA502", "Bananas")),
	)

	derived := base.Apply(
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

func TestUpdateApplySupportsCommonDerivedMods(t *testing.T) {
	base := Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Drama"),
	)

	derived := base.Apply(
		um.With("recent").As(Select(
			sm.Columns("id"),
			sm.From("recent_films"),
		)),
		um.Recursive(true),
		um.Only(),
		um.TableAs("films", "f"),
		um.Set(Quote("rating").EQ(Arg("PG"))),
		um.From("producers"),
		um.LeftJoin("studios").As("s").OnEQ(Quote("s", "id"), Quote("f", "studio_id")),
		um.Where(Quote("f", "id").EQ(Arg(1))),
		um.Returning("f.id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "UPDATE films SET\n\"kind\" = $1" {
		t.Fatalf("base update changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, derivedArgs, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expected := Update(
		um.Table("films"),
		um.SetCol("kind").ToArg("Drama"),
		um.With("recent").As(Select(
			sm.Columns("id"),
			sm.From("recent_films"),
		)),
		um.Recursive(true),
		um.Only(),
		um.TableAs("films", "f"),
		um.Set(Quote("rating").EQ(Arg("PG"))),
		um.From("producers"),
		um.LeftJoin("studios").As("s").OnEQ(Quote("s", "id"), Quote("f", "studio_id")),
		um.Where(Quote("f", "id").EQ(Arg(1))),
		um.Returning("f.id"),
	)

	expectedSQL, expectedArgs, err := expected.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != expectedSQL {
		t.Fatalf("derived update mismatch: got %#v want %#v", derivedSQL, expectedSQL)
	}
	if len(derivedArgs) != len(expectedArgs) {
		t.Fatalf("derived update args mismatch: got %d want %d", len(derivedArgs), len(expectedArgs))
	}
}

func TestDeleteApplySupportsCommonDerivedMods(t *testing.T) {
	base := Delete(
		dm.From("films"),
	)

	derived := base.Apply(
		dm.With("recent").As(Select(
			sm.Columns("id"),
			sm.From("recent_films"),
		)),
		dm.Recursive(true),
		dm.Only(),
		dm.FromAs("films", "f"),
		dm.Using("producers"),
		dm.LeftJoin("studios").As("s").OnEQ(Quote("s", "id"), Quote("f", "studio_id")),
		dm.Where(Quote("f", "id").EQ(Arg(1))),
		dm.Returning("f.id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "DELETE FROM films" {
		t.Fatalf("base delete changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, derivedArgs, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expected := Delete(
		dm.From("films"),
		dm.With("recent").As(Select(
			sm.Columns("id"),
			sm.From("recent_films"),
		)),
		dm.Recursive(true),
		dm.Only(),
		dm.FromAs("films", "f"),
		dm.Using("producers"),
		dm.LeftJoin("studios").As("s").OnEQ(Quote("s", "id"), Quote("f", "studio_id")),
		dm.Where(Quote("f", "id").EQ(Arg(1))),
		dm.Returning("f.id"),
	)

	expectedSQL, expectedArgs, err := expected.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != expectedSQL {
		t.Fatalf("derived delete mismatch: got %#v want %#v", derivedSQL, expectedSQL)
	}
	if len(derivedArgs) != len(expectedArgs) {
		t.Fatalf("derived delete args mismatch: got %d want %d", len(derivedArgs), len(expectedArgs))
	}
}

func TestInsertApplySupportsCommonDerivedMods(t *testing.T) {
	base := Insert(
		im.Into("films"),
	)

	derived := base.Apply(
		im.With("recent").As(Select(
			sm.Columns("id"),
			sm.From("recent_films"),
		)),
		im.Recursive(true),
		im.IntoAs("films", "f", "code", "title"),
		im.OverridingUser(),
		im.Rows(
			[]bob.Expression{Arg("UA502"), Arg("Bananas")},
			[]bob.Expression{Arg("UA503"), Arg("Grapes")},
		),
		im.OnConflict("code").DoUpdate(
			im.SetExcluded("title"),
		),
		im.Returning("f.id"),
	)

	baseSQL, _, err := base.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if baseSQL != "INSERT INTO films\nDEFAULT VALUES\n" {
		t.Fatalf("base insert changed unexpectedly: %#v", baseSQL)
	}

	derivedSQL, derivedArgs, err := derived.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}

	expected := Insert(
		im.Into("films"),
		im.With("recent").As(Select(
			sm.Columns("id"),
			sm.From("recent_films"),
		)),
		im.Recursive(true),
		im.IntoAs("films", "f", "code", "title"),
		im.OverridingUser(),
		im.Rows(
			[]bob.Expression{Arg("UA502"), Arg("Bananas")},
			[]bob.Expression{Arg("UA503"), Arg("Grapes")},
		),
		im.OnConflict("code").DoUpdate(
			im.SetExcluded("title"),
		),
		im.Returning("f.id"),
	)

	expectedSQL, expectedArgs, err := expected.Build(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if derivedSQL != expectedSQL {
		t.Fatalf("derived insert mismatch: got %#v want %#v", derivedSQL, expectedSQL)
	}
	if len(derivedArgs) != len(expectedArgs) {
		t.Fatalf("derived insert args mismatch: got %d want %d", len(derivedArgs), len(expectedArgs))
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
		q = q.Apply(
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
		derived := q.Apply(
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
		q = q.Apply(
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
		derived := q.Apply(
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
		q = q.Apply(
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
		derived := q.Apply(
			im.Returning("id"),
		)

		if _, _, err := derived.Build(ctx); err != nil {
			b.Fatal(err)
		}
	}
}
