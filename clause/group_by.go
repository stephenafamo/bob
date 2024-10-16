package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type GroupBy struct {
	Groups   []any
	Distinct bool
	With     string // ROLLUP | CUBE
}

func (g *GroupBy) SetGroups(groups ...any) {
	g.Groups = groups
}

func (g *GroupBy) AppendGroup(e any) {
	g.Groups = append(g.Groups, e)
}

func (g *GroupBy) SetGroupWith(with string) {
	g.With = with
}

func (g *GroupBy) SetGroupByDistinct(distinct bool) {
	g.Distinct = distinct
}

func (g GroupBy) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	// don't write anything if there are no groups
	if len(g.Groups) == 0 {
		return args, nil
	}

	w.Write([]byte("GROUP BY "))
	if g.Distinct {
		w.Write([]byte("DISTINCT "))
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, g.Groups, "", ", ", "")
	if err != nil {
		return nil, err
	}

	if g.With != "" {
		w.Write([]byte(" WITH "))
		w.Write([]byte(g.With))
	}

	return args, nil
}

type GroupingSet struct {
	Groups []bob.Expression
	Type   string // GROUPING SET | CUBE | ROLLUP
}

func (g GroupingSet) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte(g.Type))
	args, err := bob.ExpressSlice(ctx, w, d, start, g.Groups, " (", ", ", ")")
	if err != nil {
		return nil, err
	}

	return args, nil
}
