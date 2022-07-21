package sqlite

import (
	"io"

	"github.com/stephenafamo/bob/expr"
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
		Dialect:    Dialect{},
	}
}

// Trying to represent the select query structure as documented in
// https://www.sqlite.org/lang_select.html
type SelectQuery struct {
	expr.With
	expr.Select
	expr.FromItems
	expr.Where
	expr.GroupBy
	expr.Having
	expr.Windows
	expr.Combine
	expr.OrderBy
	expr.Limit
	expr.Offset
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
	withMod[*SelectQuery]              // For CTEs
	mods.FromMod[*SelectQuery]         // select *FROM*
	joinMod[*expr.FromItem]            // joins, which are mods of the FROM
	mods.TableAliasMod[*expr.FromItem] // Adding an alias to from item
	fromItemMod                        // Dialect specific fromItem mods
}

func (SelectQM) Distinct() query.Mod[*SelectQuery] {
	return mods.Distinct[*SelectQuery]{
		Distinct: true,
	}
}

func (SelectQM) Select(expressions ...any) query.Mod[*SelectQuery] {
	return mods.Select[*SelectQuery](expressions)
}

func (SelectQM) Where(e query.Expression) query.Mod[*SelectQuery] {
	return mods.Where[*SelectQuery]{e}
}

func (SelectQM) WhereClause(clause string, args ...any) query.Mod[*SelectQuery] {
	return mods.Where[*SelectQuery]{expr.Statement(clause, args...)}
}

func (SelectQM) Having(e query.Expression) query.Mod[*SelectQuery] {
	return mods.Having[*SelectQuery]{e}
}

func (SelectQM) HavingClause(clause string, args ...any) query.Mod[*SelectQuery] {
	return mods.Having[*SelectQuery]{expr.Statement(clause, args...)}
}

func (SelectQM) GroupBy(e any) query.Mod[*SelectQuery] {
	return mods.GroupBy[*SelectQuery]{
		E: e,
	}
}

func (SelectQM) Window(name string, definition any) query.Mod[*SelectQuery] {
	return mods.Window[*SelectQuery]{
		Name:      name,
		Definiton: definition,
	}
}

func (SelectQM) OrderBy(e any) query.Mod[*SelectQuery] {
	return orderBy[*SelectQuery](func() expr.OrderDef {
		return expr.OrderDef{
			Expression: e,
		}
	})
}

// Sqlite can use an expression for the limit
func (SelectQM) Limit(count any) query.Mod[*SelectQuery] {
	return mods.Limit[*SelectQuery]{
		Count: count,
	}
}

// Sqlite can use an expression for the offset
func (SelectQM) Offset(count any) query.Mod[*SelectQuery] {
	return mods.Offset[*SelectQuery]{
		Count: count,
	}
}

func (SelectQM) Union(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Union,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) UnionAll(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Union,
		Query:    q,
		All:      true,
	}
}

func (SelectQM) Intersect(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Intersect,
		Query:    q,
		All:      false,
	}
}

func (SelectQM) Except(q query.Query) query.Mod[*SelectQuery] {
	return mods.Combine[*SelectQuery]{
		Strategy: expr.Except,
		Query:    q,
		All:      false,
	}
}
