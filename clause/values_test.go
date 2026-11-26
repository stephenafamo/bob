package clause_test

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
)

func TestValuesArgumentOffsets(t *testing.T) {
	values := clause.Values{Vals: []clause.Value{
		{expr.Raw("DEFAULT"), expr.ArgGroup(1, 2)},
		{expr.Arg(3), expr.Raw("NULL")},
	}}
	var sql strings.Builder
	args, err := values.WriteSQL(t.Context(), &sql, dialect.Dialect, 4)
	if err != nil || sql.String() != "VALUES (DEFAULT, ($4, $5)), ($6, NULL)" || !slices.Equal(args, []any{1, 2, 3}) {
		t.Fatalf("got %q, %v, %v", sql.String(), args, err)
	}
}

func TestAppendRowsSkipsEmptyAndIsolatesAdjacentRows(t *testing.T) {
	rows := [][]bob.Expression{{expr.Arg(1)}, {}, {expr.Arg(2)}}
	var v clause.Values
	v.AppendRows(rows...)
	rows[2][0] = expr.Arg(99)
	v.Vals[0] = append(v.Vals[0], expr.Arg(3))
	var sql strings.Builder
	args, err := v.WriteSQL(t.Context(), &sql, dialect.Dialect, 1)
	if err != nil || sql.String() != "VALUES ($1, $2), ($3)" || !slices.Equal(args, []any{1, 3, 2}) {
		t.Fatalf("got %q %v %v", sql.String(), args, err)
	}
}

func TestValuesPartialArgumentsOnError(t *testing.T) {
	wantErr := errors.New("invalid cell")
	row := clause.Value{expr.Arg(2), bob.ExpressionFunc(func(context.Context, io.StringWriter, bob.Dialect, int) ([]any, error) {
		return nil, wantErr
	})}
	var sql strings.Builder
	args, err := row.WriteSQL(t.Context(), &sql, dialect.Dialect, 1)
	if !errors.Is(err, wantErr) || !slices.Equal(args, []any{2}) {
		t.Fatalf("standalone row: got %v, %v; want the completed cell", args, err)
	}
	values := clause.Values{Vals: []clause.Value{{expr.Arg(1)}, row}}
	args, err = values.WriteSQL(t.Context(), &sql, dialect.Dialect, 1)
	if !errors.Is(err, wantErr) || !slices.Equal(args, []any{1}) {
		t.Fatalf("row list: got %v, %v; want only the completed row", args, err)
	}
}
