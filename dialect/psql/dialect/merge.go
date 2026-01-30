package dialect

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/internal"
)

// MergeWhenType represents the type of WHEN clause in MERGE statement
type MergeWhenType string

// MergeWhenType constants for WHEN clause types
const (
	MergeWhenMatched            MergeWhenType = "MATCHED"
	MergeWhenNotMatched         MergeWhenType = "NOT MATCHED"
	MergeWhenNotMatchedByTarget MergeWhenType = "NOT MATCHED BY TARGET"
	MergeWhenNotMatchedBySource MergeWhenType = "NOT MATCHED BY SOURCE"
)

// MergeActionType represents the type of action in WHEN clause
type MergeActionType string

// MergeActionType constants for action types in WHEN clause
const (
	MergeActionDoNothing MergeActionType = "DO NOTHING"
	MergeActionDelete    MergeActionType = "DELETE"
	MergeActionInsert    MergeActionType = "INSERT"
	MergeActionUpdate    MergeActionType = "UPDATE"
)

// MergeOverridingType represents the OVERRIDING type in INSERT action
type MergeOverridingType string

// MergeOverridingType constants for OVERRIDING clause in INSERT
const (
	MergeOverridingSystem MergeOverridingType = "SYSTEM"
	MergeOverridingUser   MergeOverridingType = "USER"
)

// Trying to represent the merge query structure as documented in
// https://www.postgresql.org/docs/current/sql-merge.html
type MergeQuery struct {
	clause.With
	Only  bool
	Table clause.TableRef
	Using MergeUsing
	When  []MergeWhen
	clause.Returning

	bob.Load
	bob.EmbeddedHook
	bob.ContextualModdable[*MergeQuery]
}

// AppendCTE adds a CTE to the query (implements interface for CTEChain)
func (m *MergeQuery) AppendCTE(cte bob.Expression) {
	m.With.CTEs = append(m.With.CTEs, cte)
}

// SetRecursive sets the recursive flag for CTEs (implements interface for mods.Recursive)
func (m *MergeQuery) SetRecursive(r bool) {
	m.With.Recursive = r
}

// AppendReturning adds expressions to the RETURNING clause (implements interface for mods.Returning)
func (m *MergeQuery) AppendReturning(vals ...any) {
	m.Returning.Expressions = append(m.Returning.Expressions, vals...)
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

	if m.Only {
		w.WriteString("ONLY ")
	}

	tableArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.Table, true, "", "")
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	usingArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.Using,
		m.Using.Source != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, usingArgs...)

	for _, when := range m.When {
		whenArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), when, true, "\n", "")
		if err != nil {
			return nil, err
		}
		args = append(args, whenArgs...)
	}

	retArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), m.Returning,
		len(m.Returning.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, retArgs...)

	return args, nil
}

// MergeUsing represents the USING clause in a MERGE statement
type MergeUsing struct {
	Only      bool
	Source    any // table name or subquery
	Alias     string
	Condition bob.Expression
}

func (u MergeUsing) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("USING ")

	if u.Only {
		w.WriteString("ONLY ")
	}

	// If source is a Query, wrap it in parentheses
	var args []any
	var err error
	if _, isQuery := u.Source.(bob.Query); isQuery {
		w.WriteString("(")
		args, err = bob.Express(ctx, w, d, start, u.Source)
		if err != nil {
			return nil, err
		}
		w.WriteString(")")
	} else {
		args, err = bob.Express(ctx, w, d, start, u.Source)
		if err != nil {
			return nil, err
		}
	}

	if u.Alias != "" {
		w.WriteString(" AS ")
		d.WriteQuoted(w, u.Alias)
	}

	onArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), u.Condition,
		u.Condition != nil, " ON ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, onArgs...)

	return args, nil
}

// MergeWhen represents a WHEN clause in a MERGE statement
type MergeWhen struct {
	Type      MergeWhenType
	Condition bob.Expression
	Action    MergeAction
}

func (w MergeWhen) WriteSQL(ctx context.Context, wr io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	wr.WriteString("WHEN ")
	wr.WriteString(string(w.Type))

	args, err := bob.ExpressIf(ctx, wr, d, start, w.Condition,
		w.Condition != nil, " AND ", "")
	if err != nil {
		return nil, err
	}

	wr.WriteString(" THEN ")

	actionArgs, err := bob.Express(ctx, wr, d, start+len(args), w.Action)
	if err != nil {
		return nil, err
	}
	args = append(args, actionArgs...)

	return args, nil
}

// MergeAction represents the action in a WHEN clause
type MergeAction struct {
	Type       MergeActionType
	Columns    []string
	Overriding MergeOverridingType // MergeOverridingType for INSERT
	Values     []bob.Expression
	Set        []any
}

func (a MergeAction) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	switch a.Type {
	case MergeActionDoNothing:
		w.WriteString("DO NOTHING")
		return nil, nil

	case MergeActionDelete:
		w.WriteString("DELETE")
		return nil, nil

	case MergeActionInsert:
		w.WriteString("INSERT")

		if len(a.Columns) > 0 {
			w.WriteString(" (")
			for i, col := range a.Columns {
				if i > 0 {
					w.WriteString(", ")
				}
				d.WriteQuoted(w, col)
			}
			w.WriteString(")")
		}

		if a.Overriding != "" {
			w.WriteString(" OVERRIDING ")
			w.WriteString(string(a.Overriding))
			w.WriteString(" VALUE")
		}

		if len(a.Values) > 0 {
			w.WriteString(" VALUES (")
			args, err := bob.ExpressSlice(ctx, w, d, start, a.Values, "", ", ", "")
			if err != nil {
				return nil, err
			}
			w.WriteString(")")
			return args, nil
		}

		w.WriteString(" DEFAULT VALUES")
		return nil, nil

	case MergeActionUpdate:
		w.WriteString("UPDATE SET ")
		args, err := bob.ExpressSlice(ctx, w, d, start, internal.ToAnySlice(a.Set), "", ", ", "")
		if err != nil {
			return nil, err
		}
		return args, nil
	}

	return nil, nil
}
