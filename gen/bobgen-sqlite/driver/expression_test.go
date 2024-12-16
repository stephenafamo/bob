package driver

// var _ bob.Query = queryExpression[any]{}
//
// func TestQueryExpression(t *testing.T) {
// 	type expressions struct {
// 		from, to     int
// 		expectedSQL  string
// 		expectedArgs []any
// 	}
//
// 	cases := map[string]struct {
// 		expectedSQL  string
// 		expectedArgs []any
// 		query        queryExpression
// 		expressions  []expressions
// 	}{
// 		"simple": {
// 			expectedSQL:  "SELECT * FROM users WHERE id = ?1 AND email = ?2 OR age = ?3",
// 			expectedArgs: []any{1, "email", 100},
// 			query: queryExpression{
// 				query: "SELECT * FROM users WHERE id = ?1 AND email = ?2 OR age = ?3",
// 				args: []inQueryArg{
// 					{start: 31, stop: 32, val: expr.Arg(1)},
// 					{start: 46, stop: 47, val: expr.Arg("email")},
// 					{start: 58, stop: 59, val: expr.Arg(100)},
// 				},
// 			},
// 			expressions: []expressions{
// 				{
// 					from: 26, to: 32,
// 					expectedSQL:  "id = ?1",
// 					expectedArgs: []any{1},
// 				},
// 				{
// 					from: 38, to: 47,
// 					expectedSQL:  "email = ?1",
// 					expectedArgs: []any{"email"},
// 				},
// 				{
// 					from: 52, to: 59,
// 					expectedSQL:  "age = ?1",
// 					expectedArgs: []any{100},
// 				},
// 				{
// 					from: 20, to: 59,
// 					expectedSQL:  "WHERE id = ?1 AND email = ?2 OR age = ?3",
// 					expectedArgs: []any{1, "email", 100},
// 				},
// 			},
// 		},
// 		"IN": {
// 			expectedSQL:  "SELECT * FROM users WHERE id IN (?1, ?2, ?3) AND email = ?4 OR age = ?5",
// 			expectedArgs: []any{1, 2, 3, "email", 100},
// 			query: queryExpression{
// 				query: "SELECT * FROM users WHERE id IN ?1 AND email = ?2 OR age = ?3",
// 				args: []inQueryArg{
// 					{start: 32, stop: 33, val: expr.ArgGroup(1, 2, 3)},
// 					{start: 47, stop: 48, val: expr.Arg("email")},
// 					{start: 59, stop: 60, val: expr.Arg(100)},
// 				},
// 			},
// 			expressions: []expressions{
// 				{
// 					from: 26, to: 33,
// 					expectedSQL:  "id IN (?1, ?2, ?3)",
// 					expectedArgs: []any{1, 2, 3},
// 				},
// 				{
// 					from: 39, to: 48,
// 					expectedSQL:  "email = ?1",
// 					expectedArgs: []any{"email"},
// 				},
// 			},
// 		},
// 	}
//
// 	for name, tc := range cases {
// 		tc.query.dialect = dialect.Dialect
//
// 		t.Run(name, func(t *testing.T) {
// 			t.Run("full", func(t *testing.T) {
// 				buf := &bytes.Buffer{}
// 				args, err := tc.query.WriteQuery(context.Background(), buf, 1)
// 				if err != nil {
// 					t.Fatal(err)
// 				}
//
// 				if diff := cmp.Diff(tc.expectedSQL, buf.String()); diff != "" {
// 					t.Errorf("diff: %s", diff)
// 				}
//
// 				if diff := testutils.ArgsDiff(tc.expectedArgs, args); diff != "" {
// 					t.Errorf("diff: %s", diff)
// 				}
// 			})
//
// 			for i, testExpr := range tc.expressions {
// 				t.Run(strconv.Itoa(i), func(t *testing.T) {
// 					buf := &bytes.Buffer{}
// 					args, err := tc.query.
// 						expr(testExpr.from, testExpr.to).
// 						WriteSQL(context.Background(), buf, dialect.Dialect, 1)
// 					if err != nil {
// 						t.Fatal(err)
// 					}
//
// 					if diff := cmp.Diff(testExpr.expectedSQL, buf.String()); diff != "" {
// 						t.Errorf("diff: %s", diff)
// 					}
//
// 					if diff := testutils.ArgsDiff(testExpr.expectedArgs, args); diff != "" {
// 						t.Errorf("diff: %s", diff)
// 					}
// 				})
// 			}
// 		})
// 	}
// }
