package mm

import (
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/mssql/dialect"
	"github.com/stephenafamo/bob/expr"
)

// Into sets the target table for MERGE INTO
func Into(name any, alias ...string) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.Target = clause.TableRef{Expression: name}
		if len(alias) > 0 {
			m.Target.Alias = alias[0]
		}
	})
}

// Using sets the source table/subquery for MERGE USING
func Using(source bob.Expression, alias string) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.SetUsing(source, alias)
	})
}

// On sets the join condition for MERGE ON
func On(conditions ...bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.AppendOn(conditions...)
	})
}

// OnEQ is a convenience for ON with equality condition
func OnEQ(a, b bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.AppendOn(expr.X[dialect.Expression, dialect.Expression](a).EQ(b))
	})
}

// WhenMatchedChain builds a WHEN MATCHED clause
type WhenMatchedChain struct {
	condition bob.Expression
}

// WhenMatched starts a WHEN MATCHED clause builder
func WhenMatched(condition ...bob.Expression) WhenMatchedChain {
	var cond bob.Expression
	if len(condition) > 0 {
		cond = condition[0]
	}
	return WhenMatchedChain{condition: cond}
}

// ThenUpdate adds UPDATE SET action to WHEN MATCHED
func (w WhenMatchedChain) ThenUpdate(sets ...bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		setAny := make([]any, len(sets))
		for i, s := range sets {
			setAny[i] = s
		}
		m.AppendWhen(dialect.MergeWhen{
			Type:      "MATCHED",
			Condition: w.condition,
			Action: dialect.MergeAction{
				Kind: "UPDATE",
				Set:  setAny,
			},
		})
	})
}

// ThenDelete adds DELETE action to WHEN MATCHED
func (w WhenMatchedChain) ThenDelete() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.AppendWhen(dialect.MergeWhen{
			Type:      "MATCHED",
			Condition: w.condition,
			Action: dialect.MergeAction{
				Kind: "DELETE",
			},
		})
	})
}

// WhenNotMatchedChain builds a WHEN NOT MATCHED clause
type WhenNotMatchedChain struct {
	condition bob.Expression
}

// WhenNotMatched starts a WHEN NOT MATCHED clause with an optional AND condition
func WhenNotMatched(condition ...bob.Expression) WhenNotMatchedChain {
	var cond bob.Expression
	if len(condition) > 0 {
		cond = condition[0]
	}
	return WhenNotMatchedChain{condition: cond}
}

// ThenInsert adds INSERT action to WHEN NOT MATCHED
func (w WhenNotMatchedChain) ThenInsert(cols []string, vals ...bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.AppendWhen(dialect.MergeWhen{
			Type:      "NOT MATCHED",
			Condition: w.condition,
			Action: dialect.MergeAction{
				Kind:       "INSERT",
				InsertCols: cols,
				InsertVals: vals,
			},
		})
	})
}

// WhenNotMatchedBySourceChain builds a WHEN NOT MATCHED BY SOURCE clause
type WhenNotMatchedBySourceChain struct {
	condition bob.Expression
}

// WhenNotMatchedBySource starts a WHEN NOT MATCHED BY SOURCE clause with an optional AND condition
func WhenNotMatchedBySource(condition ...bob.Expression) WhenNotMatchedBySourceChain {
	var cond bob.Expression
	if len(condition) > 0 {
		cond = condition[0]
	}
	return WhenNotMatchedBySourceChain{condition: cond}
}

// ThenDelete adds DELETE action to WHEN NOT MATCHED BY SOURCE
func (w WhenNotMatchedBySourceChain) ThenDelete() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.AppendWhen(dialect.MergeWhen{
			Type:      "NOT MATCHED BY SOURCE",
			Condition: w.condition,
			Action: dialect.MergeAction{
				Kind: "DELETE",
			},
		})
	})
}

// ThenUpdate adds UPDATE SET action to WHEN NOT MATCHED BY SOURCE
func (w WhenNotMatchedBySourceChain) ThenUpdate(sets ...bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		setAny := make([]any, len(sets))
		for i, s := range sets {
			setAny[i] = s
		}
		m.AppendWhen(dialect.MergeWhen{
			Type:      "NOT MATCHED BY SOURCE",
			Condition: w.condition,
			Action: dialect.MergeAction{
				Kind: "UPDATE",
				Set:  setAny,
			},
		})
	})
}

// Output adds OUTPUT clause to MERGE
func Output(clauses ...any) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.AppendOutput(clauses...)
	})
}

// With adds a CTE to the MERGE statement
func With(name string, columns ...string) dialect.CTEChain[*dialect.MergeQuery] {
	return dialect.With[*dialect.MergeQuery](name, columns...)
}
