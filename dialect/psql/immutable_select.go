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
	"github.com/stephenafamo/scan"
)

type derivedSelectQuery struct {
	state immutableSelectState
	load  bob.Load
	hooks bob.EmbeddedHook
}

type immutableSelectState struct {
	DefaultSelectColumns []any
	With                 clause.With
	SelectColumns        []any
	PreloadColumns       []any
	Distinct             psqldialect.Distinct
	TableRef             clause.TableRef
	Where                clause.Where
	GroupBy              clause.GroupBy
	Having               clause.Having
	Windows              clause.Windows
	Combines             clause.Combines
	OrderBy              clause.OrderBy
	Limit                clause.Limit
	Offset               clause.Offset
	Fetch                clause.Fetch
	Locks                clause.Locks

	CombinedOrder  clause.OrderBy
	CombinedLimit  clause.Limit
	CombinedFetch  clause.Fetch
	CombinedOffset clause.Offset
}

func asImmutable(q bob.BaseQuery[*psqldialect.SelectQuery]) derivedSelectQuery {
	return derivedSelectQuery{
		state: immutableStateFromMutable(q.Expression),
		load:  q.Expression.Load,
		hooks: q.Expression.EmbeddedHook,
	}
}

func (q derivedSelectQuery) Type() bob.QueryType {
	return bob.QueryTypeSelect
}

func (q derivedSelectQuery) With(queryMods ...bob.Mod[*psqldialect.SelectQuery]) derivedSelectQuery {
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

	return asImmutable(base)
}

func (q derivedSelectQuery) AsCount() derivedSelectQuery {
	next := q.state
	next.SelectColumns = []any{"count(1)"}
	next.DefaultSelectColumns = nil
	next.PreloadColumns = nil
	next.OrderBy.Expressions = nil
	next.GroupBy.Groups = nil
	next.GroupBy.With = ""
	next.GroupBy.Distinct = false
	next.Offset.Count = nil
	next.Limit.Count = 1

	q.state = next
	return q
}

func (q derivedSelectQuery) Build(ctx context.Context) (string, []any, error) {
	return q.BuildN(ctx, 1)
}

func (q derivedSelectQuery) BuildN(ctx context.Context, start int) (string, []any, error) {
	var sb strings.Builder
	args, err := q.WriteQuery(ctx, &sb, start)
	if err != nil {
		return "", nil, err
	}

	return sb.String(), args, nil
}

func (q derivedSelectQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	if q.requiresMutableWrite() {
		return q.mutableBase().WriteQuery(ctx, w, start)
	}

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

func (q derivedSelectQuery) requiresMutableWrite() bool {
	return !q.state.supportsNativeWrite()
}

func (q derivedSelectQuery) WriteSQL(ctx context.Context, w io.StringWriter, _ bob.Dialect, start int) ([]any, error) {
	w.WriteString("(")
	args, err := q.WriteQuery(ctx, w, start)
	if err != nil {
		return nil, err
	}
	w.WriteString(")")
	return args, nil
}

func (q derivedSelectQuery) Exec(ctx context.Context, exec bob.Executor) (sql.Result, error) {
	return bob.Exec(ctx, exec, q)
}

func (q derivedSelectQuery) RunHooks(ctx context.Context, exec bob.Executor) (context.Context, error) {
	return q.hooks.RunHooks(ctx, exec)
}

func (q derivedSelectQuery) GetLoaders() []bob.Loader {
	return q.load.GetLoaders()
}

func (q derivedSelectQuery) GetMapperMods() []scan.MapperMod {
	return q.load.GetMapperMods()
}

func (q derivedSelectQuery) mutableBase() bob.BaseQuery[*psqldialect.SelectQuery] {
	mutable := &psqldialect.SelectQuery{
		With:       q.state.With,
		SelectList: clause.SelectList{Columns: q.state.selectColumns(), PreloadColumns: q.state.PreloadColumns},
		Distinct:   q.state.Distinct,
		TableRef:   q.state.TableRef,
		Where:      q.state.Where,
		GroupBy:    q.state.GroupBy,
		Having:     q.state.Having,
		Windows:    q.state.Windows,
		Combines:   q.state.Combines,
		OrderBy:    q.state.OrderBy,
		Limit:      q.state.Limit,
		Offset:     q.state.Offset,
		Fetch:      q.state.Fetch,
		Locks:      q.state.Locks,

		Load:         q.load,
		EmbeddedHook: q.hooks,

		CombinedOrder:  q.state.CombinedOrder,
		CombinedLimit:  q.state.CombinedLimit,
		CombinedFetch:  q.state.CombinedFetch,
		CombinedOffset: q.state.CombinedOffset,
	}

	return bob.BaseQuery[*psqldialect.SelectQuery]{
		Expression: mutable,
		Dialect:    psqldialect.Dialect,
		QueryType:  bob.QueryTypeSelect,
	}
}

func immutableStateFromMutable(q *psqldialect.SelectQuery) immutableSelectState {
	return immutableSelectState{
		DefaultSelectColumns: nil,
		With: clause.With{
			Recursive: q.With.Recursive,
			CTEs:      append([]bob.Expression(nil), q.With.CTEs...),
		},
		SelectColumns:  append([]any(nil), q.SelectList.Columns...),
		PreloadColumns: append([]any(nil), q.SelectList.PreloadColumns...),
		Distinct:       psqldialect.Distinct{On: cloneAnySlice(q.Distinct.On)},
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

func (s immutableSelectState) supportsNativeWrite() bool {
	return true
}

func (s immutableSelectState) selectColumns() []any {
	if len(s.SelectColumns) > 0 {
		return s.SelectColumns
	}
	return s.DefaultSelectColumns
}

func cloneAnySlice(values []any) []any {
	if values == nil {
		return nil
	}
	return append(make([]any, 0, len(values)), values...)
}

func (s immutableSelectState) toMutable() psqldialect.SelectQuery {
	return psqldialect.SelectQuery{
		With:           s.With,
		SelectList:     clause.SelectList{Columns: s.selectColumns(), PreloadColumns: s.PreloadColumns},
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

	needsParens := len(q.Combines.Queries) > 0 &&
		(len(q.OrderBy.Expressions) > 0 ||
			q.Limit.Count != nil ||
			q.Offset.Count != nil ||
			q.Fetch.Count != nil ||
			len(q.Locks.Locks) > 0)

	if needsParens {
		w.w.WriteString("(")
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
	selectColumns := q.selectColumns()
	if len(selectColumns) == 0 && len(q.PreloadColumns) == 0 {
		w.w.WriteString("*")
	} else {
		allCols := append([]any(nil), selectColumns...)
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

	if needsParens {
		w.w.WriteString(")")
	}

	for _, combine := range q.Combines.Queries {
		w.w.WriteString("\n")
		args, err := combine.WriteSQL(w.ctx, w.w, psqldialect.Dialect, w.argPos())
		if err != nil {
			return err
		}
		w.args = append(w.args, args...)
	}

	if len(q.CombinedOrder.Expressions) > 0 {
		w.w.WriteString("\nORDER BY ")
		if err := w.writeOrderExprs(q.CombinedOrder.Expressions); err != nil {
			return err
		}
	}

	if q.CombinedLimit.Count != nil {
		w.w.WriteString("\nLIMIT ")
		if err := w.writeAny(q.CombinedLimit.Count); err != nil {
			return err
		}
	}

	if q.CombinedOffset.Count != nil {
		w.w.WriteString("\nOFFSET ")
		if err := w.writeAny(q.CombinedOffset.Count); err != nil {
			return err
		}
	}

	if q.CombinedFetch.Count != nil {
		w.w.WriteString("\nFETCH NEXT ")
		if err := w.writeAny(q.CombinedFetch.Count); err != nil {
			return err
		}
		if q.CombinedFetch.WithTies {
			w.w.WriteString(" ROWS WITH TIES")
		} else {
			w.w.WriteString(" ROWS ONLY")
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
