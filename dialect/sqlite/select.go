package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/mods"
	"github.com/stephenafamo/bob/query"
)

func Select(queryMods ...query.Mod[*SelectQuery]) query.BaseQuery[*SelectQuery] {
	q := &SelectQuery{}
	for _, mod := range queryMods {
		mod.Apply(q)
	}

	return query.BaseQuery[*SelectQuery]{
		Expression: q,
		Dialect:    dialect,
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_select.html
type SelectQuery struct {
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

func (s SelectQuery) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
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
	builderMod
	withMod[*SelectQuery]                // For CTEs
	mods.FromMod[*SelectQuery]           // select *FROM*
	joinMod[*clause.FromItem]            // joins, which are mods of the FROM
	mods.TableAliasMod[*clause.FromItem] // Adding an alias to from item
	fromItemMod                          // Dialect specific fromItem mods
}

func (SelectQM) Distinct() query.Mod[*SelectQuery] {
	return mods.Distinct[*SelectQuery]{
		Distinct: true,
	}
}

func (SelectQM) Select(clauseessions ...any) query.Mod[*SelectQuery] {
	return mods.Select[*SelectQuery](clauseessions)
}

func (SelectQM) Where(e query.Expression) query.Mod[*SelectQuery] {
	return mods.Where[*SelectQuery]{e}
}

func (qm SelectQM) WhereClause(clause string, args ...any) query.Mod[*SelectQuery] {
	return mods.Where[*SelectQuery]{qm.Raw(clause, args...)}
}

func (SelectQM) Having(e query.Expression) query.Mod[*SelectQuery] {
	return mods.Having[*SelectQuery]{e}
}

func (qm SelectQM) HavingClause(clause string, args ...any) query.Mod[*SelectQuery] {
	return mods.Having[*SelectQuery]{qm.Raw(clause, args...)}
}

func (SelectQM) GroupBy(e any) query.Mod[*SelectQuery] {
	return mods.GroupBy[*SelectQuery]{
		E: e,
	}
}

func (SelectQM) Window(name string) *windowChain[*SelectQuery] {
	return &windowChain[*SelectQuery]{
		name: name,
	}
}

func (SelectQM) OrderBy(e any) query.Mod[*SelectQuery] {
	return orderBy[*SelectQuery](func() clause.OrderDef {
		return clause.OrderDef{
			Expression: e,
		}
	})
}

// Sqlite can use an clauseession for the limit
func (SelectQM) Limit(count any) query.Mod[*SelectQuery] {
	return mods.Limit[*SelectQuery]{
		Count: count,
	}
}

// Sqlite can use an clauseession for the offset
func (SelectQM) Offset(count any) query.Mod[*SelectQuery] {
	return mods.Offset[*SelectQuery]{
		Count: count,
	}
}

func (SelectQM) Union(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) UnionAll(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: clause.Union,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Intersect(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: clause.Intersect,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) Except(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: clause.Except,
		Query:    q,
		All:      false,
	}
}
