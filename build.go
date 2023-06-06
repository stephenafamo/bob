package bob

import "bytes"

// MustBuild builds a query and panics on error
// useful for initializing queries that need to be reused
func MustBuild(q QueryWriter) (string, []any) {
	return MustBuildN(q, 1)
}

func MustBuildN(q QueryWriter, start int) (string, []any) {
	sql, args, err := BuildN(q, start)
	if err != nil {
		panic(err)
	}

	return sql, args
}

// Convinient function to build query from start
func Build(q QueryWriter) (string, []any, error) {
	return BuildN(q, 1)
}

// Convinient function to build query from a point
func BuildN(q QueryWriter, start int) (string, []any, error) {
	b := &bytes.Buffer{}
	args, err := q.WriteQuery(b, start)

	return b.String(), args, err
}
