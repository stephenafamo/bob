package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Select(queryMods ...query.Mod[*selectQuery]) query.BaseQuery[*selectQuery] {
	q := &selectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*selectQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_select.html
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
}

func (s selectQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
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

	fromArgs, err := query.ExpressSlice(w, d, start+len(args), s.FromItems.Items, "\nFROM ", ",\n", "")
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

	limitArgs, err := query.ExpressIf(w, d, start+len(args), s.Limit,
		s.Limit.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, limitArgs...)

	offsetArgs, err := query.ExpressIf(w, d, start+len(args), s.Offset,
		s.Offset.Count != nil, "\n", "")
	if err != nil {
		return nil, err
	}
	args = append(args, offsetArgs...)

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

func (SelectQM) Distinct() query.Mod[*selectQuery] {
	return mods.Distinct[*selectQuery]{
		Distinct: true,
	}
}

func (SelectQM) Select(clauseessions ...any) query.Mod[*selectQuery] {
	return mods.Select[*selectQuery](clauseessions)
}

func (SelectQM) Where(e query.Expression) query.Mod[*selectQuery] {
	return mods.Where[*selectQuery]{e}
}

func (qm SelectQM) WhereClause(clause string, args ...any) query.Mod[*selectQuery] {
	return mods.Where[*selectQuery]{Raw(clause, args...)}
}

func (SelectQM) Having(e query.Expression) query.Mod[*selectQuery] {
	return mods.Having[*selectQuery]{e}
}

func (qm SelectQM) HavingClause(clause string, args ...any) query.Mod[*selectQuery] {
	return mods.Having[*selectQuery]{Raw(clause, args...)}
}

func (SelectQM) GroupBy(e any) query.Mod[*selectQuery] {
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
func (SelectQM) Limit(count any) query.Mod[*selectQuery] {
	return mods.Limit[*selectQuery]{
		Count: count,
	}
}

// Sqlite can use an clauseession for the offset
func (SelectQM) Offset(count any) query.Mod[*selectQuery] {
	return mods.Offset[*selectQuery]{
		Count: count,
	}
}

func (SelectQM) Union(q query.Query) query.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) UnionAll(q query.Query) query.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Intersect(q query.Query) query.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) Except(q query.Query) query.Mod[*selectQuery] {
	return mods.Combine[*selectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}
