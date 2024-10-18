package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

type Testcases map[string]Testcase

// Also used to generate documentation
type Testcase struct {
	Query        bob.Query
	ExpectedSQL  string
	ExpectedArgs []any
	Doc          string
}

var (
	oneOrMoreSpace      = regexp.MustCompile(`\s+`)
	spaceAroundBrackets = regexp.MustCompile(`\s*([\(|\)])\s*`)
)

func Clean(s string) string {
	s = strings.TrimSpace(s)
	s = oneOrMoreSpace.ReplaceAllLiteralString(s, " ")
	s = spaceAroundBrackets.ReplaceAllString(s, " $1 ")
	return s
}

type FormatFunc = func(string) (string, error)

func QueryDiff(a, b string, clean FormatFunc) (string, error) {
	if clean == nil {
		clean = func(s string) (string, error) { return Clean(s), nil }
	}

	cleanA, err := clean(a)
	if err != nil {
		return "", fmt.Errorf("%s\n%w", a, err)
	}

	cleanB, err := clean(b)
	if err != nil {
		return "", fmt.Errorf("%s\n%w", b, err)
	}

	return cmp.Diff(cleanA, cleanB), nil
}

func ArgsDiff(a, b []any) string {
	return cmp.Diff(a, b)
}

func ErrDiff(a, b error) string {
	return cmp.Diff(a, b)
}

func RunTests(t *testing.T, cases Testcases, format FormatFunc) {
	t.Helper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			sql, args, err := bob.Build(context.Background(), tc.Query)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			diff, err := QueryDiff(tc.ExpectedSQL, sql, format)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if diff != "" {
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
	Expression    bob.Expression
	ExpectedSQL   string
	ExpectedArgs  []any
	ExpectedError error
	Doc           string
}

func RunExpressionTests(t *testing.T, d bob.Dialect, cases ExpressionTestcases) {
	t.Helper()
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			b := &strings.Builder{}
			args, err := bob.Express(context.Background(), b, d, 1, tc.Expression)
			sql := b.String()

			if diff := ErrDiff(tc.ExpectedError, err); diff != "" {
				t.Fatalf("diff: %s", diff)
			}
			if diff, _ := QueryDiff(tc.ExpectedSQL, sql, nil); diff != "" {
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

type NoopExecutor struct{}

func (n NoopExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return nil, nil //nolint:nilnil
}

func (n NoopExecutor) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, nil //nolint:nilnil
}
