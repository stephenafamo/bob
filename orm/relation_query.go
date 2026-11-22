package orm

import (
	"strconv"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	"github.com/stephenafamo/bob/expr"
)

// Predicates matches the two sets of keys and applies each side's static
// filters, qualified with the aliases in effect for this use of the path.
func (s PreloadSide[E]) Predicates(from, to string) []bob.Expression {
	on := make([]bob.Expression, 0, len(s.FromColumns)+len(s.FromWhere)+len(s.ToWhere))
	for i, col := range s.FromColumns {
		on = append(on, expr.OP("=", expr.Quote(from, col), expr.Quote(to, s.ToColumns[i])))
	}
	for _, w := range s.FromWhere {
		on = append(on, expr.OP("=", expr.Quote(from, w.Column), expr.Raw(w.SQLValue)))
	}
	for _, w := range s.ToWhere {
		on = append(on, expr.OP("=", expr.Quote(to, w.Column), expr.Raw(w.SQLValue)))
	}
	return on
}

// Correlate builds the FROM, WHERE and JOIN clauses shared by correlated
// EXISTS and COUNT subqueries. The dialect supplies its FROM mod.
func Correlate[Q interface {
	AppendJoin(clause.Join)
	AppendWhere(...any)
}, E bob.Expression](rel PreloadRel[E], parent string, from func(Q, bob.Expression)) bob.Mod[Q] {
	return bob.ModFunc[Q](func(q Q) {
		// Reserve each table's usual alias for its last occurrence, so filters
		// on the final table keep working. The outer reference takes priority.
		owners := make(map[string]int, len(rel.Sides)+1)
		for i, s := range rel.Sides {
			owners[strings.ToLower(s.To.Alias())] = i + 1
		}
		owners[strings.ToLower(parent)] = -1

		left := parent
		for i, s := range rel.Sides {
			right := s.To.Alias()
			// Keep the established relationship alias for self-references,
			// including when the outer query uses a custom alias.
			if owners[strings.ToLower(right)] != i+1 || right == rel.Sides[0].From.Alias() {
				right = rel.Name
				if i > 0 {
					right += strconv.Itoa(i)
				}
				for owners[strings.ToLower(right)] != 0 {
					right += "_"
				}
				owners[strings.ToLower(right)] = i + 1
			}
			ref := clause.TableRef{Expression: s.To.NameExpr(), Alias: right}
			on := s.Predicates(left, right)
			if i == 0 {
				from(q, ref)
				for _, predicate := range on {
					q.AppendWhere(predicate)
				}
			} else {
				q.AppendJoin(clause.Join{Type: clause.InnerJoin, To: ref, On: on})
			}
			left = right
		}
	})
}
