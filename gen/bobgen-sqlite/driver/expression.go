package driver

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

func NewExecQuery[Q bob.Expression, R any](dialect bob.Dialect, queryType bob.QueryType, expr orm.ModExpression[Q]) orm.ModExecQuery[Q] {
	return orm.ModExecQuery[Q]{
		ExecQuery: orm.ExecQuery[orm.ModExpression[Q]]{
			BaseQuery: bob.BaseQuery[orm.ModExpression[Q]]{
				Expression: expr,
				Dialect:    dialect,
				QueryType:  queryType,
			},
		},
	}
}

func NewQuery[Q bob.Expression, T any, Ts ~[]T](dialect bob.Dialect, queryType bob.QueryType, expr orm.ModExpression[Q], m scan.Mapper[T]) orm.ModQuery[Q, T, Ts] {
	return orm.ModQuery[Q, T, Ts]{
		Query: orm.Query[orm.ModExpression[Q], T, Ts]{
			ExecQuery: orm.ExecQuery[orm.ModExpression[Q]]{
				BaseQuery: bob.BaseQuery[orm.ModExpression[Q]]{
					Expression: expr,
					Dialect:    dialect,
					QueryType:  queryType,
				},
			},
		},
	}
}

// func (q queryExpression[E]) expr(from, to int) bob.Expression {
// 	return bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
// 		args := []any{}
// 		for _, arg := range q.args {
// 			if arg.start < from || arg.start > to {
// 				continue
// 			}
//
// 			// Ideally, there should be no arg that does not start within the range,
// 			// but stops after the range
// 			if arg.stop > to {
// 				return nil, fmt.Errorf("arg key stop(%d) is greater than to(%d)", arg.stop, to)
// 			}
//
// 			// Write the query before the arg
// 			fmt.Fprint(w, q.query[from:arg.start])
// 			from = arg.stop + 1
//
// 			arg, err := bob.Express(ctx, w, d, start, arg.val)
// 			if err != nil {
// 				return nil, err
// 			}
// 			args = append(args, arg...)
//
// 			start += len(arg)
// 		}
//
// 		// write the rest of the query
// 		fmt.Fprint(w, q.query[from:to+1])
// 		return args, nil
// 	})
// }
