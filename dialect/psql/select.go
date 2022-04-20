package psql

import (
	"io"

	"github.com/stephenafamo/typesql/expr"
	"github.com/stephenafamo/typesql/mods"
	"github.com/stephenafamo/typesql/query"
)

func Select(mods ...mods.QueryMod[*SelectQuery]) *SelectQuery {
	s := &SelectQuery{}
	for _, mod := range mods {
		mod.Apply(s)
	}

	return s
}

// Trying to represent the select query structure as documented in
// https://www.postgresql.org/docs/current/sql-select.html
type SelectQuery struct {
	expr.With
	expr.Select
	expr.From
	expr.Where
	expr.GroupBy
	expr.Having
	expr.Windows
	expr.Combine
	expr.OrderBy
	expr.Limit
	expr.Offset
	expr.Fetch
	expr.For
}

func (s SelectQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	return s.WriteSQL(w, dialect, start)
}

func (s SelectQuery) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := query.ExpressIf(w, d, start+len(args), s.With,
		len(s.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	selArgs, err := query.ExpressIf(w, d, start+len(args), s.Select, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, selArgs...)

	fromArgs, err := query.ExpressIf(w, d, start+len(args), s.From, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := query.ExpressIf(w, d, start+len(args), s.Where,
		len(s.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	groupByArgs, err := query.ExpressIf(w, d, start+len(args), s.GroupBy,
		len(s.GroupBy.Groups) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, groupByArgs...)

	havingArgs, err := query.ExpressIf(w, d, start+len(args), s.Having,
		len(s.Having.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, havingArgs...)

	windowArgs, err := query.ExpressIf(w, d, start+len(args), s.Windows,
		len(s.Windows.Windows) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, windowArgs...)

	combineArgs, err := query.ExpressIf(w, d, start+len(args), s.Combine,
		s.Combine.Query != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combineArgs...)

	orderArgs, err := query.ExpressIf(w, d, start+len(args), s.OrderBy,
		len(s.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	_, err = query.ExpressIf(w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	_, err = query.ExpressIf(w, d, start+len(args), s.Fetch,
		s.Fetch.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	forArgs, err := query.ExpressIf(w, d, start+len(args), s.For,
		s.For.Strength != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, forArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

type SelectQM struct{}

func (qm SelectQM) With(q query.Query, name string, columns ...string) mods.QueryMod[*SelectQuery] {
	return mods.With[*SelectQuery]{
		Query:   q,
		Name:    name,
		Columns: columns,
	}
}

func (qm SelectQM) WithRecursive(q query.Query, name string, columns ...string) mods.QueryMod[*SelectQuery] {
	return mods.With[*SelectQuery]{
		Query:     q,
		Name:      name,
		Columns:   columns,
		Recursive: true,
	}
}

func (qm SelectQM) WithMaterialized(q query.Query, name string, columns ...string) mods.QueryMod[*SelectQuery] {
	return mods.With[*SelectQuery]{
		Query:        q,
		Name:         name,
		Columns:      columns,
		Materialized: true,
	}
}

func (qm SelectQM) WithCTE(cte expr.CTE) mods.QueryMod[*SelectQuery] {
	return mods.With[*SelectQuery](cte)
}

func (qm SelectQM) Distinct(expressions ...any) mods.QueryMod[*SelectQuery] {
	return mods.Distinct[*SelectQuery]{
		Distinct: true,
		On:       expressions,
	}
}

func (qm SelectQM) Select(expressions ...any) mods.QueryMod[*SelectQuery] {
	return mods.Select[*SelectQuery](expressions)
}

func (qm SelectQM) From(expression any) mods.QueryMod[*SelectQuery] {
	return mods.From[*SelectQuery](expr.T(expression))
}

// For easy migration from sqlboiler/v4
func (qm SelectQM) InnerJoin(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.InnerJoin,
		To:   expr.Statement(clause, args),
	}
}

func (qm SelectQM) InnerJoinOn(to any, on ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.InnerJoin,
		On:   on,
	}
}

func (qm SelectQM) InnerJoinUsing(to any, using ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:  expr.InnerJoin,
		Using: using,
	}
}

func (qm SelectQM) InnerJoinNatural(to any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:    expr.InnerJoin,
		Natural: true,
	}
}

func (qm SelectQM) LeftJoin(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.LeftJoin,
		To:   expr.Statement(clause, args),
	}
}

func (qm SelectQM) LeftJoinOn(to any, on ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.LeftJoin,
		On:   on,
	}
}

func (qm SelectQM) LeftJoinUsing(to any, using ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:  expr.LeftJoin,
		Using: using,
	}
}

func (qm SelectQM) LeftJoinNatural(to any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:    expr.LeftJoin,
		Natural: true,
	}
}

func (qm SelectQM) RightJoin(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.RightJoin,
		To:   expr.Statement(clause, args),
	}
}

func (qm SelectQM) RightJoinOn(to any, on ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.RightJoin,
		On:   on,
	}
}

func (qm SelectQM) RightJoinUsing(to any, using ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:  expr.RightJoin,
		Using: using,
	}
}

func (qm SelectQM) RightJoinNatural(to any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:    expr.RightJoin,
		Natural: true,
	}
}

func (qm SelectQM) FullJoin(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.FullJoin,
		To:   expr.Statement(clause, args),
	}
}

func (qm SelectQM) FullJoinOn(to any, on ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.FullJoin,
		On:   on,
	}
}

func (qm SelectQM) FullJoinUsing(to any, using ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:  expr.FullJoin,
		Using: using,
	}
}

func (qm SelectQM) FullJoinNatural(to any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type:    expr.FullJoin,
		Natural: true,
	}
}

func (qm SelectQM) CrossJoin(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Join[*SelectQuery]{
		Type: expr.CrossJoin,
		To:   expr.Statement(clause, args),
	}
}

func (qm SelectQM) Where(e query.Expression) mods.QueryMod[*SelectQuery] {
	return mods.Where[*SelectQuery]{e}
}

func (qm SelectQM) WhereClause(clause string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.Where[*SelectQuery]{expr.Statement(clause, args...)}
}

func (qm SelectQM) GroupBy(e any) mods.QueryMod[*SelectQuery] {
	return mods.GroupBy[*SelectQuery]{
		E: e,
	}
}

func (qm SelectQM) GroupByDistinct(distinct bool) mods.QueryMod[*SelectQuery] {
	return mods.GroupByDistinct[*SelectQuery](distinct)
}

func (qm SelectQM) Window(name string, definition any) mods.QueryMod[*SelectQuery] {
	return mods.Window[*SelectQuery]{
		Name:      name,
		Definiton: definition,
	}
}

// For easy upgrade from sqlboiler/v4
func (qm SelectQM) OrderBy(stmt string, args ...any) mods.QueryMod[*SelectQuery] {
	return mods.OrderBy[*SelectQuery]{
		Expression: expr.Statement(stmt, args...),
	}
}

func (qm SelectQM) OrderByAsc(e any) mods.QueryMod[*SelectQuery] {
	return mods.OrderBy[*SelectQuery]{
		Expression: e,
		Direction:  "ASC",
	}
}

func (qm SelectQM) OrderByDesc(e any) mods.QueryMod[*SelectQuery] {
	return mods.OrderBy[*SelectQuery]{
		Expression: e,
		Direction:  "DESC",
	}
}

func (qm SelectQM) OrderByUsing(e any, operator string) mods.QueryMod[*SelectQuery] {
	return mods.OrderBy[*SelectQuery]{
		Expression: e,
		Direction:  "USING " + operator,
	}
}

func (qm SelectQM) OrderByDef(order expr.OrderDef) mods.QueryMod[*SelectQuery] {
	return mods.OrderBy[*SelectQuery](order)
}

func (qm SelectQM) Limit(count int64) mods.QueryMod[*SelectQuery] {
	return mods.Limit[*SelectQuery]{
		Count: &count,
	}
}

func (qm SelectQM) Offset(count int64) mods.QueryMod[*SelectQuery] {
	return mods.Offset[*SelectQuery]{
		Count: &count,
	}
}

func (qm SelectQM) Fetch(count int64, withTies bool) mods.QueryMod[*SelectQuery] {
	return mods.Fetch[*SelectQuery]{
		Count:    &count,
		WithTies: withTies,
	}
}

func (qm SelectQM) Union(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Union,
		Query:    q,
		All:      false,
	}
}

func (qm SelectQM) UnionAll(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Union,
		Query:    q,
		All:      true,
	}
}

func (qm SelectQM) Intersect(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Intersect,
		Query:    q,
		All:      false,
	}
}

func (qm SelectQM) IntersectAll(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Intersect,
		Query:    q,
		All:      true,
	}
}

func (qm SelectQM) Except(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Except,
		Query:    q,
		All:      false,
	}
}

func (qm SelectQM) ExceptAll(q query.Query) mods.QueryMod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Except,
		Query:    q,
		All:      true,
	}
}

func (qm SelectQM) ForUpdate(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthUpdate,
		Tables:   tables,
	})
}

func (qm SelectQM) ForUpdateNoWait(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthUpdate,
		Tables:   tables,
		Wait:     expr.LockWaitNoWait,
	})
}

func (qm SelectQM) ForUpdateSkipLocked(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthUpdate,
		Tables:   tables,
		Wait:     expr.LockWaitSkipLocked,
	})
}

func (qm SelectQM) ForNoKeyUpdate(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthNoKeyUpdate,
		Tables:   tables,
	})
}

func (qm SelectQM) ForNoKeyUpdateNoWait(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthNoKeyUpdate,
		Tables:   tables,
		Wait:     expr.LockWaitNoWait,
	})
}

func (qm SelectQM) ForNoKeyUpdateSkipLocked(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthNoKeyUpdate,
		Tables:   tables,
		Wait:     expr.LockWaitSkipLocked,
	})
}

func (qm SelectQM) ForShare(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthShare,
		Tables:   tables,
	})
}

func (qm SelectQM) ForShareNoWait(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthShare,
		Tables:   tables,
		Wait:     expr.LockWaitNoWait,
	})
}

func (qm SelectQM) ForShareSkipLocked(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthShare,
		Tables:   tables,
		Wait:     expr.LockWaitSkipLocked,
	})
}

func (qm SelectQM) ForKeyShare(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthKeyShare,
		Tables:   tables,
	})
}

func (qm SelectQM) ForKeyShareNoWait(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthKeyShare,
		Tables:   tables,
		Wait:     expr.LockWaitNoWait,
	})
}

func (qm SelectQM) ForKeyShareSkipLocked(tables ...string) mods.QueryMod[*SelectQuery] {
	return mods.For[*SelectQuery](expr.For{
		Strength: expr.LockStrengthKeyShare,
		Tables:   tables,
		Wait:     expr.LockWaitSkipLocked,
	})
}
