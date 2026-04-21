package dialect

import (
	"context"
	"io"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

func rawExpression(s string) bob.Expression {
	return bob.ExpressionFunc(func(context.Context, io.StringWriter, bob.Dialect, int) ([]any, error) {
		return nil, nil
	})
}

func TestSelectQueryCloneDoesNotShareMutableState(t *testing.T) {
	original := &SelectQuery{
		With: clause.With{
			CTEs: []bob.Expression{rawExpression("cte")},
		},
		SelectList: clause.SelectList{
			Columns:        []any{"id"},
			PreloadColumns: []any{"email"},
		},
		Distinct: Distinct{
			On: []any{"id"},
		},
		TableRef: clause.TableRef{
			Expression: "users",
			Joins: []clause.Join{{
				Type: clause.LeftJoin,
				To: clause.TableRef{
					Expression: "profiles",
					Alias:      "p",
				},
			}},
		},
		Where: clause.Where{
			Conditions: []any{"tenant_id = 1"},
		},
		GroupBy: clause.GroupBy{
			Groups: []any{"id"},
		},
		OrderBy: clause.OrderBy{
			Expressions: []bob.Expression{rawExpression("id DESC")},
		},
		Load: bob.Load{},
	}
	original.AppendLoader(bob.LoaderFunc(func(_ context.Context, _ bob.Executor, _ any) error { return nil }))

	cloned := original.Clone()

	cloned.With.CTEs = append(cloned.With.CTEs, rawExpression("other_cte"))
	cloned.SelectList.Columns = append(cloned.SelectList.Columns, "name")
	cloned.SelectList.PreloadColumns = append(cloned.SelectList.PreloadColumns, "phone")
	cloned.Distinct.On = append(cloned.Distinct.On, "name")
	cloned.TableRef.Joins[0].To.Alias = "profiles_alias"
	cloned.Where.Conditions = append(cloned.Where.Conditions, "active = true")
	cloned.GroupBy.Groups = append(cloned.GroupBy.Groups, "name")
	cloned.OrderBy.Expressions = append(cloned.OrderBy.Expressions, rawExpression("name ASC"))
	cloned.AppendLoader(bob.LoaderFunc(func(_ context.Context, _ bob.Executor, _ any) error { return nil }))

	if len(original.With.CTEs) != 1 {
		t.Fatalf("original with changed unexpectedly: %#v", original.With.CTEs)
	}
	if len(original.SelectList.Columns) != 1 {
		t.Fatalf("original select columns changed unexpectedly: %#v", original.SelectList.Columns)
	}
	if len(original.SelectList.PreloadColumns) != 1 {
		t.Fatalf("original preload columns changed unexpectedly: %#v", original.SelectList.PreloadColumns)
	}
	if len(original.Distinct.On) != 1 {
		t.Fatalf("original distinct changed unexpectedly: %#v", original.Distinct.On)
	}
	if original.TableRef.Joins[0].To.Alias != "p" {
		t.Fatalf("original join alias changed unexpectedly: %#v", original.TableRef.Joins[0].To.Alias)
	}
	if len(original.Where.Conditions) != 1 {
		t.Fatalf("original where changed unexpectedly: %#v", original.Where.Conditions)
	}
	if len(original.GroupBy.Groups) != 1 {
		t.Fatalf("original group by changed unexpectedly: %#v", original.GroupBy.Groups)
	}
	if len(original.OrderBy.Expressions) != 1 {
		t.Fatalf("original order by changed unexpectedly: %#v", original.OrderBy.Expressions)
	}
	if len(original.GetLoaders()) != 1 {
		t.Fatalf("original loaders changed unexpectedly: %d", len(original.GetLoaders()))
	}
}
