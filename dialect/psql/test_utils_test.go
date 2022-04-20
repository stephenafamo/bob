package psql

import (
	"regexp"
	"strings"

	"github.com/google/go-cmp/cmp"
)

var oneOrMoreSpace = regexp.MustCompile(`\s+`)
var spaceAroundBrackets = regexp.MustCompile(`\s*([\(|\)])\s*`)

func clean(s string) string {
	s = strings.TrimSpace(s)
	s = oneOrMoreSpace.ReplaceAllLiteralString(s, " ")
	s = spaceAroundBrackets.ReplaceAllString(s, " $1 ")
	return s
}

func queryDiff(a, b string) string {
	return cmp.Diff(clean(a), clean(b))
}

func argsDiff(a, b []any) string {
	return cmp.Diff(a, b)
}
