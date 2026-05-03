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

// columnsAssignment represents (columns...) = [ROW] (values...) or (columns...) = (subquery)
type columnsAssignment struct {
	cols   []bob.Expression
	values []any // bob.Expression values or a single bob.Query for subquery
	isRow  bool
}

func (a columnsAssignment) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("(")
	colArgs, err := bob.ExpressSlice(ctx, w, d, start, a.cols, "", ", ", "")
	if err != nil {
		return nil, err
	}

	w.WriteString(") = ")

	if a.isRow {
		w.WriteString("ROW ")
	}

	w.WriteString("(")
	valArgs, err := bob.ExpressSlice(ctx, w, d, start+len(colArgs), a.values, "", ", ", "")
	if err != nil {
		return nil, err
	}
	w.WriteString(")")

	return append(colArgs, valArgs...), nil
}

// With adds a WITH clause (CTE) to the MERGE statement.
func With(name string, columns ...string) dialect.CTEChain[*dialect.MergeQuery] {
	return dialect.With[*dialect.MergeQuery](name, columns...)
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

// Using specifies the data source for the MERGE statement.
// Accepts a table name or a bob.Query (subquery).
func Using(source any) UsingChain {
	return UsingChain{source: source}
}

type UsingChain struct {
	source any
	alias  string
	only   bool
}

// As sets an alias for the USING source.
func (u UsingChain) As(alias string) UsingChain {
	u.alias = alias
	return u
}

// Only adds the ONLY modifier to the USING source.
func (u UsingChain) Only() UsingChain {
	u.only = true
	return u
}

// On sets the join condition for the USING clause.
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

// OnEQ is a shorthand for On(left.EQ(right)).
func (u UsingChain) OnEQ(left, right bob.Expression) bob.Mod[*dialect.MergeQuery] {
	return u.On(expr.X[dialect.Expression, dialect.Expression](left).EQ(right))
}

// whenBase holds common fields for all WHEN chain types.
type whenBase struct {
	whenType  dialect.MergeWhenType
	condition bob.Expression
}

func (b whenBase) andCondition(condition bob.Expression) whenBase {
	if b.condition == nil {
		b.condition = condition
	} else {
		b.condition = expr.X[dialect.Expression, dialect.Expression](b.condition).And(condition)
	}
	return b
}

func (b whenBase) thenDoNothing() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      b.whenType,
			Condition: b.condition,
			Action:    dialect.MergeAction{Type: dialect.MergeActionDoNothing},
		})
	})
}

// WhenMatchedChain builds WHEN MATCHED and WHEN NOT MATCHED BY SOURCE clauses.
// Available actions: UPDATE, DELETE, DO NOTHING.
type WhenMatchedChain struct {
	whenBase
}

// And adds a condition to the WHEN clause
func (c WhenMatchedChain) And(condition bob.Expression) WhenMatchedChain {
	c.whenBase = c.andCondition(condition)
	return c
}

// ThenDoNothing sets the action to DO NOTHING
func (c WhenMatchedChain) ThenDoNothing() bob.Mod[*dialect.MergeQuery] {
	return c.thenDoNothing()
}

// ThenDelete sets the action to DELETE
func (c WhenMatchedChain) ThenDelete() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      c.whenType,
			Condition: c.condition,
			Action:    dialect.MergeAction{Type: dialect.MergeActionDelete},
		})
	})
}

// ThenUpdate sets the action to UPDATE with SET clauses
func (c WhenMatchedChain) ThenUpdate(sets ...bob.Mod[*UpdateAction]) bob.Mod[*dialect.MergeQuery] {
	ua := &UpdateAction{}
	for _, s := range sets {
		s.Apply(ua)
	}
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      c.whenType,
			Condition: c.condition,
			Action: dialect.MergeAction{
				Type: dialect.MergeActionUpdate,
				Set:  ua.Set,
			},
		})
	})
}

// WhenNotMatchedChain builds WHEN NOT MATCHED [BY TARGET] clauses.
// Available actions: INSERT, DO NOTHING.
type WhenNotMatchedChain struct {
	whenBase
}

// And adds a condition to the WHEN clause
func (c WhenNotMatchedChain) And(condition bob.Expression) WhenNotMatchedChain {
	c.whenBase = c.andCondition(condition)
	return c
}

// ThenDoNothing sets the action to DO NOTHING
func (c WhenNotMatchedChain) ThenDoNothing() bob.Mod[*dialect.MergeQuery] {
	return c.thenDoNothing()
}

// ThenInsert sets the action to INSERT
func (c WhenNotMatchedChain) ThenInsert(mods ...bob.Mod[*InsertAction]) bob.Mod[*dialect.MergeQuery] {
	ia := &InsertAction{}
	for _, mod := range mods {
		mod.Apply(ia)
	}
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      c.whenType,
			Condition: c.condition,
			Action: dialect.MergeAction{
				Type:       dialect.MergeActionInsert,
				Columns:    ia.Columns,
				Values:     ia.Values,
				Overriding: ia.Overriding,
			},
		})
	})
}

// ThenInsertDefaultValues sets the action to INSERT DEFAULT VALUES
func (c WhenNotMatchedChain) ThenInsertDefaultValues() bob.Mod[*dialect.MergeQuery] {
	return bob.ModFunc[*dialect.MergeQuery](func(m *dialect.MergeQuery) {
		m.When = append(m.When, dialect.MergeWhen{
			Type:      c.whenType,
			Condition: c.condition,
			Action:    dialect.MergeAction{Type: dialect.MergeActionInsert},
		})
	})
}

// WhenMatched creates a WHEN MATCHED clause chain
func WhenMatched() WhenMatchedChain {
	return WhenMatchedChain{whenBase{whenType: dialect.MergeWhenMatched}}
}

// WhenNotMatched creates a WHEN NOT MATCHED (BY TARGET) clause chain
func WhenNotMatched() WhenNotMatchedChain {
	return WhenNotMatchedChain{whenBase{whenType: dialect.MergeWhenNotMatched}}
}

// WhenNotMatchedByTarget is an alias for WhenNotMatched with explicit BY TARGET
func WhenNotMatchedByTarget() WhenNotMatchedChain {
	return WhenNotMatchedChain{whenBase{whenType: dialect.MergeWhenNotMatchedByTarget}}
}

// WhenNotMatchedBySourceChain is an alias for WhenMatchedChain.
// It exists for API clarity so WhenNotMatchedBySource() returns a chain name
// that matches the clause kind, while reusing the same action set.
// Available actions: UPDATE, DELETE, DO NOTHING.
type WhenNotMatchedBySourceChain = WhenMatchedChain

// WhenNotMatchedBySource creates a WHEN NOT MATCHED BY SOURCE clause chain
func WhenNotMatchedBySource() WhenNotMatchedBySourceChain {
	return WhenNotMatchedBySourceChain{whenBase{whenType: dialect.MergeWhenNotMatchedBySource}}
}

// UpdateAction collects SET clauses for WHEN ... THEN UPDATE actions.
type UpdateAction struct {
	Set []any
}

func (u *UpdateAction) AppendSet(clauses ...any) {
	u.Set = append(u.Set, clauses...)
}

// Set adds raw SET expressions to the UPDATE action.
func Set(sets ...bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, internal.ToAnySlice(sets)...)
	})
}

// SetCol creates a single column setter using mods.Set.
// Usage:
//
//	mm.SetCol("name").To(psql.Quote("s", "name"))
//	mm.SetCol("status").ToArg("active")
func SetCol(column string) mods.Set[*UpdateAction] {
	return mods.Set[*UpdateAction]{column}
}

// SetCols creates a multi-column setter: (columns...) = ROW(...) | (values...) | (subquery)
func SetCols(columns ...string) SetColsChain {
	return SetColsChain{columns: columns}
}

type SetColsChain struct {
	columns []string
}

func (s SetColsChain) colExprs() []bob.Expression {
	cols := make([]bob.Expression, len(s.columns))
	for i, c := range s.columns {
		cols[i] = expr.Quote(c)
	}
	return cols
}

// ToRow sets columns to ROW of expressions: (columns...) = ROW (expressions...)
func (s SetColsChain) ToRow(values ...bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, columnsAssignment{cols: s.colExprs(), values: internal.ToAnySlice(values), isRow: true})
	})
}

// ToExprs sets columns to expressions: (columns...) = (expressions...)
func (s SetColsChain) ToExprs(values ...bob.Expression) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, columnsAssignment{cols: s.colExprs(), values: internal.ToAnySlice(values)})
	})
}

// ToQuery sets columns from a subquery: (columns...) = (subquery)
func (s SetColsChain) ToQuery(q bob.Query) bob.Mod[*UpdateAction] {
	return bob.ModFunc[*UpdateAction](func(u *UpdateAction) {
		u.Set = append(u.Set, columnsAssignment{cols: s.colExprs(), values: []any{q}})
	})
}

// InsertAction collects options for WHEN ... THEN INSERT actions.
type InsertAction struct {
	Columns    []string
	Values     []bob.Expression
	Overriding dialect.OverridingType
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
// Use psql.Raw("DEFAULT") for DEFAULT keyword
func Values(values ...bob.Expression) bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Values = append(i.Values, values...)
	})
}

// OverridingSystem adds OVERRIDING SYSTEM VALUE for INSERT action
// Use when inserting into identity columns defined as GENERATED ALWAYS
func OverridingSystem() bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Overriding = dialect.OverridingSystem
	})
}

// OverridingUser adds OVERRIDING USER VALUE for INSERT action
// Use when identity columns defined as GENERATED BY DEFAULT should use sequence values
func OverridingUser() bob.Mod[*InsertAction] {
	return bob.ModFunc[*InsertAction](func(i *InsertAction) {
		i.Overriding = dialect.OverridingUser
	})
}

// Returning adds a RETURNING clause
func Returning(clauses ...any) bob.Mod[*dialect.MergeQuery] {
	return mods.Returning[*dialect.MergeQuery](clauses)
}
