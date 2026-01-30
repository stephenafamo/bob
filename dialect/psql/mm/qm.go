package mm

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/mods"
)

// rowAssignment represents (columns...) = [ROW] (values...)
type rowAssignment struct {
	cols   []bob.Expression
	values []bob.Expression
	isRow  bool
}

func (r rowAssignment) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	// Write (col1, col2, ...)
	w.WriteString("(")
	colArgs, err := bob.ExpressSlice(ctx, w, d, start, r.cols, "", ", ", "")
	if err != nil {
		return nil, err
	}

	w.WriteString(") = ")

	if r.isRow {
		w.WriteString("ROW ")
	}

	// Write (val1, val2, ...)
	w.WriteString("(")
	valArgs, err := bob.ExpressSlice(ctx, w, d, start+len(colArgs), r.values, "", ", ", "")
	if err != nil {
		return nil, err
	}
	w.WriteString(")")

	return append(colArgs, valArgs...), nil
}

// queryAssignment represents (columns...) = (subquery)
type queryAssignment struct {
	cols  []bob.Expression
	query bob.Query
}

func (q queryAssignment) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	// Write (col1, col2, ...)
	w.WriteString("(")
	colArgs, err := bob.ExpressSlice(ctx, w, d, start, q.cols, "", ", ", "")
	if err != nil {
		return nil, err
	}

	w.WriteString(") = (")

	// Write subquery
	queryArgs, err := bob.Express(ctx, w, d, start+len(colArgs), q.query)
	if err != nil {
		return nil, err
	}
	w.WriteString(")")

	return append(colArgs, queryArgs...), nil
}

func With(name string, columns ...string) dialect.CTEChain[*dialect.MergeQuery] {
	return dialect.With[*dialect.MergeQuery](name, columns...)
}

func Recursive(r bool) bob.Mod[*dialect.MergeQuery] {
	return mods.Recursive[*dialect.MergeQuery](r)
}

// Into specifies the target table for the MERGE statement
func Into(name any) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.Table = clause.TableRef{
			Expression: name,
		}
	})
}

// IntoAs specifies the target table with an alias for the MERGE statement
func IntoAs(name any, alias string) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.Table = clause.TableRef{
			Expression: name,
			Alias:      alias,
		}
	})
}

// Only specifies ONLY modifier for the target table
func Only() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.Only = true
	})
}

// Using specifies the data source for the MERGE statement
func Using(source any) UsingChain {
	return UsingChain{source: source}
}

// UsingQuery specifies a subquery as the data source for the MERGE statement
func UsingQuery(q bob.Query) UsingChain {
	return UsingChain{source: q}
}

// UsingChain is a chain for building the USING clause
type UsingChain struct {
	source any
	alias  string
	only   bool
}

func (u UsingChain) As(alias string) UsingChain {
	u.alias = alias
	return u
}

func (u UsingChain) Only() UsingChain {
	u.only = true
	return u
}

func (u UsingChain) On(condition bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.Using = dialect.MergeUsing{
			Only:      u.only,
			Source:    u.source,
			Alias:     u.alias,
			Condition: condition,
		}
	})
}

func (u UsingChain) OnEQ(left, right bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return u.On(expr.X[dialect.Expression, dialect.Expression](left).EQ(right))
}

// WhenMatched creates a WHEN MATCHED clause
func WhenMatched(mods ...bob.Mod[*WhenClause]) bob.Mod[*dialect.MergeQuery] {
	wc := &WhenClause{Type: dialect.MergeWhenMatched}
	for _, mod := range mods {
		mod.Apply(wc)
	}
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      wc.Type,
			Condition: wc.Condition,
			Action:    wc.Action,
		})
	})
}

// WhenNotMatched creates a WHEN NOT MATCHED (BY TARGET) clause
func WhenNotMatched(mods ...bob.Mod[*WhenClause]) bob.Mod[*dialect.MergeQuery] {
	wc := &WhenClause{Type: dialect.MergeWhenNotMatched}
	for _, mod := range mods {
		mod.Apply(wc)
	}
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      wc.Type,
			Condition: wc.Condition,
			Action:    wc.Action,
		})
	})
}

// WhenNotMatchedByTarget is an alias for WhenNotMatched with explicit BY TARGET
func WhenNotMatchedByTarget(mods ...bob.Mod[*WhenClause]) bob.Mod[*dialect.MergeQuery] {
	wc := &WhenClause{Type: dialect.MergeWhenNotMatchedByTarget}
	for _, mod := range mods {
		mod.Apply(wc)
	}
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      wc.Type,
			Condition: wc.Condition,
			Action:    wc.Action,
		})
	})
}

// WhenNotMatchedBySource creates a WHEN NOT MATCHED BY SOURCE clause
func WhenNotMatchedBySource(mods ...bob.Mod[*WhenClause]) bob.Mod[*dialect.MergeQuery] {
	wc := &WhenClause{Type: dialect.MergeWhenNotMatchedBySource}
	for _, mod := range mods {
		mod.Apply(wc)
	}
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      wc.Type,
			Condition: wc.Condition,
			Action:    wc.Action,
		})
	})
}

// WhenClause is a builder for WHEN clauses
type WhenClause struct {
	Type      dialect.MergeWhenType
	Condition bob.Expression
	Action    dialect.MergeAction
}

// And adds a condition to the WHEN clause
func And(condition bob.Expression) bob.Mod[*WhenClause] {
	return bob.ModFunc[*WhenClause](func(w *WhenClause) {
		if w.Condition == nil {
			w.Condition = condition
		} else {
			w.Condition = expr.X[dialect.Expression, dialect.Expression](w.Condition).And(condition)
		}
	})
}

// ThenDoNothing sets the action to DO NOTHING
func ThenDoNothing() bob.Mod[*WhenClause] {
	return bob.ModFunc[*WhenClause](func(w *WhenClause) {
		w.Action = dialect.MergeAction{Type: dialect.MergeActionDoNothing}
	})
}

// ThenDelete sets the action to DELETE
func ThenDelete() bob.Mod[*WhenClause] {
	return bob.ModFunc[*WhenClause](func(w *WhenClause) {
		w.Action = dialect.MergeAction{Type: dialect.MergeActionDelete}
	})
}

// ThenUpdate sets the action to UPDATE with SET clauses
// Supports MERGE UPDATE syntax:
//   - column = expression
//   - column = DEFAULT
//   - (columns...) = ROW (expressions...)
//   - (columns...) = (subquery)
func ThenUpdate(sets ...bob.Mod[*UpdateAction]) bob.Mod[*WhenClause] {
	ua := &UpdateAction{}
	for _, s := range sets {
		s.Apply(ua)
	}
	return bob.ModFunc[*WhenClause](func(w *WhenClause) {
		w.Action = dialect.MergeAction{
			Type: dialect.MergeActionUpdate,
			Set:  ua.Set,
		}
	})
}

// UpdateAction is a builder for UPDATE action in MERGE
type UpdateAction struct {
	Set []any
}

// Set adds raw SET expressions to the UPDATE action
func Set(sets ...bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, internal.ToAnySlice(sets)...)
	})
}

// SetCol creates a single column setter: column = expression | DEFAULT
func SetCol(column string) SetChain {
	return SetChain{column: column}
}

// SetChain is a chain for building SET column = value
type SetChain struct {
	column string
}

// To sets column to a raw value: column = value
func (s SetChain) To(value any) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, expr.OP("=", expr.Quote(s.column), value))
	})
}

// ToArg sets column to a parameterized value: column = $N
func (s SetChain) ToArg(value any) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, expr.OP("=", expr.Quote(s.column), expr.Arg(value)))
	})
}

// ToExpr sets column to an expression: column = expression
// Use psql.Quote("source_alias", "column") to reference source columns
// Use psql.Quote("target_alias", "column") to reference target columns
func (s SetChain) ToExpr(e bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, expr.OP("=", expr.Quote(s.column), e))
	})
}

// ToDefault sets column to DEFAULT: column = DEFAULT
func (s SetChain) ToDefault() bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, expr.OP("=", expr.Quote(s.column), expr.Raw("DEFAULT")))
	})
}

// SetCols creates a multi-column setter: (columns...) = ROW(...) | (subquery)
func SetCols(columns ...string) SetColsChain {
	return SetColsChain{columns: columns}
}

// SetColsChain is a chain for building SET (columns...) = ROW(...) | (subquery)
type SetColsChain struct {
	columns []string
}

// ToRow sets columns to ROW of expressions: (columns...) = ROW (expressions...)
func (s SetColsChain) ToRow(values ...bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		// Build (col1, col2, ...) = ROW (val1, val2, ...)
		cols := make([]bob.Expression, len(s.columns))
		for i, c := range s.columns {
			cols[i] = expr.Quote(c)
		}
		u.Set = append(u.Set, rowAssignment{cols: cols, values: values, isRow: true})
	})
}

// ToExprs sets columns to expressions without ROW: (columns...) = (expressions...)
func (s SetColsChain) ToExprs(values ...bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		cols := make([]bob.Expression, len(s.columns))
		for i, c := range s.columns {
			cols[i] = expr.Quote(c)
		}
		u.Set = append(u.Set, rowAssignment{cols: cols, values: values, isRow: false})
	})
}

// ToQuery sets columns from a subquery: (columns...) = (subquery)
func (s SetColsChain) ToQuery(q bob.Query) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		cols := make([]bob.Expression, len(s.columns))
		for i, c := range s.columns {
			cols[i] = expr.Quote(c)
		}
		u.Set = append(u.Set, queryAssignment{cols: cols, query: q})
	})
}

// ThenInsert sets the action to INSERT
// Use with Columns(), Values(), OverridingSystem(), OverridingUser() modifiers
// If no Values() is specified, DEFAULT VALUES will be used
func ThenInsert(mods ...bob.Mod[*InsertAction]) bob.Mod[*WhenClause] {
	ia := &InsertAction{}
	for _, mod := range mods {
		mod.Apply(ia)
	}
	return bob.ModFunc[*WhenClause](func(w *WhenClause) {
		w.Action = dialect.MergeAction{
			Type:       dialect.MergeActionInsert,
			Columns:    ia.Columns,
			Values:     ia.Values,
			Overriding: ia.Overriding,
		}
	})
}

// ThenInsertDefaultValues sets the action to INSERT DEFAULT VALUES (shortcut)
func ThenInsertDefaultValues() bob.Mod[*WhenClause] {
	return bob.ModFunc[*WhenClause](func(w *WhenClause) {
		w.Action = dialect.MergeAction{
			Type: dialect.MergeActionInsert,
			// Empty Values signals DEFAULT VALUES
		}
	})
}

// InsertAction is a builder for INSERT action in MERGE
// Supports: INSERT [(columns...)] [OVERRIDING {SYSTEM|USER} VALUE] {VALUES (...) | DEFAULT VALUES}
type InsertAction struct {
	Columns    []string
	Values     []bob.Expression
	Overriding dialect.MergeOverridingType
}

// Columns specifies the target columns for INSERT action
// Column names can include subfield names or array subscripts if needed
func Columns(columns ...string) bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Columns = append(i.Columns, columns...)
	})
}

// Values specifies the values for INSERT action
// Expressions can reference source data columns (for WHEN NOT MATCHED BY TARGET)
// Use psql.Quote("source_alias", "column") to reference source columns
// Use psql.Arg(value) for literal values
// Use expr.Raw("DEFAULT") for DEFAULT keyword
func Values(values ...bob.Expression) bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Values = append(i.Values, values...)
	})
}

// OverridingSystem adds OVERRIDING SYSTEM VALUE for INSERT action
// Use when inserting into identity columns defined as GENERATED ALWAYS
func OverridingSystem() bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Overriding = dialect.MergeOverridingSystem
	})
}

// OverridingUser adds OVERRIDING USER VALUE for INSERT action
// Use when identity columns defined as GENERATED BY DEFAULT should use sequence values
func OverridingUser() bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Overriding = dialect.MergeOverridingUser
	})
}

// Returning adds a RETURNING clause
func Returning(clauses ...any) bob.Mod[*dialect.MergeQuery] {
	return mods.Returning[*dialect.MergeQuery](clauses)
}
