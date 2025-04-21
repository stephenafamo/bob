package mysql

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"reflect"
	"testing"

	"github.com/aarondl/opt/omit"
	"github.com/antlr4-go/antlr/v4"
	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/mysql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
	testutils "github.com/stephenafamo/bob/test/utils"
	mysqlparser "github.com/stephenafamo/sqlparser/mysql"
)

type WithAutoIncr struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id,autoincr"`
}

func (w *WithAutoIncr) PrimaryKeyVals() bob.Expression {
	return Arg(w.ID)
}

type OptionalWithAutoIncr struct {
	ID       omit.Val[int]    `db:"id,pk"`
	Title    omit.Val[string] `db:"title"`
	AuthorID omit.Val[int]    `db:"author_id,autoincr"`

	orm.Setter[*WithAutoIncr, *dialect.InsertQuery, *dialect.UpdateQuery]
}

type WithUnique struct {
	ID       int    `db:"id,pk"`
	Title    string `db:"title"`
	AuthorID int    `db:"author_id"`
}

func (w *WithUnique) PrimaryKeyVals() bob.Expression {
	return Arg(w.ID)
}

type OptionalWithUnique struct {
	ID       omit.Val[int]    `db:"id,pk"`
	Title    omit.Val[string] `db:"title"`
	AuthorID omit.Val[int]    `db:"author_id"`

	orm.Setter[*WithUnique, *dialect.InsertQuery, *dialect.UpdateQuery]
}

type WithoutAutoIncr struct {
	TranslationKey string `db:"translation_key"`
	Language       string `db:"language"`
}

func (o *WithoutAutoIncr) PrimaryKeyVals() bob.Expression {
	return ArgGroup(
		o.TranslationKey,
		o.Language,
	)
}

type WithoutAutoIncrSetter struct {
	TranslationKey omit.Val[string] `db:"translation_key"`
	Language       omit.Val[string] `db:"language"`

	orm.Setter[*WithoutAutoIncr, *dialect.InsertQuery, *dialect.UpdateQuery]
}

func (s WithoutAutoIncrSetter) Apply(q *dialect.InsertQuery) {

	q.AppendInsertExprs(s.Expressions("without_auto_incr"))

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
		vals := make([]bob.Expression, 2)
		if s.TranslationKey.IsUnset() {
			vals[0] = Raw("DEFAULT")
		} else {
			vals[0] = Arg(s.TranslationKey)
		}

		if s.Language.IsUnset() {
			vals[1] = Raw("DEFAULT")
		} else {
			vals[1] = Arg(s.Language)
		}
		return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")

	}))
}

func (s WithoutAutoIncrSetter) Expressions(prefix ...string) []bob.Expression {
	exprs := make([]bob.Expression, 0, 3)

	if !s.TranslationKey.IsUnset() {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "translation_key")...),
			Arg(s.TranslationKey),
		}})
	}

	if !s.Language.IsUnset() {
		exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
			Quote(append(prefix, "language")...),
			Arg(s.Language),
		}})
	}

	return exprs
}

var (
	table1 = NewTablex[*WithAutoIncr, []*WithAutoIncr, *OptionalWithAutoIncr]("")
	table2 = NewTablex[*WithUnique, []*WithUnique, *OptionalWithUnique](
		"", []string{"id"}, []string{"title", "author_id"},
	)
	table3 = NewTablex[*WithoutAutoIncr, []*WithoutAutoIncr, *WithoutAutoIncrSetter](
		"without_auto_incr", []string{"translation_key", "language"})
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

	if table1.unretrievable {
		t.Fatalf("table1 marked as unretrievable")
	}

	if table2.unretrievable {
		t.Fatalf("table2 marked as unretrievable")
	}
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
			row: &OptionalWithUnique{Title: omit.From("a title")},
		},
		"id set": {
			row:  &OptionalWithUnique{ID: omit.From(10)},
			cols: []string{"id"},
			args: []bob.Expression{Arg(omit.From(10))},
		},
		"title/author set": {
			row: &OptionalWithUnique{
				Title:    omit.From("a title"),
				AuthorID: omit.From(1),
			},
			cols: []string{"title", "author_id"},
			args: []bob.Expression{Arg(omit.From("a title")), Arg(omit.From(1))},
		},
		"all set": {
			row: &OptionalWithUnique{
				ID:       omit.From(10),
				Title:    omit.From("a title"),
				AuthorID: omit.From(1),
			},
			cols: []string{"id"},
			args: []bob.Expression{Arg(omit.From(10))},
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			rowExpr := make([]bob.Expression, 3)

			if tc.row != nil {
				if tc.row.ID.IsSet() {
					rowExpr[0] = expr.Join{Sep: " = ", Exprs: []bob.Expression{
						Quote(append([]string{"prefix"}, "a")...),
						Arg(tc.row.ID),
					}}
				}

				if tc.row.Title.IsSet() {
					rowExpr[1] = expr.Join{Sep: " = ", Exprs: []bob.Expression{
						Quote(append([]string{"prefix"}, "b")...),
						Arg(tc.row.Title),
					}}
				}

				if tc.row.AuthorID.IsSet() {
					rowExpr[2] = expr.Join{Sep: " = ", Exprs: []bob.Expression{
						Quote(append([]string{"prefix"}, "c")...),
						Arg(tc.row.AuthorID),
					}}
				}
			}

			cols, args := table2.uniqueSet(bytes.NewBuffer(nil), rowExpr)

			if diff := cmp.Diff(toQuote(tc.cols), table2.uniqueColNames(cols)); diff != "" {
				t.Errorf("cols: %s", diff)
			}

			if diff := cmp.Diff(tc.args, args, cmp.Comparer(compareArg)); diff != "" {
				t.Errorf("args: %s", diff)
			}
		})
	}
}

func TestRetrievalWithoutAutoIncr(t *testing.T) {

	if table3.unretrievable {
		t.Fatalf("table3 marked as unretrievable")
	}

	insertQ := table3.Insert(
		WithoutAutoIncrSetter{
			TranslationKey: omit.From("a translation key"),
			Language:       omit.From("a language"),
		},
		WithoutAutoIncrSetter{
			TranslationKey: omit.From("another translation key"),
			Language:       omit.From("another language"),
		},
	)

	getInsertedQ, err := insertQ.getInserted(insertQ.Expression.InsertExprs, []sql.Result{})
	if err != nil {
		t.Fatal(err)
	}

	testutils.RunTests(t, map[string]testutils.Testcase{
		"insert": {
			Query: insertQ,
			ExpectedSQL: `INSERT INTO ` + "`without_auto_incr`" + ` (` + "`translation_key`" + `, ` + "`language`" + `)
				VALUES (?, ?), (?, ?)`,
			ExpectedArgs: []any{
				omit.From("a translation key"),
				omit.From("a language"),
				omit.From("another translation key"),
				omit.From("another language"),
			},
		},
		"get inserted": {
			Query:       getInsertedQ,
			ExpectedSQL: `SELECT * FROM ` + "`without_auto_incr`" + ` AS ` + "`without_auto_incr`" + ` WHERE (((` + "`translation_key`" + ` = ?) AND (` + "`language`" + ` = ?)) OR ((` + "`translation_key`" + ` = ?) AND (` + "`language`" + ` = ?)))`,
			ExpectedArgs: []any{
				omit.From("a translation key"),
				omit.From("a language"),
				omit.From("another translation key"),
				omit.From("another language"),
			},
		},
	}, formatter)

}

func toQuote(s []string) []Expression {
	if len(s) == 0 {
		return nil
	}

	exprs := make([]Expression, len(s))
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

func formatter(s string) (string, error) {
	input := antlr.NewInputStream(s)
	lexer := mysqlparser.NewMySqlLexer(input)
	stream := antlr.NewCommonTokenStream(lexer, 0)
	p := mysqlparser.NewMySqlParser(stream)

	el := &errorListener{}
	p.AddErrorListener(el)

	tree := p.Root()
	if el.err != "" {
		return "", errors.New(el.err)
	}

	return tree.GetText(), nil
}

type errorListener struct {
	*antlr.DefaultErrorListener

	err string
}

func (el *errorListener) SyntaxError(recognizer antlr.Recognizer, offendingSymbol any, line, column int, msg string, e antlr.RecognitionException) {
	el.err = msg
}
