package genruntime

import (
	"strconv"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/orm"
)

type queryHop struct {
	Table clause.TableRef
	On    []bob.Expression
}

// QueryPath reserves each table's usual alias for its last occurrence so
// filters can address the target. A correlated query also reserves its parent.
func QueryPath[E bob.Expression](rel orm.PreloadRel[E]) []queryHop {
	return queryPath(rel, "")
}

func queryPath[E bob.Expression](rel orm.PreloadRel[E], parent string) []queryHop {
	owners := make(map[string]int, len(rel.Sides)+1)
	for i, s := range rel.Sides {
		owners[strings.ToLower(s.To.Alias())] = i + 1
	}
	if parent != "" {
		owners[strings.ToLower(parent)] = -1
	}
	path := make([]queryHop, len(rel.Sides))
	left := parent
	for i, s := range rel.Sides {
		right := s.To.Alias()
		// Preserve the relationship alias for correlated self-references.
		if owners[strings.ToLower(right)] != i+1 || (parent != "" && right == rel.Sides[0].From.Alias()) {
			right = rel.Name
			if i > 0 {
				right += strconv.Itoa(i)
			}
			for owners[strings.ToLower(right)] != 0 {
				right += "_"
			}
			owners[strings.ToLower(right)] = i + 1
		}
		path[i].Table = clause.TableRef{Expression: s.To.NameExpr(), Alias: right}
		if i > 0 || parent != "" {
			path[i].On = s.Predicates(left, right)
		}
		left = right
	}
	return path
}

// Correlate builds the FROM, WHERE and JOIN clauses shared by correlated
// EXISTS and COUNT subqueries. The dialect supplies its FROM mod.
func Correlate[Q interface {
	AppendJoin(clause.Join)
	AppendWhere(...any)
}, E bob.Expression](rel orm.PreloadRel[E], parent string, from func(Q, bob.Expression)) bob.Mod[Q] {
	if parent == "" && len(rel.Sides) > 0 {
		parent = rel.Sides[0].From.Alias()
	}
	return bob.ModFunc[Q](func(q Q) {
		for i, hop := range queryPath(rel, parent) {
			if i == 0 {
				from(q, hop.Table)
				for _, predicate := range hop.On {
					q.AppendWhere(predicate)
				}
			} else {
				q.AppendJoin(clause.Join{Type: clause.InnerJoin, To: hop.Table, On: hop.On})
			}
		}
	})
}
