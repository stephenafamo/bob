package sqlite

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Select(queryMods ...bob.Mod[*selectQuery]) bob.BaseQuery[*selectQuery] {
	q := &selectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*selectQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_select.html
type selectQuery struct {
	clause.With
	clause.Select
	clause.From
	clause.Where
	clause.GroupBy
	clause.Having
	clause.Windows
	clause.Combine
	clause.OrderBy
	clause.Limit
	clause.Offset
}

func (s selectQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, d, start+len(args), s.With,
		len(s.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	selArgs, err := bob.ExpressIf(w, d, start+len(args), s.Select, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, selArgs...)

	fromArgs, err := bob.ExpressIf(w, d, start+len(args), s.From, true, "\nFROM ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, fromArgs...)

	whereArgs, err := bob.ExpressIf(w, d, start+len(args), s.Where,
		len(s.Where.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, whereArgs...)

	groupByArgs, err := bob.ExpressIf(w, d, start+len(args), s.GroupBy,
		len(s.GroupBy.Groups) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, groupByArgs...)

	havingArgs, err := bob.ExpressIf(w, d, start+len(args), s.Having,
		len(s.Having.Conditions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, havingArgs...)

	windowArgs, err := bob.ExpressIf(w, d, start+len(args), s.Windows,
		len(s.Windows.Windows) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, windowArgs...)

	combineArgs, err := bob.ExpressIf(w, d, start+len(args), s.Combine,
		s.Combine.Query != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, combineArgs...)

	orderArgs, err := bob.ExpressIf(w, d, start+len(args), s.OrderBy,
		len(s.OrderBy.Expressions) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	limitArgs, err := bob.ExpressIf(w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, limitArgs...)

	offsetArgs, err := bob.ExpressIf(w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, offsetArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

type SelectQM struct {
	withMod[*selectQuery]     // For CTEs
	joinMod[*clause.From]     // joins, which are mods of the FROM
	fromItemMod[*selectQuery] // Dialect specific fromItem mods
}

func (SelectQM) Distinct() bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = []any{"DISTINCT"}
	})
}

func (SelectQM) Columns(clauses ...any) bob.Mod[*selectQuery] {
	return mods.Select[*selectQuery](clauses)
}

func (SelectQM) From(table any) bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.SetTable(table)
	})
}

func (SelectQM) Where(e bob.Expression) bob.Mod[*selectQuery] {
	return mods.Where[*selectQuery]{e}
}

func (qm SelectQM) WhereClause(clause string, args ...any) bob.Mod[*selectQuery] {
	return mods.Where[*selectQuery]{Raw(clause, args...)}
}

func (SelectQM) Having(e bob.Expression) bob.Mod[*selectQuery] {
	return mods.Having[*selectQuery]{e}
}

func (qm SelectQM) HavingClause(clause string, args ...any) bob.Mod[*selectQuery] {
	return mods.Having[*selectQuery]{Raw(clause, args...)}
}

func (SelectQM) GroupBy(e any) bob.Mod[*selectQuery] {
	return mods.GroupBy[*selectQuery]{
		E: e,
	}
}

func (SelectQM) Window(name string) windowMod[*selectQuery] {
	m := windowMod[*selectQuery]{
		name: name,
	}

	m.windowChain.def = &m
	return m
}

func (SelectQM) OrderBy(e any) orderBy[*selectQuery] {
	return orderBy[*selectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// Sqlite can use an clauseession for the limit
func (SelectQM) Limit(count any) bob.Mod[*selectQuery] {
	return mods.Limit[*selectQuery]{
		Count: count,
	}
}

// Sqlite can use an clauseession for the offset
func (SelectQM) Offset(count any) bob.Mod[*selectQuery] {
	return mods.Offset[*selectQuery]{
		Count: count,
	}
}

func (SelectQM) Union(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) UnionAll(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Intersect(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) Except(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}
