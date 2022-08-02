package mysql

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

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/select.html
type selectQuery struct {
	hints
	modifiers[any]
	into *into

	clause.With
	clause.Select
	clause.FromItems
	clause.Where
	clause.GroupBy
	clause.Having
	clause.Windows

	clause.Combine
	clause.OrderBy
	clause.Limit
	clause.Offset
	clause.For
}

func (s *selectQuery) setInto(i into) {
	s.into = &i
}

func (s selectQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	withArgs, err := bob.ExpressIf(w, d, start+len(args), s.With,
		len(s.With.CTEs) > 0, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, withArgs...)

	// Add hints as the first modifier to the select clause
	s.Select.Modifiers = append(s.modifiers.modifiers, s.Select.Modifiers...)

	// Add hints first if any exists
	if len(s.hints.hints) > 0 {
		s.Select.Modifiers = append([]any{s.hints}, s.Select.Modifiers...)
	}
	selArgs, err := bob.ExpressIf(w, d, start+len(args), s.Select, true, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, selArgs...)

	fromArgs, err := bob.ExpressSlice(w, d, start+len(args), s.FromItems.Items, "\nFROM ", ",\n", "")
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

	_, err = bob.ExpressIf(w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	_, err = bob.ExpressIf(w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	forArgs, err := bob.ExpressIf(w, d, start+len(args), s.For,
		s.For.Strength != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, forArgs...)

	intoArgs, err := bob.ExpressIf(w, d, start+len(args), s.into,
		s.into != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, intoArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

type SelectQM struct {
	hintMod[*selectQuery]      // for optimizer hints
	withMod[*selectQuery]      // For CTEs
	mods.FromMod[*selectQuery] // select *FROM*
	joinMod[*clause.FromItem]  // joins, which are mods of the FROM
	fromItemMod                // Dialect specific fromItem mods
	intoMod[*selectQuery]      // INTO clause
}

func (SelectQM) Distinct(on ...any) bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "DISTINCT")
	})
}

func (SelectQM) HighPriority() bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "HIGH_PRIORITY")
	})
}

func (SelectQM) Straight() bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "STRAIGHT_JOIN")
	})
}

func (SelectQM) SmallResult() bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "SQL_SMALL_RESULT")
	})
}

func (SelectQM) BigResult() bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "SQL_BIG_RESULT")
	})
}

func (SelectQM) BufferResult() bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "SQL_BUFFER_RESULT")
	})
}

func (SelectQM) Columns(clauses ...any) bob.Mod[*selectQuery] {
	return mods.Select[*selectQuery](clauses)
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

func (SelectQM) WithRollup(distinct bool) bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.SetGroupWith("ROLLUP")
	})
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

func (SelectQM) Limit(count int64) bob.Mod[*selectQuery] {
	return mods.Limit[*selectQuery]{
		Count: count,
	}
}

func (SelectQM) Offset(count int64) bob.Mod[*selectQuery] {
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

func (SelectQM) ForUpdate(tables ...string) lockChain[*selectQuery] {
	return lockChain[*selectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

func (SelectQM) ForShare(tables ...string) lockChain[*selectQuery] {
	return lockChain[*selectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}
