package parser

import (
	"regexp"
	"strings"
	"sync/atomic"
	"testing"

	pgparse "github.com/wasilibs/go-pgquery"
)

var paramRe = regexp.MustCompile(`\$\d+`)

func TestDuplicateParamRef(t *testing.T) {
	input := "SELECT id FROM users WHERE id = $1 OR parent_id = $1"

	scanResult, err := pgparse.Scan(input)
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	parseResult, err := pgparse.Parse(input)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if len(parseResult.Stmts) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(parseResult.Stmts))
	}

	w := walker{
		input:       input,
		tokens:      scanResult.GetTokens(),
		names:       make(map[position]string),
		nullability: make(map[position]nullable),
		groups:      make(map[argPos]struct{}),
		multiple:    make(map[[2]int]struct{}),
		atom:        &atomic.Int64{},
		paramIdxMap: make(map[int64]int64),
	}

	stmt := parseResult.Stmts[0]
	w.walk(stmt.Stmt)

	formatted, err := w.formattedQuery()
	if err != nil {
		t.Fatalf("formattedQuery: %v", err)
	}

	t.Logf("formatted: %s", formatted)
	t.Logf("len(w.args): %d", len(w.args))

	// Collect unique $N params
	uniqueParams := make(map[string]int)
	for _, m := range paramRe.FindAllString(formatted, -1) {
		uniqueParams[m]++
	}

	t.Logf("unique $N in formatted: %d", len(uniqueParams))
	for k, v := range uniqueParams {
		t.Logf("  %s: %d occurrences", k, v)
	}

	if len(w.args) != len(uniqueParams) {
		t.Errorf("walker has %d unique args but formatted query has %d unique $N params",
			len(w.args), len(uniqueParams))
	}

	// The original $1 should appear only as $1 in the formatted query
	if strings.Count(formatted, "$1") != 2 {
		t.Errorf("expected 2 occurrences of $1 in formatted query, got %d", strings.Count(formatted, "$1"))
	}
}
