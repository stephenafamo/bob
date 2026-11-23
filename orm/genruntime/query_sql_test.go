package genruntime_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/bob/orm/genruntime"
)

func TestBoundSQLFragments(t *testing.T) {
	// One tuple expands to two. The trailing occurrences share an argument
	// and must be renumbered after the expanded tuples.
	const sql = "VALUES (?, ?) RETURNING ? + ?"
	group := genruntime.SQLGroups(2, func(i int) bob.Expression {
		return genruntime.SQLGroup("(?, ?)", func() []orm.ArgWithPosition {
			return []orm.ArgWithPosition{
				{Start: 1, Stop: 2, Expression: psql.Arg(i)},
				{Start: 4, Stop: 5, Expression: psql.Arg(i + 10)},
			}
		})
	})
	ret := strings.Index(sql, "RETURNING")
	arg := psql.Arg("again")
	fragment := genruntime.BindSQL(sql,
		orm.ArgWithPosition{Start: 7, Stop: 13, Expression: group},
		orm.ArgWithPosition{Start: ret + 10, Stop: ret + 11, Expression: arg},
		orm.ArgWithPosition{Start: len(sql) - 1, Stop: len(sql), Expression: arg})
	statement := fragment(0, len(sql))
	for _, tc := range []struct {
		expr bob.Expression
		want string
		args []any
	}{
		{statement, "VALUES ($5, $6), ($7, $8) RETURNING $9 + $10", []any{0, 10, 1, 11, "again", "again"}},
		{fragment(0, 7), "VALUES ", []any{}},
		{fragment(7, 13), "($5, $6), ($7, $8)", []any{0, 10, 1, 11}},
		{fragment(ret, len(sql)), "RETURNING $5 + $6", []any{"again", "again"}},
		// Building the same statement again starts from the original byte range.
		{statement, "VALUES ($5, $6), ($7, $8) RETURNING $9 + $10", []any{0, 10, 1, 11, "again", "again"}},
	} {
		var out strings.Builder
		args, err := tc.expr.WriteSQL(t.Context(), &out, dialect.Dialect, 5)
		if err != nil || out.String() != tc.want || !reflect.DeepEqual(args, tc.args) {
			t.Errorf("got %q %v, %v; want %q %v", out.String(), args, err, tc.want, tc.args)
		}
	}
	var out strings.Builder
	if _, err := fragment(8, 13).WriteSQL(t.Context(), &out, dialect.Dialect, 1); err == nil {
		t.Error("expected error for a fragment starting inside a tuple")
	}
}
