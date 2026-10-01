package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

// MergeQuery represents T-SQL's MERGE statement
// MERGE INTO target USING source ON condition
// WHEN MATCHED THEN UPDATE SET ...
// WHEN NOT MATCHED THEN INSERT (...) VALUES (...)
// WHEN NOT MATCHED BY SOURCE THEN DELETE
// OUTPUT ...
type MergeQuery struct {
	clause.With
	Target clause.TableRef
	Using  MergeUsing
	On     []bob.Expression
	Whens  []MergeWhen
	Output

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*MergeQuery]
}

// MergeUsing represents the USING clause of a MERGE statement
type MergeUsing struct {
	Source bob.Expression
	Alias  string
}

func (u MergeUsing) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if u.Source == nil {
		return nil, nil
	}

	args, err := bob.Express(ctx, w, d, start, u.Source)
	if err != nil {
		return nil, err
	}

	if u.Alias != "" {
		w.WriteString(" AS ")
		d.WriteQuoted(w, u.Alias)
	}

	return args, nil
}

// MergeWhen represents a single WHEN clause in a MERGE statement
type MergeWhen struct {
	Type      string         // "MATCHED", "NOT MATCHED", "NOT MATCHED BY SOURCE"
	Condition bob.Expression // optional AND condition
	Action    MergeAction
}

// MergeAction represents the action to take in a WHEN clause
type MergeAction struct {
	Kind       string // "UPDATE", "INSERT", "DELETE"
	Set        []any  // for UPDATE: SET assignments
	InsertCols []string
	InsertVals []bob.Expression
}

func (m MergeQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var err error
	var args []any

	if ctx, err = m.RunContextualMods(ctx, &m); err != nil {
		return nil, err
	}

	withArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.With,
		len(m.With.CTEs) > 0, "", "\n")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	w.WriteString("MERGE INTO ")

	tableArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.Target, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.Using,
		m.Using.Source != nil, "\nUSING ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	onArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), m.On,
		"\nON ", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, onArgs...)

	for _, when := range m.Whens {
		whenArgs, err := writeWhen(ctx, w, d, start+len(args), when)
		if err != nil {
			return nil, err
		}
		args = append(args, whenArgs...)
	}

	outputArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.Output,
		m.Output.HasOutput(), "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, outputArgs...)

	w.WriteString(";\n")
	return args, nil
}

func writeWhen(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, when MergeWhen) ([]any, error) {
	var args []any

	w.WriteString("\nWHEN ")
	w.WriteString(when.Type)

	if when.Condition != nil {
		w.WriteString(" AND ")
		condArgs, err := when.Condition.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, condArgs...)
	}

	w.WriteString(" THEN\n")

	switch when.Action.Kind {
	case "UPDATE":
		w.WriteString("UPDATE SET\n")
		setArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), when.Action.Set,
			"", ",\n", "")
		if err != nil {
			return nil, err
		}
		args = append(args, setArgs...)

	case "INSERT":
		w.WriteString("INSERT")
		if len(when.Action.InsertCols) > 0 {
			w.WriteString(" (")
			for i, col := range when.Action.InsertCols {
				if i > 0 {
					w.WriteString(", ")
				}
				d.WriteQuoted(w, col)
			}
			w.WriteString(")")
		}
		w.WriteString(" VALUES (")
		valArgs, err := bob.ExpressSlice(ctx, w, d, start+len(args), when.Action.InsertVals,
			"", ", ", "")
		if err != nil {
			return nil, err
		}
		args = append(args, valArgs...)
		w.WriteString(")")

	case "DELETE":
		w.WriteString("DELETE")
	}

	return args, nil
}

func (m *MergeQuery) SetUsing(source bob.Expression, alias string) {
	m.Using = MergeUsing{Source: source, Alias: alias}
}

func (m *MergeQuery) AppendOn(conditions ...bob.Expression) {
	m.On = append(m.On, conditions...)
}

func (m *MergeQuery) AppendWhen(when MergeWhen) {
	m.Whens = append(m.Whens, when)
}
