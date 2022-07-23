package dialect

import (
	"regexp"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob/query"
)

type Testcases map[string]Testcase
type Testcase struct {
	Query         query.Query
	ExpectedQuery string
	ExpectedArgs  []any
}

var oneOrMoreSpace = regexp.MustCompile(`\s+`)
var spaceAroundBrackets = regexp.MustCompile(`\s*([\(|\)]+)\s*`)

func Clean(s string) string {
	s = strings.TrimSpace(s)
	s = oneOrMoreSpace.ReplaceAllLiteralString(s, " ")
	s = spaceAroundBrackets.ReplaceAllString(s, " $1 ")
	return s
}

func QueryDiff(a, b string) string {
	return cmp.Diff(Clean(a), Clean(b))
}

func ArgsDiff(a, b []any) string {
	return cmp.Diff(a, b)
}
