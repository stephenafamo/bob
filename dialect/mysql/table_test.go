package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"reflect"
	"slices"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/dialect/mysql/im"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	testutils "github.com/stephenafamo/bob/test/utils"
	"github.com/stephenafamo/scan"
	_ "modernc.org/sqlite"
)

type readbackProbe struct {
	testutils.NoopExecutor
	check func(context.Context, []any) error
}

func TestInsertOwnsTableColumns(t *testing.T) {
	table := NewTable[*WithUnique, *OptionalWithUnique]("books", expr.ColsForStruct[WithUnique]("books"),
		orm.RowMapping[*WithUnique, []*WithUnique]{}, nil)
	before := table.Insert()
	want, _ := before.MustBuild(t.Context(), nil)
	changed := table.Insert(bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) { cols := q.ColumnNames(); cols[0] = "changed"; q.SetTableColumns(cols...) }))
	if got, _ := changed.MustBuild(t.Context(), nil); got == want {
		t.Fatal("the edited insert still uses the default columns")
	}
	for _, q := range []bob.Query{before, table.Insert()} {
		if got, _ := bob.MustBuild(t.Context(), nil, q); got != want {
			t.Fatalf("editing one insert changed another: got %q; want %q", got, want)
		}
	}
}

func (p readbackProbe) QueryContext(ctx context.Context, _ string, args ...any) (scan.Rows, error) {
	return nil, p.check(ctx, args)
}

func TestInsertReadbackUsesHookContext(t *testing.T) {
	type key struct{}
	table := NewTable[*WithUnique, *OptionalWithUnique]("books", expr.ColsForStruct[WithUnique]("books"),
		orm.RowMapping[*WithUnique, []*WithUnique]{}, nil, []string{"id"})
	// The unique key must resolve in the same context for INSERT and readback.
	id := bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return Arg(ctx.Value(key{})).WriteSQL(ctx, w, d, start)
	})
	table.InsertQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q InsertEditor) (context.Context, error) {
		im.Values(id, Arg("title"), Arg(1)).Edit(q)
		return context.WithValue(ctx, key{}, 17), nil
	})
	insert := table.Insert()
	readback := errors.New("reached readback with hook context")
	exec := readbackProbe{check: func(ctx context.Context, args []any) error {
		if ctx.Value(key{}) != 17 {
			t.Error("SELECT did not receive the insert hook context")
		}
		if !slices.Equal(args, []any{17}) {
			t.Fatalf("SELECT got unique keys %v; want the row returned by the hook", args)
		}
		return readback
	}}
	for _, method := range []string{"One", "All", "Cursor"} {
		t.Run(method, func(t *testing.T) {
			var err error
			switch method {
			case "One":
				_, err = insert.One(t.Context(), exec)
			case "All":
				_, err = insert.All(t.Context(), exec)
			case "Cursor":
				_, err = insert.Cursor(t.Context(), exec)
			}
			if !errors.Is(err, readback) {
				t.Fatalf("expected the readback query; got %v", err)
			}
		})
	}
}

func TestInsertReadbackUsesResolvedBatch(t *testing.T) {
	for _, method := range []string{"One", "All", "Cursor"} {
		t.Run(method, func(t *testing.T) {
			ctx := t.Context()
			db, err := bob.Open("sqlite", ":memory:")
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			// A stale lookup succeeds, but returns an old title instead of a new one.
			_, err = db.ExecContext(ctx, `CREATE TABLE books (id INTEGER PRIMARY KEY, title TEXT, author_id INTEGER);
				INSERT INTO books VALUES (1, 'old', 0), (2, 'old', 0)`)
			if err != nil {
				t.Fatal(err)
			}
			table := NewTable[*WithUnique, *OptionalWithUnique]("books", expr.ColsForStruct[WithUnique]("books"),
				orm.RowMapping[*WithUnique, []*WithUnique]{}, nil, []string{"id"})
			type resolvedKey struct{}
			mods := 0
			table.InsertQueryHooks.AppendHooks(func(ctx context.Context, _ bob.Executor, q InsertEditor) (context.Context, error) {
				q.AppendContextualModFunc(func(ctx context.Context, batch dialect.InsertEditor) (context.Context, error) {
					mods++
					batch.SetTableColumns("author_id", "title", "id")
					for i := range batch.RowCount() {
						batch.SetValue(i, 2, Arg(11+i))
					}
					return context.WithValue(ctx, resolvedKey{}, true), nil
				})
				q.AppendMapperMod(func(ctx context.Context, _ []string) (scan.BeforeFunc, scan.AfterMod) {
					if ctx.Value(resolvedKey{}) != true {
						t.Error("mapper did not receive the contextual mod's context")
					}
					return func(*scan.Row) (any, error) { return nil, nil }, func(_, row any) error {
						row.(*WithUnique).Title += " (mapped)"
						return nil
					}
				})
				return ctx, nil
			})
			insert := table.Insert(im.Values(Arg(1), Arg("new"), Arg(0)))
			if method != "One" {
				insert = insert.WithMods(im.Values(Arg(2), Arg("new"), Arg(0)))
			}
			var rows []*WithUnique
			switch method {
			case "One":
				var row *WithUnique
				row, err = insert.One(ctx, db)
				rows = []*WithUnique{row}
			case "All":
				rows, err = insert.All(ctx, db)
			case "Cursor":
				cur, err := insert.Cursor(ctx, db)
				if err != nil {
					t.Fatal(err)
				}
				defer cur.Close()
				for cur.Next() {
					row, err := cur.Get()
					if err != nil {
						t.Fatal(err)
					}
					rows = append(rows, row)
				}
				if err := cur.Err(); err != nil {
					t.Fatal(err)
				}
			}
			if err != nil {
				t.Fatal(err)
			}
			want := 2
			if method == "One" {
				want = 1
			}
			if len(rows) != want || mods != 1 {
				t.Fatalf("got %d rows and %d contextual mod calls; want %d and 1", len(rows), mods, want)
			}
			for _, row := range rows {
				if row.ID != row.AuthorID+10 || row.Title != "new (mapped)" {
					t.Fatalf("unexpected readback: %+v", row)
				}
			}
		})
	}
}

func TestInsertSetWithoutAutoIncrementRejectsBeforeWrite(t *testing.T) {
	_, err := table2.Insert(im.Set("id", Arg(7))).One(t.Context(), nil)
	if !errors.Is(err, orm.ErrCannotRetrieveRow) {
		t.Fatalf("expected ErrCannotRetrieveRow; got %v", err)
	}
}

type failBatch struct {
	bob.Executor
	args []any
	err  error
}

func (f *failBatch) ExecContext(_ context.Context, _ string, args ...any) (sql.Result, error) {
	f.args = append(f.args, args...)
	if len(f.args) == 2 {
		return nil, f.err
	}
	return nil, nil
}

func TestInsertBatchFailurePreservesQuery(t *testing.T) {
	hooks := 0
	base := table1.Insert(im.Values(Arg(1)), bob.ModFunc[dialect.InsertEditor](func(q dialect.InsertEditor) {
		q.AppendHooks(func(ctx context.Context, _ bob.Executor) (context.Context, error) { hooks++; return ctx, nil })
	}))
	batch := base.WithMods(im.Values(Arg(2)))
	for i := 1; i <= 2; i++ {
		exec := &failBatch{err: errors.New("second insert failed")}
		if _, err := batch.All(t.Context(), exec); !errors.Is(err, exec.err) {
			t.Fatal(err)
		}
		if hooks != i || !slices.Equal(exec.args, []any{1, 2}) {
			t.Fatalf("attempt %d: hooks=%d args=%v", i, hooks, exec.args)
		}
	}
	_, args := base.MustBuild(t.Context(), nil)
	if !slices.Equal(args, []any{1}) {
		t.Fatalf("base changed: %v", args)
	}
}

type WithAutoIncr struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id,autoincr"`
}

type OptionalWithAutoIncr struct {
	ID       *int    `db:"id,pk"`
	Title    *string `db:"title"`
	AuthorID *int    `db:"author_id,autoincr"`

	orm.Setter[*WithAutoIncr, dialect.InsertEditor, dialect.UpdateEditor]
}

type WithUnique struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id"`
}

type OptionalWithUnique struct {
	ID       *int    `db:"id,pk"`
	Title    *string `db:"title"`
	AuthorID *int    `db:"author_id"`

	orm.Setter[*WithUnique, dialect.InsertEditor, dialect.UpdateEditor]
}

var (
	// These fixtures only exercise inserts and unique-index lookup.
	table1 = NewTablex[*WithAutoIncr, []*WithAutoIncr, *OptionalWithAutoIncr](
		"", expr.ColsForStruct[WithAutoIncr](""), nil, orm.RowMapping[*WithAutoIncr, []*WithAutoIncr]{}, nil,
	)
	table2 = NewTablex[*WithUnique, []*WithUnique, *OptionalWithUnique](
		"books", expr.NewColumnsExpr("author_id", "title", "id").WithParent("books"), nil, orm.RowMapping[*WithUnique, []*WithUnique]{}, nil,
		[]string{"id"}, []string{"title", "author_id"},
	)
)

func TestNewTable(t *testing.T) {
	expected := "author_id"
	got := table1.autoIncrementColumn
	if got != expected {
		t.Fatalf("missing autoIncrementColumn. expected %q, got %q", expected, got)
	}

	if diff := cmp.Diff([][]int{{0}, {1, 2}}, table2.uniqueIdx); diff != "" {
		t.Fatalf("diff: %s", diff)
	}

	Insert(bob.ModFunc[dialect.InsertEditor](func(e dialect.InsertEditor) {
		if err := table1.retrievable(e); err != nil {
			t.Fatalf("table1 marked as unretrievable: %v", err)
		}

		if err := table2.retrievable(e); err != nil {
			t.Fatalf("table2 marked as unretrievable: %v", err)
		}
	}))
}

func TestUniqueSetRow(t *testing.T) {
	cases := map[string]struct {
		row  *OptionalWithUnique
		cols []string
		args []bob.Expression
	}{
		"nil": {
			row: nil,
		},
		"none fully set": {
			row: &OptionalWithUnique{Title: internal.Pointer("a title")},
		},
		"id set": {
			row:  &OptionalWithUnique{ID: internal.Pointer(10)},
			cols: []string{"id"},
			args: []bob.Expression{Arg(internal.Pointer(10))},
		},
		"title/author set": {
			row: &OptionalWithUnique{
				Title:    internal.Pointer("a title"),
				AuthorID: internal.Pointer(1),
			},
			cols: []string{"title", "author_id"},
			args: []bob.Expression{Arg(internal.Pointer("a title")), Arg(internal.Pointer(1))},
		},
		"all set": {
			row: &OptionalWithUnique{
				ID:       internal.Pointer(10),
				Title:    internal.Pointer("a title"),
				AuthorID: internal.Pointer(1),
			},
			cols: []string{"id"},
			args: []bob.Expression{Arg(internal.Pointer(10))},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rowExpr := make([]bob.Expression, 3)

			if tc.row != nil {
				if tc.row.ID != nil {
					rowExpr[0] = Arg(tc.row.ID)
				}

				if tc.row.Title != nil {
					rowExpr[1] = Arg(tc.row.Title)
				}

				if tc.row.AuthorID != nil {
					rowExpr[2] = Arg(tc.row.AuthorID)
				}
			}

			cols, args := table2.uniqueSet(t.Context(), bytes.NewBuffer(nil), table2.nonGeneratedCols, rowExpr)

			if diff := cmp.Diff(toQuote(tc.cols), table2.uniqueColNames(cols)); diff != "" {
				t.Errorf("cols: %s", diff)
			}

			if diff := cmp.Diff(tc.args, args, cmp.Comparer(compareArg)); diff != "" {
				t.Errorf("args: %s", diff)
			}
		})
	}
}

func toQuote(s []string) []bob.Expression {
	if len(s) == 0 {
		return nil
	}

	exprs := make([]bob.Expression, len(s))
	for i, v := range s {
		exprs[i] = Quote(v)
	}
	return exprs
}

func compareArg(a, b bob.Expression) bool {
	ctx := context.Background()
	buf := &bytes.Buffer{}

	aArg, aErr := a.WriteSQL(ctx, buf, dialect.Dialect, 1)
	aStr := buf.String()

	buf.Reset()

	bArg, bErr := b.WriteSQL(ctx, buf, dialect.Dialect, 1)
	bStr := buf.String()

	if aErr != nil || bErr != nil {
		return false
	}

	if aStr != bStr {
		return false
	}

	if len(aArg) != len(bArg) {
		return false
	}

	for i := range aArg {
		if !reflect.DeepEqual(aArg[i], bArg[i]) {
			return false
		}
	}

	return true
}

func TestGetInsertedUsesExplicitColumns(t *testing.T) {
	idVal := internal.Pointer(10)
	batch := insertBatch{
		first:   table2.Insert().QueryBuilder,
		columns: []string{"id", "title", "author_id"},
		rows:    []clause.Value{{Arg(idVal), Arg(internal.Pointer("test")), Arg(internal.Pointer(1))}},
	}
	query, err := table2.getInserted(t.Context(), batch, nil)
	if err != nil {
		t.Fatalf("getInserted: %v", err)
	}

	gotSQL, args, err := bob.Build(t.Context(), nil, query)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	expectedSQL := "SELECT `books`.`author_id` AS `author_id`, `books`.`title` AS `title`, `books`.`id` AS `id` FROM `books` WHERE ((`id` IN ((?))))"
	if diff, err := testutils.QueryDiff(expectedSQL, gotSQL, nil); err != nil {
		t.Fatalf("QueryDiff: %v", err)
	} else if diff != "" {
		t.Fatalf("sql diff: %s", diff)
	}
	if diff := cmp.Diff([]any{idVal}, args); diff != "" {
		t.Fatalf("args diff: %s", diff)
	}
}

func TestIsDefaultOrNull(t *testing.T) {
	cases := map[string]struct {
		value  bob.Expression // value to check
		expect bool
	}{
		"nil": {
			value:  nil,
			expect: true,
		},
		"nil Arg": {
			value:  Arg(nil),
			expect: true,
		},
		"sql.NullString": {
			value:  Arg(sql.NullString{}),
			expect: true,
		},
		"sql.NullStringTrue": {
			value:  Arg(sql.NullString{Valid: true}),
			expect: false,
		},
		"sql.Null[string]": {
			value:  Arg(sql.Null[string]{}),
			expect: true,
		},
		"sql.Null[string]True": {
			value:  Arg(sql.Null[string]{Valid: true}),
			expect: false,
		},
		"null expression": {
			value:  Raw("null"),
			expect: true,
		},
		"null expression capital": {
			value:  Raw("NULL"),
			expect: true,
		},
		"default expression": {
			value:  Raw("DEFAULT"),
			expect: true,
		},
		"int zero": {
			value:  Arg(0),
			expect: false,
		},
		"int non-zero": {
			value:  Arg(1),
			expect: false,
		},
		"string empty": {
			value:  Arg(""),
			expect: false,
		},
		"string non-empty": {
			value:  Arg("hello"),
			expect: false,
		},
	}

	b := &bytes.Buffer{}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := isDefaultOrNull(t.Context(), b, tc.value); got != tc.expect {
				t.Errorf("expected %v, got %v", tc.expect, got)
			}
		})
	}
}
