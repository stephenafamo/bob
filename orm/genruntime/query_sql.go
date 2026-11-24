package genruntime

import (
	"context"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

// WriteSQL renders a complete statement or tuple. All occurrences are in
// byte order, so a full render does not need the fragment iterator.
func WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, sql string, occurrences []orm.ArgWithPosition) ([]any, error) {
	return writeSQL(ctx, w, d, start, sql, 0, len(sql), occurrences)
}

func writeSQL(
	ctx context.Context,
	w io.StringWriter,
	d bob.Dialect,
	start int,
	sql string,
	from, to int,
	occurrences []orm.ArgWithPosition,
) ([]any, error) {
	if from < 0 || from > to || to > len(sql) {
		return nil, fmt.Errorf("invalid SQL fragment [%d:%d] for statement of length %d", from, to, len(sql))
	}
	args := []any{}
	if len(occurrences) > 1 {
		args = make([]any, 0, len(occurrences))
	}
	cursor := from
	for _, arg := range occurrences {
		if arg.Start < cursor || arg.Stop < arg.Start || arg.Stop > to {
			return nil, fmt.Errorf("arg %q [%d:%d] outside remaining SQL fragment [%d:%d]", arg.Name, arg.Start, arg.Stop, cursor, to)
		}
		w.WriteString(sql[cursor:arg.Start])
		values, err := bob.Express(ctx, w, d, start+len(args), arg.Expression)
		if err != nil {
			return nil, err
		}
		args = append(args, values...)
		cursor = arg.Stop
	}
	w.WriteString(sql[cursor:to])
	return args, nil
}

// SQLTuples binds fixed-width rows in input order. The spans contain the
// original SQL before, between and after the placeholders of one row.
func SQLTuples(rows int, appendRow func([]any, int) []any, spans ...string) bob.Expression {
	args := make([]any, 0, rows*(len(spans)-1))
	for i := range rows {
		args = appendRow(args, i)
	}
	return bob.ExpressionFunc(func(_ context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		for i := range rows {
			if i != 0 {
				w.WriteString(", ")
			}
			w.WriteString(spans[0])
			for _, span := range spans[1:] {
				d.WriteArg(w, start)
				start++
				w.WriteString(span)
			}
		}
		return args, nil
	})
}

// BindSQL binds the argument occurrences for a statement. The returned
// function renders any half-open byte range using the same bindings.
func BindSQL(sql string, args ...orm.ArgWithPosition) func(int, int) bob.Expression {
	return func(from, to int) bob.Expression {
		// Slice only at argument boundaries. A partially overlapping
		// argument stays in the range so writeSQL reports it as an error.
		occurrences := args
		for len(occurrences) > 0 && occurrences[0].Stop <= from {
			occurrences = occurrences[1:]
		}
		for n := len(occurrences); n > 0 && occurrences[n-1].Start >= to; n-- {
			occurrences = occurrences[:n-1]
		}
		return bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
			return writeSQL(ctx, w, d, start, sql, from, to, occurrences)
		})
	}
}

// SQLGroup binds a tuple's child expressions when the enclosing query is built.
func SQLGroup(sql string, args func() []orm.ArgWithPosition) bob.Expression {
	return bob.ExpressionFunc(func(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
		return WriteSQL(ctx, w, d, start, sql, args())
	})
}

// SQLGroups collects the tuples of an expanding argument in input order.
func SQLGroups(n int, group func(int) bob.Expression) bob.Expression {
	groups := make([]bob.Expression, n)
	for i := range groups {
		groups[i] = group(i)
	}
	return expr.Join{Exprs: groups, Sep: ", "}
}

// NewModExecQuery connects the original SQL expression and its dialect builder.
func NewModExecQuery[Q, E bob.Expression](e E, d bob.Dialect, typ bob.QueryType,
	build func(...bob.Mod[Q]) bob.BaseQuery[Q], mod bob.ModFunc[Q],
) *orm.ModExecQuery[Q, E] {
	return &orm.ModExecQuery[Q, E]{
		ExecQuery: orm.ExecQuery[E]{BaseQuery: bob.BaseQuery[E]{Expression: e, Dialect: d, QueryType: typ}},
		Build:     build,
		Mod:       mod,
	}
}

// NewModQuery also attaches the mapper used by One, All, Cursor and Each.
func NewModQuery[Tr bob.Transformer[T, Ts], T, Ts any, Q, E bob.Expression](e E, d bob.Dialect, typ bob.QueryType,
	build func(...bob.Mod[Q]) bob.BaseQuery[Q], mod bob.ModFunc[Q], mapper scan.Mapper[T],
) *orm.ModQuery[Q, E, T, Ts, Tr] {
	q := NewModExecQuery(e, d, typ, build, mod)
	return &orm.ModQuery[Q, E, T, Ts, Tr]{
		Query: orm.Query[E, T, Ts, Tr]{ExecQuery: q.ExecQuery, Scanner: mapper},
		Build: build,
		Mod:   mod,
	}
}
