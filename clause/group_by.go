package clause

import (
	"context"
	"io"
	"slices"

	"github.com/stephenafamo/bob"
)

type GroupBy struct {
	Groups   []any
	Distinct bool
	With     string // ROLLUP | CUBE
}

func (g *GroupBy) SetGroups(groups ...any) {
	g.Groups = copyGroups(groups)
}

func (g *GroupBy) AppendGroup(e any) {
	g.Groups = append(g.Groups, copyGroup(e))
}

// Grouping sets can contain other grouping sets. Copy their lists, but
// keep the column expressions in those lists as read-only inputs.
func copyGroups(groups []any) []any {
	groups = slices.Clone(groups)
	for i, g := range groups {
		groups[i] = copyGroup(g)
	}
	return groups
}

func copyGroup(e any) any {
	switch g := e.(type) {
	case Grouping:
		return g.copy()
	case GroupingSet:
		return g.copy()
	case *Grouping:
		return copyPtr(g, Grouping.copy)
	case *GroupingSet:
		return copyPtr(g, GroupingSet.copy)
	default:
		return e
	}
}

func (g *GroupBy) SetGroupWith(with string) {
	g.With = with
}

func (g *GroupBy) SetGroupByDistinct(distinct bool) {
	g.Distinct = distinct
}

func (g GroupBy) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any

	// don't write anything if there are no groups
	if len(g.Groups) == 0 {
		return args, nil
	}

	w.WriteString("GROUP BY ")
	if g.Distinct {
		w.WriteString("DISTINCT ")
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, g.Groups, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if g.With != "" {
		w.WriteString(" WITH ")
		w.WriteString(g.With)
	}

	return args, nil
}

type GroupingSet struct {
	Groups []any
	Type   string // GROUPING SETS | CUBE | ROLLUP
}

func (g GroupingSet) copy() GroupingSet {
	g.Groups = copyGroups(g.Groups)
	return g
}

func (g GroupingSet) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString(g.Type)
	args, err := bob.ExpressSlice(ctx, w, d, start, g.Groups, " (", ", ", ")")
	if err != nil {
		return nil, err
	}

	return args, nil
}

type Grouping struct {
	Groups []any
}

func (g Grouping) copy() Grouping {
	g.Groups = copyGroups(g.Groups)
	return g
}

func (g Grouping) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, g.Groups, "(", ", ", ")")
}
