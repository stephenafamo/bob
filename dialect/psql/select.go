package psql

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
// https://www.postgresql.org/docs/current/sql-select.html
type selectQuery struct {
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
	clause.Fetch
	clause.For
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

	_, err = bob.ExpressIf(w, d, start+len(args), s.Fetch,
		s.Fetch.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}

	forArgs, err := bob.ExpressIf(w, d, start+len(args), s.For,
		s.For.Strength != "", "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, forArgs...)

	w.Write([]byte("\n"))
	return args, nil
}

type SelectQM struct {
	withMod[*selectQuery]                // For CTEs
	mods.FromMod[*selectQuery]           // select *FROM*
	joinMod[*clause.FromItem]            // joins, which are mods of the FROM
	mods.TableAliasMod[*clause.FromItem] // Adding an alias to from item
	fromItemMod                          // Dialect specific fromItem mods
}

func (SelectQM) Distinct(on ...any) bob.Mod[*selectQuery] {
	return mods.QueryModFunc[*selectQuery](func(q *selectQuery) {
		q.Select.Modifiers = []any{
			distinct{on: on},
		}
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

func (SelectQM) GroupByDistinct(distinct bool) bob.Mod[*selectQuery] {
	return mods.GroupByDistinct[*selectQuery](distinct)
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

func (SelectQM) Fetch(count int64, withTies bool) bob.Mod[*selectQuery] {
	return mods.Fetch[*selectQuery]{
		Count:    &count,
		WithTies: withTies,
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

func (SelectQM) IntersectAll(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Except(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) ExceptAll(q bob.Query) bob.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Except,
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

func (SelectQM) ForNoKeyUpdate(tables ...string) lockChain[*selectQuery] {
	return lockChain[*selectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthNoKeyUpdate,
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

func (SelectQM) ForKeyShare(tables ...string) lockChain[*selectQuery] {
	return lockChain[*selectQuery](func() clause.For {
		return clause.For{
			Strength: clause.LockStrengthKeyShare,
			Tables:   tables,
		}
	})
}
