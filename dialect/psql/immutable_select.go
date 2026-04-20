package psql

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	psqldialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/scan"
)

type ImmutableSelectQuery struct {
	state immutableSelectState
}

type immutableSelectState struct {
	With           clause.With
	SelectColumns  []any
	PreloadColumns []any
	Distinct       psqldialect.Distinct
	TableRef       clause.TableRef
	Where          clause.Where
	GroupBy        clause.GroupBy
	Having         clause.Having
	Windows        clause.Windows
	Combines       clause.Combines
	OrderBy        clause.OrderBy
	Limit          clause.Limit
	Offset         clause.Offset
	Fetch          clause.Fetch
	Locks          clause.Locks

	CombinedOrder  clause.OrderBy
	CombinedLimit  clause.Limit
	CombinedFetch  clause.Fetch
	CombinedOffset clause.Offset
}

func asImmutable(q bob.BaseQuery[*psqldialect.SelectQuery]) ImmutableSelectQuery {
	return ImmutableSelectQuery{state: immutableStateFromMutable(q.Expression)}
}

func (q ImmutableSelectQuery) Type() bob.QueryType {
	return bob.QueryTypeSelect
}

func (q ImmutableSelectQuery) With(queryMods ...bob.Mod[*psqldialect.SelectQuery]) ImmutableSelectQuery {
	next, ok := q.state.withMods(queryMods...)
	if ok {
		return ImmutableSelectQuery{state: next}
	}

	mutable := q.state.toMutable()
	for _, mod := range queryMods {
		mod.Apply(&mutable)
	}

	return ImmutableSelectQuery{state: immutableStateFromMutable(&mutable)}
}

func (q ImmutableSelectQuery) AsCount() ImmutableSelectQuery {
	next := q.state
	next.SelectColumns = []any{"count(1)"}
	next.PreloadColumns = nil
	next.OrderBy.Expressions = nil
	next.GroupBy.Groups = nil
	next.GroupBy.With = ""
	next.GroupBy.Distinct = false
	next.Offset.Count = nil
	next.Limit.Count = 1

	return ImmutableSelectQuery{state: next}
}

func (q ImmutableSelectQuery) Build(ctx context.Context) (string, []any, error) {
	return q.BuildN(ctx, 1)
}

func (q ImmutableSelectQuery) BuildN(ctx context.Context, start int) (string, []any, error) {
	var sb strings.Builder
	args, err := q.WriteQuery(ctx, &sb, start)
	if err != nil {
		return "", nil, err
	}

	return sb.String(), args, nil
}

func (q ImmutableSelectQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	writer := immutableSelectWriter{
		ctx:   ctx,
		w:     w,
		start: start,
	}

	if err := writer.writeQuery(q.state); err != nil {
		return nil, err
	}

	return writer.args, nil
}

func (q ImmutableSelectQuery) WriteSQL(ctx context.Context, w io.StringWriter, _ bob.Dialect, start int) ([]any, error) {
	w.WriteString("(")
	args, err := q.WriteQuery(ctx, w, start)
	if err != nil {
		return nil, err
	}
	w.WriteString(")")
	return args, nil
}

type ImmutableViewQuery[T any, Ts ~[]T] struct {
	Query   ImmutableSelectQuery
	Scanner scan.Mapper[T]
	Hooks   *bob.Hooks[*psqldialect.SelectQuery, bob.SkipQueryHooksKey]
}

func (q *ViewQuery[T, Ts]) With(queryMods ...bob.Mod[*psqldialect.SelectQuery]) ImmutableViewQuery[T, Ts] {
	state := immutableStateFromMutable(q.BaseQuery.Expression)
	if len(state.SelectColumns) == 0 && q.defaultSelect != nil {
		state.SelectColumns = append(state.SelectColumns, q.defaultSelect)
	}

	return ImmutableViewQuery[T, Ts]{
		Query:   ImmutableSelectQuery{state: state}.With(queryMods...),
		Scanner: q.Scanner,
		Hooks:   q.Hooks,
	}
}

func (q ImmutableViewQuery[T, Ts]) With(queryMods ...bob.Mod[*psqldialect.SelectQuery]) ImmutableViewQuery[T, Ts] {
	q.Query = q.Query.With(queryMods...)
	return q
}

func (q ImmutableViewQuery[T, Ts]) One(ctx context.Context, exec bob.Executor) (T, error) {
	return q.mutable().One(ctx, exec)
}

func (q ImmutableViewQuery[T, Ts]) All(ctx context.Context, exec bob.Executor) (Ts, error) {
	return q.mutable().All(ctx, exec)
}

func (q ImmutableViewQuery[T, Ts]) Cursor(ctx context.Context, exec bob.Executor) (scan.ICursor[T], error) {
	return q.mutable().Cursor(ctx, exec)
}

func (q ImmutableViewQuery[T, Ts]) Each(ctx context.Context, exec bob.Executor) (func(func(T, error) bool), error) {
	return q.mutable().Each(ctx, exec)
}

func (q ImmutableViewQuery[T, Ts]) Count(ctx context.Context, exec bob.Executor) (int64, error) {
	mq := q.mutable()
	ctx, err := mq.RunHooks(ctx, exec)
	if err != nil {
		return 0, err
	}
	return bob.One(ctx, exec, asCountQuery(mq.BaseQuery), scan.SingleColumnMapper[int64])
}

func (q ImmutableViewQuery[T, Ts]) Exists(ctx context.Context, exec bob.Executor) (bool, error) {
	count, err := q.Count(ctx, exec)
	return count > 0, err
}

func (q ImmutableViewQuery[T, Ts]) CountQuery() ImmutableSelectQuery {
	return q.Query.AsCount()
}

func (q ImmutableViewQuery[T, Ts]) Build(ctx context.Context) (string, []any, error) {
	return q.Query.Build(ctx)
}

func (q ImmutableViewQuery[T, Ts]) mutable() orm.Query[*psqldialect.SelectQuery, T, Ts, bob.SliceTransformer[T, Ts]] {
	mutable := q.Query.state.toMutable()
	return orm.Query[*psqldialect.SelectQuery, T, Ts, bob.SliceTransformer[T, Ts]]{
		ExecQuery: orm.ExecQuery[*psqldialect.SelectQuery]{
			BaseQuery: bob.BaseQuery[*psqldialect.SelectQuery]{
				Expression: &mutable,
				Dialect:    psqldialect.Dialect,
				QueryType:  bob.QueryTypeSelect,
			},
			Hooks: q.Hooks,
		},
		Scanner: q.Scanner,
	}
}

func immutableStateFromMutable(q *psqldialect.SelectQuery) immutableSelectState {
	return immutableSelectState{
		With: clause.With{
			Recursive: q.With.Recursive,
			CTEs:      append([]bob.Expression(nil), q.With.CTEs...),
		},
		SelectColumns:  append([]any(nil), q.SelectList.Columns...),
		PreloadColumns: append([]any(nil), q.SelectList.PreloadColumns...),
		Distinct:       psqldialect.Distinct{On: append([]any(nil), q.Distinct.On...)},
		TableRef:       cloneTableRef(q.TableRef),
		Where:          clause.Where{Conditions: append([]any(nil), q.Where.Conditions...)},
		GroupBy: clause.GroupBy{
			Groups:   append([]any(nil), q.GroupBy.Groups...),
			Distinct: q.GroupBy.Distinct,
			With:     q.GroupBy.With,
		},
		Having: clause.Having{
			Conditions: append([]any(nil), q.Having.Conditions...),
		},
		Windows: clause.Windows{
			Windows: append([]bob.Expression(nil), q.Windows.Windows...),
		},
		Combines: clause.Combines{
			Queries: append([]clause.Combine(nil), q.Combines.Queries...),
		},
		OrderBy: clause.OrderBy{
			Expressions: append([]bob.Expression(nil), q.OrderBy.Expressions...),
		},
		Limit: clause.Limit{Count: q.Limit.Count},
		Offset: clause.Offset{
			Count: q.Offset.Count,
		},
		Fetch: clause.Fetch{
			Count:    q.Fetch.Count,
			WithTies: q.Fetch.WithTies,
		},
		Locks: clause.Locks{
			Locks: append([]bob.Expression(nil), q.Locks.Locks...),
		},
		CombinedOrder: clause.OrderBy{
			Expressions: append([]bob.Expression(nil), q.CombinedOrder.Expressions...),
		},
		CombinedLimit: clause.Limit{Count: q.CombinedLimit.Count},
		CombinedFetch: clause.Fetch{
			Count:    q.CombinedFetch.Count,
			WithTies: q.CombinedFetch.WithTies,
		},
		CombinedOffset: clause.Offset{Count: q.CombinedOffset.Count},
	}
}

func (s immutableSelectState) toMutable() psqldialect.SelectQuery {
	return psqldialect.SelectQuery{
		With:           s.With,
		SelectList:     clause.SelectList{Columns: s.SelectColumns, PreloadColumns: s.PreloadColumns},
		Distinct:       s.Distinct,
		TableRef:       s.TableRef,
		Where:          s.Where,
		GroupBy:        s.GroupBy,
		Having:         s.Having,
		Windows:        s.Windows,
		Combines:       s.Combines,
		OrderBy:        s.OrderBy,
		Limit:          s.Limit,
		Offset:         s.Offset,
		Fetch:          s.Fetch,
		Locks:          s.Locks,
		CombinedOrder:  s.CombinedOrder,
		CombinedLimit:  s.CombinedLimit,
		CombinedFetch:  s.CombinedFetch,
		CombinedOffset: s.CombinedOffset,
	}
}

func (s immutableSelectState) withMods(queryMods ...bob.Mod[*psqldialect.SelectQuery]) (immutableSelectState, bool) {
	next := s
	var cloneSelect, cloneWhere, cloneGroup, cloneHaving, cloneOrder, cloneWindows, cloneLocks bool

	for _, mod := range queryMods {
		switch m := mod.(type) {
		case mods.Select[*psqldialect.SelectQuery]:
			if !cloneSelect {
				next.SelectColumns = append([]any(nil), s.SelectColumns...)
				cloneSelect = true
			}
			next.SelectColumns = append(next.SelectColumns, []any(m)...)
		case mods.Where[*psqldialect.SelectQuery]:
			if !cloneWhere {
				next.Where.Conditions = append([]any(nil), s.Where.Conditions...)
				cloneWhere = true
			}
			next.Where.Conditions = append(next.Where.Conditions, m.E)
		case mods.GroupBy[*psqldialect.SelectQuery]:
			if !cloneGroup {
				next.GroupBy.Groups = append([]any(nil), s.GroupBy.Groups...)
				cloneGroup = true
			}
			next.GroupBy.Groups = append(next.GroupBy.Groups, m.E)
		case mods.Having[*psqldialect.SelectQuery]:
			if !cloneHaving {
				next.Having.Conditions = append([]any(nil), s.Having.Conditions...)
				cloneHaving = true
			}
			next.Having.Conditions = append(next.Having.Conditions, []any(m)...)
		case mods.Limit[*psqldialect.SelectQuery]:
			next.Limit.Count = m.Count
		case mods.Offset[*psqldialect.SelectQuery]:
			next.Offset.Count = m.Count
		case mods.Fetch[*psqldialect.SelectQuery]:
			next.Fetch = clause.Fetch(m)
		case psqldialect.OrderBy[*psqldialect.SelectQuery]:
			if !cloneOrder {
				next.OrderBy.Expressions = append([]bob.Expression(nil), s.OrderBy.Expressions...)
				cloneOrder = true
			}
			next.OrderBy.Expressions = append(next.OrderBy.Expressions, m())
		case mods.NamedWindow[*psqldialect.SelectQuery]:
			if !cloneWindows {
				next.Windows.Windows = append([]bob.Expression(nil), s.Windows.Windows...)
				cloneWindows = true
			}
			next.Windows.Windows = append(next.Windows.Windows, clause.NamedWindow(m))
		case psqldialect.LockChain[*psqldialect.SelectQuery]:
			if !cloneLocks {
				next.Locks.Locks = append([]bob.Expression(nil), s.Locks.Locks...)
				cloneLocks = true
			}
			next.Locks.Locks = append(next.Locks.Locks, m())
		case psqldialect.FromChain[*psqldialect.SelectQuery]:
			next.TableRef = cloneTableRef(m())
		default:
			return next, false
		}
	}

	return next, true
}

func cloneTableRef(from clause.TableRef) clause.TableRef {
	from.Columns = append([]string(nil), from.Columns...)
	from.Partitions = append([]string(nil), from.Partitions...)
	from.IndexHints = append([]clause.IndexHint(nil), from.IndexHints...)
	from.Joins = append([]clause.Join(nil), from.Joins...)
	for i := range from.Joins {
		from.Joins[i].On = append([]bob.Expression(nil), from.Joins[i].On...)
		from.Joins[i].Using = append([]string(nil), from.Joins[i].Using...)
		from.Joins[i].To = cloneTableRef(from.Joins[i].To)
	}
	return from
}

type immutableSelectWriter struct {
	ctx   context.Context
	w     io.StringWriter
	args  []any
	start int
}

func (w *immutableSelectWriter) writeQuery(q immutableSelectState) error {
	if len(q.With.CTEs) > 0 {
		if _, err := q.With.WriteSQL(w.ctx, w.w, psqldialect.Dialect, w.argPos()); err != nil {
			return err
		}
		w.w.WriteString("\n")
	}

	w.w.WriteString("SELECT ")

	if q.Distinct.On != nil {
		w.w.WriteString("DISTINCT")
		if len(q.Distinct.On) > 0 {
			w.w.WriteString(" ON (")
			if err := w.writeSliceAny(q.Distinct.On, ", "); err != nil {
				return err
			}
			w.w.WriteString(")")
		}
		w.w.WriteString(" ")
	}

	w.w.WriteString("\n")
	if len(q.SelectColumns) == 0 && len(q.PreloadColumns) == 0 {
		w.w.WriteString("*")
	} else {
		allCols := append([]any(nil), q.SelectColumns...)
		allCols = append(allCols, q.PreloadColumns...)
		if err := w.writeSliceAny(allCols, ", "); err != nil {
			return err
		}
	}

	if q.TableRef.Expression != nil {
		w.w.WriteString("\nFROM ")
		args, err := q.TableRef.WriteSQL(w.ctx, w.w, psqldialect.Dialect, w.argPos())
		if err != nil {
			return err
		}
		w.args = append(w.args, args...)
	}

	if len(q.Where.Conditions) > 0 {
		w.w.WriteString("\nWHERE ")
		if err := w.writeSliceAny(q.Where.Conditions, " AND "); err != nil {
			return err
		}
	}

	if len(q.GroupBy.Groups) > 0 {
		w.w.WriteString("\nGROUP BY ")
		if q.GroupBy.Distinct {
			w.w.WriteString("DISTINCT ")
		}
		if err := w.writeSliceAny(q.GroupBy.Groups, ", "); err != nil {
			return err
		}
		if q.GroupBy.With != "" {
			w.w.WriteString(" WITH ")
			w.w.WriteString(q.GroupBy.With)
		}
	}

	if len(q.Having.Conditions) > 0 {
		w.w.WriteString("\nHAVING ")
		if err := w.writeSliceAny(q.Having.Conditions, " AND "); err != nil {
			return err
		}
	}

	if len(q.Windows.Windows) > 0 {
		w.w.WriteString("\nWINDOW ")
		if err := w.writeSliceExpr(q.Windows.Windows, ", "); err != nil {
			return err
		}
	}

	if len(q.OrderBy.Expressions) > 0 {
		w.w.WriteString("\nORDER BY ")
		if err := w.writeOrderExprs(q.OrderBy.Expressions); err != nil {
			return err
		}
	}

	if q.Limit.Count != nil {
		w.w.WriteString("\nLIMIT ")
		if err := w.writeAny(q.Limit.Count); err != nil {
			return err
		}
	}

	if q.Offset.Count != nil {
		w.w.WriteString("\nOFFSET ")
		if err := w.writeAny(q.Offset.Count); err != nil {
			return err
		}
	}

	if q.Fetch.Count != nil {
		w.w.WriteString("\nFETCH NEXT ")
		if err := w.writeAny(q.Fetch.Count); err != nil {
			return err
		}
		if q.Fetch.WithTies {
			w.w.WriteString(" ROWS WITH TIES")
		} else {
			w.w.WriteString(" ROWS ONLY")
		}
	}

	for _, lock := range q.Locks.Locks {
		w.w.WriteString("\n")
		if err := w.writeAny(lock); err != nil {
			return err
		}
	}

	w.w.WriteString("\n")
	return nil
}

func (w *immutableSelectWriter) argPos() int {
	return w.start + len(w.args)
}

func (w *immutableSelectWriter) writeSliceAny(values []any, sep string) error {
	for i, value := range values {
		if i > 0 {
			w.w.WriteString(sep)
		}
		if err := w.writeAny(value); err != nil {
			return err
		}
	}
	return nil
}

func (w *immutableSelectWriter) writeSliceExpr(values []bob.Expression, sep string) error {
	for i, value := range values {
		if i > 0 {
			w.w.WriteString(sep)
		}
		if err := w.writeExpression(value); err != nil {
			return err
		}
	}
	return nil
}

func (w *immutableSelectWriter) writeOrderExprs(values []bob.Expression) error {
	for i, value := range values {
		if i > 0 {
			w.w.WriteString(", ")
		}

		switch order := value.(type) {
		case clause.OrderDef:
			if err := w.writeAny(order.Expression); err != nil {
				return err
			}
			if order.Collation != "" {
				w.w.WriteString(" COLLATE ")
				psqldialect.Dialect.WriteQuoted(w.w, order.Collation)
			}
			if order.Direction != "" {
				w.w.WriteString(" ")
				w.w.WriteString(order.Direction)
			}
			if order.Nulls != "" {
				w.w.WriteString(" NULLS ")
				w.w.WriteString(order.Nulls)
			}
		default:
			if err := w.writeExpression(value); err != nil {
				return err
			}
		}
	}
	return nil
}

func (w *immutableSelectWriter) writeExpression(value bob.Expression) error {
	args, err := value.WriteSQL(w.ctx, w.w, psqldialect.Dialect, w.argPos())
	if err != nil {
		return err
	}
	w.args = append(w.args, args...)
	return nil
}

func (w *immutableSelectWriter) writeAny(value any) error {
	switch v := value.(type) {
	case nil:
		w.w.WriteString("NULL")
	case string:
		w.w.WriteString(v)
	case []byte:
		w.w.WriteString(string(v))
	case int:
		w.w.WriteString(strconv.Itoa(v))
	case int8:
		w.w.WriteString(strconv.FormatInt(int64(v), 10))
	case int16:
		w.w.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		w.w.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		w.w.WriteString(strconv.FormatInt(v, 10))
	case uint:
		w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint8:
		w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint16:
		w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		w.w.WriteString(strconv.FormatUint(v, 10))
	case sql.NamedArg:
		return fmt.Errorf("named args are not supported by psql dialect")
	case bob.Expression:
		return w.writeExpression(v)
	default:
		w.w.WriteString(fmt.Sprint(v))
	}

	return nil
}
