package bob

import (
	"fmt"
	"io"
)

func replaceArgumentBindingsWithCheck(buildArgs []any, args ...any) ([]any, error) {
	var nargs []ArgumentBinding
	hasNonBinding := false
	for _, buildArg := range buildArgs {
		if na, ok := buildArg.(ArgumentBinding); ok {
			nargs = append(nargs, na)
		} else {
			hasNonBinding = true
		}
	}
	if len(nargs) == 0 {
		return args, nil
	}
	if hasNonBinding {
		return nil, fmt.Errorf("cannot mix argument bindings with other arguments")
	}
	return replaceArgumentBindings(nargs, args...)
}

type BoundQuery interface {
	Query

	// MustBuild builds the query and panics on error
	// useful for initializing queries that need to be reused
	MustBuild() (string, []any)

	// MustBuildN builds the query and panics on error
	// start numbers the arguments from a different point
	MustBuildN(start int) (string, []any)

	// Convinient function to build query from start
	Build() (string, []any, error)

	// Convinient function to build query from a point
	BuildN(start int) (string, []any, error)
}

func BindQuery(q Query, args any) BoundQuery {
	return &boundQuery{
		q:    q,
		args: args,
	}
}

type boundQuery struct {
	q    Query
	args any
}

func (q boundQuery) WriteQuery(w io.Writer, start int) ([]any, error) {
	buildArgs, err := q.q.WriteQuery(w, start)
	if err != nil {
		return nil, err
	}
	return replaceArgumentBindingsWithCheck(buildArgs, q.args)
}

func (q boundQuery) WriteSQL(w io.Writer, d Dialect, start int) ([]any, error) {
	buildArgs, err := q.q.WriteSQL(w, d, start)
	if err != nil {
		return nil, err
	}
	return replaceArgumentBindingsWithCheck(buildArgs, q.args)
}

// MustBuild builds the query and panics on error
// useful for initializing queries that need to be reused
func (q boundQuery) MustBuild() (string, []any) {
	return MustBuild(q)
}

// MustBuildN builds the query and panics on error
// start numbers the arguments from a different point
func (q boundQuery) MustBuildN(start int) (string, []any) {
	return MustBuildN(q, start)
}

// Convinient function to build query from start
func (q boundQuery) Build() (string, []any, error) {
	return Build(q)
}

// Convinient function to build query from a point
func (q boundQuery) BuildN(start int) (string, []any, error) {
	return BuildN(q, start)
}
