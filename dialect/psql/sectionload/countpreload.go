package sectionload

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
)

// declaredCountPreload is one bob PreloadCount declaration read off the
// caller's parentQuery: the relationship name PreloadCount(name, count)
// matches against, and the SQL column alias ("__count_" + name) the
// correlated subquery is aliased to in the parent's own SELECT list.
type declaredCountPreload struct {
	RelName string
	Alias   string
}

// countPreloadable is satisfied by every generated model with a to-many
// relationship (the PreloadCount method of bob's
// gen/templates/counts/table/115_counts.go.tpl); a failed assertion is an
// error, not a silent skip.
type countPreloadable interface {
	PreloadCount(name string, count int64) error
}

// declaredCountPreloads reads every bob PreloadCount declaration off
// parentQuery's own SELECT list, using the SAME psqlSelectModdable +
// no-op-mod trick installBuildTimeMapperModCheck (run.go) uses to reach
// the concrete *dialect.SelectQuery — clause.SelectList.PreloadColumns is
// a public field, but only reachable through the concrete type, never
// through the opaque bob.Query interface. Capturing the pointer this way
// is read-only (the no-op mod only assigns a local variable) and
// harmless to call a second time before the real bob.Build later in
// All/Build.
//
// lenient: when true, a parentQuery that is neither a psqlSelectModdable
// nor a bob.BaseQuery[expr.Clause] (that is, one whose declarations
// cannot be read) yields no declarations (nil, nil) instead of
// *UncheckableParentQueryError. Build passes true: Build never inspects
// mapper mods by design (see the doc comment in run.go: Build does not
// reject a parent query that carries mapper mods), and rejecting new
// parent query types merely because declarations cannot be read would
// break that contract. All passes false and rejects with
// *UncheckableParentQueryError as before (installBuildTimeMapperModCheck
// already makes the same decision for the same types, so the duplication
// is harmless).
//
// totalPreloadColumns (the second return value) is an additional
// defensive condition obtainable independently of the declarations'
// contents: the total number of the parent query's PreloadColumns,
// including columns that are not count declarations (for example ones
// from a JOIN Preload the caller added). checkMapperMods (run.go)
// compares it with the number of declarations, so it conservatively
// rejects the case where the declaration count and the GetMapperMods()
// count happen to match while PreloadColumns contains undeclared
// columns. It does not prove the origin of a MapperMod by type; it
// remains a proxy condition and a residual risk.
func declaredCountPreloads(ctx context.Context, parentQuery bob.Query, lenient bool) ([]declaredCountPreload, int, error) {
	moddable, ok := parentQuery.(psqlSelectModdable)
	if !ok {
		switch parentQuery.(type) {
		case bob.BaseQuery[expr.Clause], *bob.BaseQuery[expr.Clause]:
			return nil, 0, nil
		default:
			if lenient {
				return nil, 0, nil
			}
			return nil, 0, &UncheckableParentQueryError{Type: fmt.Sprintf("%T", parentQuery)}
		}
	}

	var selectQuery *pdialect.SelectQuery
	moddable.Apply(bob.ModFunc[*pdialect.SelectQuery](
		func(q *pdialect.SelectQuery) { selectQuery = q },
	))
	if selectQuery == nil {
		if lenient {
			return nil, 0, nil
		}
		return nil, 0, &UncheckableParentQueryError{Type: fmt.Sprintf("%T", parentQuery)}
	}

	// Until bob itself provides an API for this (orm.CountPreloadColumn),
	// call the shim below (declaredCountPreloadsShim). Once such an API
	// exists, replace this one line with the type-assertion version.
	declared, err := declaredCountPreloadsShim(ctx, selectQuery.PreloadColumns, pdialect.Dialect)
	if err != nil {
		return nil, 0, err
	}
	return declared, len(selectQuery.PreloadColumns), nil
}
