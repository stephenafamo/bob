package psql

import (
	"context"
	"database/sql"
	"io"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	psqldialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/scan"
)

type derivedUpdateQuery struct {
	state immutableUpdateState
	load  bob.Load
	hooks bob.EmbeddedHook
}

type immutableUpdateState struct {
	With      clause.With
	Only      bool
	Table     clause.TableRef
	From      clause.TableRef
	Set       clause.Set
	Where     clause.Where
	Returning clause.Returning
}

func asImmutableUpdate(q bob.BaseQuery[*psqldialect.UpdateQuery]) derivedUpdateQuery {
	return derivedUpdateQuery{
		state: immutableUpdateState{
			With: clause.With{
				Recursive: q.Expression.With.Recursive,
				CTEs:      append([]bob.Expression(nil), q.Expression.With.CTEs...),
			},
			Only:  q.Expression.Only,
			Table: cloneTableRef(q.Expression.Table),
			From:  cloneTableRef(q.Expression.TableRef),
			Set: clause.Set{
				Set: append([]any(nil), q.Expression.Set.Set...),
			},
			Where: clause.Where{
				Conditions: append([]any(nil), q.Expression.Where.Conditions...),
			},
			Returning: clause.Returning{
				Expressions: append([]any(nil), q.Expression.Returning.Expressions...),
			},
		},
		load:  q.Expression.Load,
		hooks: q.Expression.EmbeddedHook,
	}
}

func (q derivedUpdateQuery) Type() bob.QueryType { return bob.QueryTypeUpdate }

func (q derivedUpdateQuery) With(queryMods ...bob.Mod[*psqldialect.UpdateQuery]) derivedUpdateQuery {
	next, ok := q.state.withMods(queryMods...)
	if ok {
		q.state = next
		return q
	}

	base := q.mutableBase()
	mutable := base.Expression
	for _, mod := range queryMods {
		mod.Apply(mutable)
	}

	return asImmutableUpdate(base)
}

func (q derivedUpdateQuery) Exec(ctx context.Context, exec bob.Executor) (sql.Result, error) {
	return bob.Exec(ctx, exec, q)
}

func (q derivedUpdateQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	return q.hooks.RunHooks(ctx, exec)
}

func (q derivedUpdateQuery) GetLoaders() []bob.Loader {
	return q.load.GetLoaders()
}

func (q derivedUpdateQuery) GetMapperMods() []scan.MapperMod {
	return q.load.GetMapperMods()
}

func (q derivedUpdateQuery) Build(ctx context.Context) (string, []any, error) {
	return q.BuildN(ctx, 1)
}

func (q derivedUpdateQuery) BuildN(ctx context.Context, start int) (string, []any, error) {
	var sb strings.Builder
	args, err := q.WriteQuery(ctx, &sb, start)
	if err != nil {
		return "", nil, err
	}
	return sb.String(), args, nil
}

func (q derivedUpdateQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	var args []any

	if len(q.state.With.CTEs) > 0 {
		withArgs, err := q.state.With.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, withArgs...)
		w.WriteString("\n")
	}

	w.WriteString("UPDATE ")
	if q.state.Only {
		w.WriteString("ONLY ")
	}

	tableArgs, err := q.state.Table.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	w.WriteString(" SET\n")
	setArgs, err := q.state.Set.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
	if err != nil {
		return nil, err
	}
	args = append(args, setArgs...)

	if q.state.From.Expression != nil {
		w.WriteString("\nFROM ")
		fromArgs, err := q.state.From.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, fromArgs...)
	}

	if len(q.state.Where.Conditions) > 0 {
		w.WriteString("\n")
		whereArgs, err := q.state.Where.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, whereArgs...)
	}

	if len(q.state.Returning.Expressions) > 0 {
		w.WriteString("\n")
		retArgs, err := q.state.Returning.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, retArgs...)
	}

	return args, nil
}

func (q derivedUpdateQuery) WriteSQL(ctx context.Context, w io.StringWriter, _ bob.Dialect, start int) ([]any, error) {
	return q.WriteQuery(ctx, w, start)
}

func (q derivedUpdateQuery) mutableBase() bob.BaseQuery[*psqldialect.UpdateQuery] {
	mutable := &psqldialect.UpdateQuery{
		With:         q.state.With,
		Only:         q.state.Only,
		Table:        q.state.Table,
		Set:          q.state.Set,
		TableRef:     q.state.From,
		Where:        q.state.Where,
		Returning:    q.state.Returning,
		Load:         q.load,
		EmbeddedHook: q.hooks,
	}

	return bob.BaseQuery[*psqldialect.UpdateQuery]{
		Expression: mutable,
		Dialect:    psqldialect.Dialect,
		QueryType:  bob.QueryTypeUpdate,
	}
}

func (s immutableUpdateState) withMods(queryMods ...bob.Mod[*psqldialect.UpdateQuery]) (immutableUpdateState, bool) {
	next := s
	var cloneWhere, cloneReturning bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Where[*psqldialect.UpdateQuery]:
			if !cloneWhere {
				next.Where.Conditions = append([]any(nil), s.Where.Conditions...)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.Returning[*psqldialect.UpdateQuery]:
			if !cloneReturning {
				next.Returning.Expressions = append([]any(nil), s.Returning.Expressions...)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case psqldialect.FromChain[*psqldialect.UpdateQuery]:
			next.From = cloneTableRef(m())
		default:
			return next, false
		}
	}

	return next, true
}

type derivedDeleteQuery struct {
	state immutableDeleteState
	load  bob.Load
	hooks bob.EmbeddedHook
}

type immutableDeleteState struct {
	With      clause.With
	Only      bool
	Table     clause.TableRef
	Using     clause.TableRef
	Where     clause.Where
	Returning clause.Returning
}

func asImmutableDelete(q bob.BaseQuery[*psqldialect.DeleteQuery]) derivedDeleteQuery {
	return derivedDeleteQuery{
		state: immutableDeleteState{
			With: clause.With{
				Recursive: q.Expression.With.Recursive,
				CTEs:      append([]bob.Expression(nil), q.Expression.With.CTEs...),
			},
			Only:  q.Expression.Only,
			Table: cloneTableRef(q.Expression.Table),
			Using: cloneTableRef(q.Expression.TableRef),
			Where: clause.Where{Conditions: append([]any(nil), q.Expression.Where.Conditions...)},
			Returning: clause.Returning{
				Expressions: append([]any(nil), q.Expression.Returning.Expressions...),
			},
		},
		load:  q.Expression.Load,
		hooks: q.Expression.EmbeddedHook,
	}
}

func (q derivedDeleteQuery) Type() bob.QueryType { return bob.QueryTypeDelete }

func (q derivedDeleteQuery) With(queryMods ...bob.Mod[*psqldialect.DeleteQuery]) derivedDeleteQuery {
	next, ok := q.state.withMods(queryMods...)
	if ok {
		q.state = next
		return q
	}

	base := q.mutableBase()
	mutable := base.Expression
	for _, mod := range queryMods {
		mod.Apply(mutable)
	}

	return asImmutableDelete(base)
}

func (q derivedDeleteQuery) Exec(ctx context.Context, exec bob.Executor) (sql.Result, error) {
	return bob.Exec(ctx, exec, q)
}

func (q derivedDeleteQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	return q.hooks.RunHooks(ctx, exec)
}

func (q derivedDeleteQuery) GetLoaders() []bob.Loader { return q.load.GetLoaders() }

func (q derivedDeleteQuery) GetMapperMods() []scan.MapperMod { return q.load.GetMapperMods() }

func (q derivedDeleteQuery) Build(ctx context.Context) (string, []any, error) {
	return q.BuildN(ctx, 1)
}

func (q derivedDeleteQuery) BuildN(ctx context.Context, start int) (string, []any, error) {
	var sb strings.Builder
	args, err := q.WriteQuery(ctx, &sb, start)
	if err != nil {
		return "", nil, err
	}
	return sb.String(), args, nil
}

func (q derivedDeleteQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	var args []any
	if len(q.state.With.CTEs) > 0 {
		withArgs, err := q.state.With.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, withArgs...)
		w.WriteString("\n")
	}

	w.WriteString("DELETE FROM ")
	if q.state.Only {
		w.WriteString("ONLY ")
	}

	tableArgs, err := q.state.Table.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	if q.state.Using.Expression != nil {
		w.WriteString("\nUSING ")
		usingArgs, err := q.state.Using.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, usingArgs...)
	}

	if len(q.state.Where.Conditions) > 0 {
		w.WriteString("\n")
		whereArgs, err := q.state.Where.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, whereArgs...)
	}

	if len(q.state.Returning.Expressions) > 0 {
		w.WriteString("\n")
		retArgs, err := q.state.Returning.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, retArgs...)
	}

	return args, nil
}

func (q derivedDeleteQuery) WriteSQL(ctx context.Context, w io.StringWriter, _ bob.Dialect, start int) ([]any, error) {
	return q.WriteQuery(ctx, w, start)
}

func (q derivedDeleteQuery) mutableBase() bob.BaseQuery[*psqldialect.DeleteQuery] {
	mutable := &psqldialect.DeleteQuery{
		With:         q.state.With,
		Only:         q.state.Only,
		Table:        q.state.Table,
		TableRef:     q.state.Using,
		Where:        q.state.Where,
		Returning:    q.state.Returning,
		Load:         q.load,
		EmbeddedHook: q.hooks,
	}

	return bob.BaseQuery[*psqldialect.DeleteQuery]{
		Expression: mutable,
		Dialect:    psqldialect.Dialect,
		QueryType:  bob.QueryTypeDelete,
	}
}

func (s immutableDeleteState) withMods(queryMods ...bob.Mod[*psqldialect.DeleteQuery]) (immutableDeleteState, bool) {
	next := s
	var cloneWhere, cloneReturning bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Where[*psqldialect.DeleteQuery]:
			if !cloneWhere {
				next.Where.Conditions = append([]any(nil), s.Where.Conditions...)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.Returning[*psqldialect.DeleteQuery]:
			if !cloneReturning {
				next.Returning.Expressions = append([]any(nil), s.Returning.Expressions...)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case psqldialect.FromChain[*psqldialect.DeleteQuery]:
			next.Using = cloneTableRef(m())
		default:
			return next, false
		}
	}

	return next, true
}

type derivedInsertQuery struct {
	state immutableInsertState
	load  bob.Load
	hooks bob.EmbeddedHook
}

type immutableInsertState struct {
	With       clause.With
	Overriding string
	Table      clause.TableRef
	Values     clause.Values
	Conflict   clause.Conflict
	Returning  clause.Returning
}

func asImmutableInsert(q bob.BaseQuery[*psqldialect.InsertQuery]) derivedInsertQuery {
	return derivedInsertQuery{
		state: immutableInsertState{
			With: clause.With{
				Recursive: q.Expression.With.Recursive,
				CTEs:      append([]bob.Expression(nil), q.Expression.With.CTEs...),
			},
			Overriding: q.Expression.Overriding,
			Table:      cloneTableRef(q.Expression.TableRef),
			Values: clause.Values{
				Query: q.Expression.Values.Query,
				Vals:  append([]clause.Value(nil), q.Expression.Values.Vals...),
			},
			Conflict: clause.Conflict{Expression: q.Expression.Conflict.Expression},
			Returning: clause.Returning{
				Expressions: append([]any(nil), q.Expression.Returning.Expressions...),
			},
		},
		load:  q.Expression.Load,
		hooks: q.Expression.EmbeddedHook,
	}
}

func (q derivedInsertQuery) Type() bob.QueryType { return bob.QueryTypeInsert }

func (q derivedInsertQuery) With(queryMods ...bob.Mod[*psqldialect.InsertQuery]) derivedInsertQuery {
	next, ok := q.state.withMods(queryMods...)
	if ok {
		q.state = next
		return q
	}

	base := q.mutableBase()
	mutable := base.Expression
	for _, mod := range queryMods {
		mod.Apply(mutable)
	}

	return asImmutableInsert(base)
}

func (q derivedInsertQuery) Exec(ctx context.Context, exec bob.Executor) (sql.Result, error) {
	return bob.Exec(ctx, exec, q)
}

func (q derivedInsertQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	return q.hooks.RunHooks(ctx, exec)
}

func (q derivedInsertQuery) GetLoaders() []bob.Loader { return q.load.GetLoaders() }

func (q derivedInsertQuery) GetMapperMods() []scan.MapperMod { return q.load.GetMapperMods() }

func (q derivedInsertQuery) Build(ctx context.Context) (string, []any, error) {
	return q.BuildN(ctx, 1)
}

func (q derivedInsertQuery) BuildN(ctx context.Context, start int) (string, []any, error) {
	var sb strings.Builder
	args, err := q.WriteQuery(ctx, &sb, start)
	if err != nil {
		return "", nil, err
	}
	return sb.String(), args, nil
}

func (q derivedInsertQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	var args []any
	if len(q.state.With.CTEs) > 0 {
		withArgs, err := q.state.With.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, withArgs...)
		w.WriteString("\n")
	}

	w.WriteString("INSERT INTO ")
	tableArgs, err := q.state.Table.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
	if err != nil {
		return nil, err
	}
	args = append(args, tableArgs...)

	if q.state.Overriding != "" {
		w.WriteString("\nOVERRIDING ")
		w.WriteString(q.state.Overriding)
		w.WriteString(" VALUE")
	}

	w.WriteString("\n")
	valArgs, err := q.state.Values.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
	if err != nil {
		return nil, err
	}
	args = append(args, valArgs...)

	if q.state.Conflict.Expression != nil {
		w.WriteString("\n")
		conflictArgs, err := q.state.Conflict.Expression.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, conflictArgs...)
	}

	if len(q.state.Returning.Expressions) > 0 {
		w.WriteString("\n")
		retArgs, err := q.state.Returning.WriteSQL(ctx, w, psqldialect.Dialect, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, retArgs...)
	}

	w.WriteString("\n")
	return args, nil
}

func (q derivedInsertQuery) WriteSQL(ctx context.Context, w io.StringWriter, _ bob.Dialect, start int) ([]any, error) {
	return q.WriteQuery(ctx, w, start)
}

func (q derivedInsertQuery) mutableBase() bob.BaseQuery[*psqldialect.InsertQuery] {
	values := clause.Values{
		Query: q.state.Values.Query,
		Vals:  append([]clause.Value(nil), q.state.Values.Vals...),
	}

	mutable := &psqldialect.InsertQuery{
		With:         q.state.With,
		Overriding:   q.state.Overriding,
		TableRef:     q.state.Table,
		Values:       values,
		Conflict:     q.state.Conflict,
		Returning:    q.state.Returning,
		Load:         q.load,
		EmbeddedHook: q.hooks,
	}

	return bob.BaseQuery[*psqldialect.InsertQuery]{
		Expression: mutable,
		Dialect:    psqldialect.Dialect,
		QueryType:  bob.QueryTypeInsert,
	}
}

func (s immutableInsertState) withMods(queryMods ...bob.Mod[*psqldialect.InsertQuery]) (immutableInsertState, bool) {
	next := s
	var cloneReturning, cloneVals bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Returning[*psqldialect.InsertQuery]:
			if !cloneReturning {
				next.Returning.Expressions = append([]any(nil), s.Returning.Expressions...)
				cloneReturning = true
			}
			next.Returning.Expressions = append(next.Returning.Expressions, []any(m)...)
		case mods.Values[*psqldialect.InsertQuery]:
			if !cloneVals {
				next.Values.Vals = append([]clause.Value(nil), s.Values.Vals...)
				cloneVals = true
			}
			next.Values.Vals = append(next.Values.Vals, clause.Value(m))
		case mods.Rows[*psqldialect.InsertQuery]:
			if !cloneVals {
				next.Values.Vals = append([]clause.Value(nil), s.Values.Vals...)
				cloneVals = true
			}
			for _, row := range m {
				next.Values.Vals = append(next.Values.Vals, clause.Value(row))
			}
		default:
			return next, false
		}
	}

	return next, true
}
