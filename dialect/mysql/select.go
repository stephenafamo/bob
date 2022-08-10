package mysql

import (
	"io"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
)

func Select(queryMods ...bob.Mod[*SelectQuery]) bob.BaseQuery[*SelectQuery] {
	q := &SelectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return bob.BaseQuery[*SelectQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the query structure as documented in
// https://dev.mysql.com/doc/refman/8.0/en/select.html
type SelectQuery struct {
	hints
	modifiers[any]
	into *into

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
	clause.For
}

func (s *SelectQuery) setInto(i into) {
	s.into = &i
}

func (s SelectQuery) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
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

//nolint:gochecknoglobals
var SelectQM = selectQM{}

type selectQM struct {
	hintMod[*SelectQuery]     // for optimizer hints
	withMod[*SelectQuery]     // For CTEs
	joinMod[*clause.From]     // joins, which are mods of the FROM
	fromItemMod[*SelectQuery] // Dialect specific fromItem mods
	intoMod[*SelectQuery]     // INTO clause
}

func (selectQM) Distinct(on ...any) bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "DISTINCT")
	})
}

func (selectQM) HighPriority() bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "HIGH_PRIORITY")
	})
}

func (selectQM) Straight() bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "STRAIGHT_JOIN")
	})
}

func (selectQM) SmallResult() bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "SQL_SMALL_RESULT")
	})
}

func (selectQM) BigResult() bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "SQL_BIG_RESULT")
	})
}

func (selectQM) BufferResult() bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.Select.Modifiers = append(q.Select.Modifiers, "SQL_BUFFER_RESULT")
	})
}

func (selectQM) Columns(clauses ...any) bob.Mod[*SelectQuery] {
	return mods.Select[*SelectQuery](clauses)
}

func (selectQM) From(table any) bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.SetTable(table)
	})
}

func (selectQM) Where(e bob.Expression) bob.Mod[*SelectQuery] {
	return mods.Where[*SelectQuery]{e}
}

func (qm selectQM) WhereClause(clause string, args ...any) bob.Mod[*SelectQuery] {
	return mods.Where[*SelectQuery]{Raw(clause, args...)}
}

func (selectQM) Having(e bob.Expression) bob.Mod[*SelectQuery] {
	return mods.Having[*SelectQuery]{e}
}

func (qm selectQM) HavingClause(clause string, args ...any) bob.Mod[*SelectQuery] {
	return mods.Having[*SelectQuery]{Raw(clause, args...)}
}

func (selectQM) GroupBy(e any) bob.Mod[*SelectQuery] {
	return mods.GroupBy[*SelectQuery]{
		E: e,
	}
}

func (selectQM) WithRollup(distinct bool) bob.Mod[*SelectQuery] {
	return mods.QueryModFunc[*SelectQuery](func(q *SelectQuery) {
		q.SetGroupWith("ROLLUP")
	})
}

func (selectQM) Window(name string) windowMod[*SelectQuery] {
	m := windowMod[*SelectQuery]{
		name: name,
	}

	m.windowChain.def = &m
	return m
}

func (selectQM) OrderBy(e any) orderBy[*SelectQuery] {
	return orderBy[*SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

func (selectQM) Limit(count int64) bob.Mod[*SelectQuery] {
	return mods.Limit[*SelectQuery]{
		Count: count,
	}
}

func (selectQM) Offset(count int64) bob.Mod[*SelectQuery] {
	return mods.Offset[*SelectQuery]{
		Count: count,
	}
}

func (selectQM) Union(q bob.Query) bob.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func (selectQM) UnionAll(q bob.Query) bob.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func (selectQM) ForUpdate(tables ...string) lockChain[*SelectQuery] {
	return lockChain[*SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthUpdate,
			Tables:   tables,
		}
	})
}

func (selectQM) ForShare(tables ...string) lockChain[*SelectQuery] {
	return lockChain[*SelectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthShare,
			Tables:   tables,
		}
	})
}
