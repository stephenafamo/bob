package dialect

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob/query"
)

type Testcases map[string]Testcase

// Also used to generate documentation
type Testcase struct {
	Query        query.Query
	ExpectedSQL  string
	ExpectedArgs []any
	Doc          string
}

var oneOrMoreSpace = regexp.MustCompile(`\s+`)
var spaceAroundBrackets = regexp.MustCompile(`\s*([\(|\)])\s*`)

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

func ErrDiff(a, b error) string {
	return cmp.Diff(a, b)
}

func RunTests(t *testing.T, cases Testcases) {
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sql, args, err := query.Build(tc.Query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff := QueryDiff(tc.ExpectedSQL, sql); diff != "" {
				fmt.Println(sql)
				fmt.Println(args)
				t.Fatalf("diff: %s", diff)
			}
			if diff := ArgsDiff(tc.ExpectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}

type ExpressionTestcases map[string]ExpressionTestcase

// Also used to generate documentation
type ExpressionTestcase struct {
	Expression    query.Expression
	ExpectedSQL   string
	ExpectedArgs  []any
	ExpectedError error
	Doc           string
}

func RunExpressionTests(t *testing.T, d query.Dialect, cases ExpressionTestcases) {
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := &strings.Builder{}
			args, err := query.Express(b, d, 1, tc.Expression)
			sql := b.String()

			if diff := ErrDiff(tc.ExpectedError, err); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff := QueryDiff(tc.ExpectedSQL, sql); diff != "" {
				fmt.Println(sql)
				fmt.Println(args)
				t.Fatalf("diff: %s", diff)
			}
			if diff := ArgsDiff(tc.ExpectedArgs, args); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
		})
	}
}
