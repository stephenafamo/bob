package sectionload

// P-17: before this fix, sectionload.All built the parent query twice per
// call (once to check for reserved names, once again when the composed
// query embedding the parent was itself built), while sectionload.Build
// built it once but skipped the reserved-name check entirely. These
// tests are DB-free (per Codex A-17-3) and assert the target behavior:
// All and Build both build the parent exactly once, and both reject
// reserved identifiers.
//
// Baseline recorded 2026-08-08 against the pre-fix code (run.go's
// separate pre-Build + reservedNameRe check before composeIsland, and
// compose.go's `sm.From(parent).As("q")` with the raw parent query):
//   - TestContextualModRunCount_All: n was 2, not 1.
//   - TestReservedName_Build: Build returned err=nil instead of
//     *ReservedNameError.
//   - TestHintMarkerNotDuplicated: n was 2, so a mutating contextual mod
//     (e.g. fvuls's pgHintMod, which rewrites SelectList.Columns[0] in
//     place) would merge its own hint with itself once.
// All three now pass against the fixed code (parentexpr.go's
// parentSource, wired in from compose.go, builds the parent exactly once
// inside composeIsland; run.go's separate pre-Build was removed).
//
// bob's *dialect.SelectQuery.WriteSQL calls RunContextualMods on every
// invocation (bob@v0.49.0 dialect/psql/dialect/select.go), so counting
// contextual-mod runs is equivalent to counting parent WriteQuery/WriteSQL
// calls.

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/dialect/psql/sm"
	"github.com/stephenafamo/scan"
)

// --- fixtures: a minimal, columnless TableParts so All()/Build() can
// compose a query without any generated model. ---

type countingRow struct{}

type countingBuf struct{}

func (countingBuf) TargetsFor(cols []Column) []any          { return nil }
func (countingBuf) BuildFor(modelCols []Column) countingRow { return countingRow{} }

func countingParts() TableParts[countingRow] {
	return TableParts[countingRow]{
		Schema:  "",
		Table:   "counting_rows",
		Columns: nil,
		NewBuf:  func() Buf[countingRow] { return countingBuf{} },
	}
}

// errStopAfterBuild is returned by the fake executor below. It lets All()
// run all the way through building the composed SQL (where the parent's
// contextual mods fire) without needing a real database connection.
var errStopAfterBuild = errors.New("sectionload test: stop after build")

type stubExecutor struct{}

func (stubExecutor) QueryContext(ctx context.Context, query string, args ...any) (scan.Rows, error) {
	return nil, errStopAfterBuild
}

func (stubExecutor) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, errStopAfterBuild
}

// newCountingParentQuery returns a parent query with a ContextualMod that
// increments *n on every WriteSQL, mimicking how fvuls's pgHintMod
// (pkg/db/query.go AppendContextualModFunc) hooks into the same
// mechanism. It also mutates SelectList.Columns[0] the same way
// pgHintMod does, so a second, unwanted run is independently observable
// as a duplicated marker in the built SQL (mirrors issue §6 step 2).
func newCountingParentQuery(n *int) bob.BaseQuery[*dialect.SelectQuery] {
	q := psql.Select(
		sm.Columns(psql.Raw("1 AS id")),
		sm.From(psql.Raw("counting_rows")),
	)
	q.Expression.AppendContextualModFunc(func(ctx context.Context, sq *dialect.SelectQuery) (context.Context, error) {
		*n++
		cols := sq.SelectList.Columns
		if len(cols) == 0 {
			return ctx, nil
		}
		if existing, ok := cols[0].(hintMarker); ok {
			cols[0] = hintMarker{hint: existing.hint + " hint", wrapped: existing.wrapped}
		} else {
			cols[0] = hintMarker{hint: "hint", wrapped: cols[0]}
		}
		return ctx, nil
	})
	return q
}

// hintMarker is a minimal stand-in for fvuls's pgHintWrapper
// (pkg/db/query.go): it wraps the first SELECT column and prepends a
// marker comment, merging with any marker already there.
type hintMarker struct {
	hint    string
	wrapped any
}

func (h hintMarker) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString("/*+ " + h.hint + " */ ")
	return bob.Express(ctx, w, d, start, h.wrapped)
}

// runAll drives sectionload.All with the stub executor above. The stub
// always errors right after the composed SQL is built (see
// errStopAfterBuild), so any OTHER error (reserved-name, layout, etc.)
// means we did not reach that point.
func runAll(parent bob.Query) error {
	_, err := All[countingRow](context.Background(), stubExecutor{}, parent, countingParts())
	return err
}

// TestContextualModRunCount_All: All() must build the parent query
// exactly once (bob.All's own build count). Pre-fix this was 2 (see file
// header).
func TestContextualModRunCount_All(t *testing.T) {
	var n int
	parent := newCountingParentQuery(&n)

	err := runAll(parent)
	if !errors.Is(err, errStopAfterBuild) {
		t.Fatalf("expected to reach the stub executor, got: %v", err)
	}
	if n != 1 {
		t.Fatalf("contextual mod run count for All() = %d, want 1", n)
	}
}

// TestContextualModRunCount_Build: Build() already built the parent
// query once pre-fix (11.1: it had no pre-Build). This pins that down so
// the fix does not regress it.
func TestContextualModRunCount_Build(t *testing.T) {
	var n int
	parent := newCountingParentQuery(&n)

	_, err := Build[countingRow](context.Background(), parent, countingParts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if n != 1 {
		t.Fatalf("contextual mod run count for Build() = %d, want 1", n)
	}
}

// TestReservedName_Build: Build() must reject a parent query that uses a
// reserved identifier, same as All() (A-17-2's added completion
// condition). Pre-fix, Build() had no reserved-name check at all (11.1)
// and this returned err == nil.
func TestReservedName_Build(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Raw(`1 AS "_sl_ord"`)),
		sm.From(psql.Raw("counting_rows")),
	)

	_, err := Build[countingRow](context.Background(), parent, countingParts())
	var target *ReservedNameError
	if !errors.As(err, &target) {
		t.Fatalf("expected ReservedNameError, got: %v", err)
	}
}

// TestReservedName_All: All() already rejects reserved identifiers in the
// parent query (existing behavior, must not regress).
func TestReservedName_All(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Raw(`1 AS "_sl_ord"`)),
		sm.From(psql.Raw("counting_rows")),
	)

	err := runAll(parent)
	var target *ReservedNameError
	if !errors.As(err, &target) {
		t.Fatalf("expected ReservedNameError, got: %v", err)
	}
}

// TestParentArgsOrder: a parent query with a placeholder argument must
// keep that argument's value and position through composition. This did
// not change across the fix (Build never had a pre-Build to begin with),
// but is asserted here so a future regression on placeholder numbering
// through parentSource.WriteSQL's start passthrough is caught.
func TestParentArgsOrder(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Raw("id")),
		sm.From(psql.Raw("counting_rows")),
		sm.Where(psql.Raw("id = ?", 42)),
	)

	built, err := Build[countingRow](context.Background(), parent, countingParts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(built.Args) != 1 || built.Args[0] != 42 {
		t.Fatalf("args = %#v, want [42]", built.Args)
	}
	if !bytes.Contains([]byte(built.SQL), []byte("$1")) {
		t.Fatalf("SQL does not reference $1 placeholder:\n%s", built.SQL)
	}
}

// TestHintMarkerNotDuplicated: mirrors issue §6 step 2 (明示列+pgHintMod)
// without depending on pkg/db. newCountingParentQuery's mod mutates
// SelectList.Columns[0] in place, exactly like fvuls's pgHintMod; running
// it twice on the same query value would merge the marker with itself,
// producing a doubled marker in the composed SQL. Pre-fix this ran twice
// (n == 2, and the built SQL carried "/*+ hint hint */"); post-fix it
// must run at most once, and the marker must appear at most once in the
// SQL that would actually be executed.
//
// This does not by itself prove pg_hint_plan output was ever duplicated
// in production (issue §4/A-17-4: unmeasured, and out of scope to
// measure here) — it only proves the mechanism that WOULD have caused
// duplication is gone. Build (not runAll's stub-executor path) is used
// here specifically to inspect that SQL, per code review finding I-4:
// asserting only the run count leaves the marker's actual text
// unchecked.
func TestHintMarkerNotDuplicated(t *testing.T) {
	var n int
	parent := newCountingParentQuery(&n)

	built, err := Build[countingRow](context.Background(), parent, countingParts())
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if n > 1 {
		t.Fatalf("contextual mod ran %d times; a mutating mod like pgHintMod would merge its own hint with itself", n)
	}
	if got := strings.Count(built.SQL, "/*+ hint"); got != 1 {
		t.Fatalf("hint marker appears %d times in built SQL, want 1:\n%s", got, built.SQL)
	}
	if strings.Contains(built.SQL, "hint hint") {
		t.Fatalf("hint marker merged with itself in built SQL:\n%s", built.SQL)
	}
}

// TestParentSourceStartPassthrough: parentSource.WriteSQL must forward
// the start argument it is given to the wrapped parent, not hardcode its
// own (code review finding I-3 — TestParentArgsOrder only exercises
// start=1, since composeIsland always renders the parent first).
func TestParentSourceStartPassthrough(t *testing.T) {
	parent := psql.Select(
		sm.Columns(psql.Raw("id")),
		sm.From(psql.Raw("counting_rows")),
		sm.Where(psql.Raw("id = ?", 42)),
	)

	var buf bytes.Buffer
	args, err := parentSource{parent}.WriteSQL(context.Background(), &buf, dialect.Dialect, 5)
	if err != nil {
		t.Fatalf("WriteSQL: %v", err)
	}
	if len(args) != 1 || args[0] != 42 {
		t.Fatalf("args = %#v, want [42]", args)
	}
	if !strings.Contains(buf.String(), "$5") {
		t.Fatalf("SQL does not reference $5 (start=5 was not forwarded):\n%s", buf.String())
	}
}

// TestParentSourceRendersIdenticallyToRawParent: parentSource must render
// byte-identical output to the pre-existing sm.From(parent) path it
// replaced, for every shape of bob.Query — not only the flat
// bob.BaseQuery[*dialect.SelectQuery] sectionload actually receives from
// fvuls today (code review finding I-2: nothing previously pinned this).
// Covers a flat Select, a Select nested inside a Query-of-Query
// BaseQuery (exercises bob.BaseQuery.WriteSQL's Query-in-Query branch,
// query.go:161-163), and psql.RawQuery (a non-Select bob.Query).
func TestParentSourceRendersIdenticallyToRawParent(t *testing.T) {
	build := func(t *testing.T, parent any) (string, []any) {
		t.Helper()
		q := psql.Select(
			sm.Columns(psql.Raw("q.*")),
			sm.From(parent).As("q"),
		)
		sqlStr, args, err := bob.Build(context.Background(), q)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return sqlStr, args
	}

	cases := map[string]bob.Query{
		"flat select": psql.Select(
			sm.Columns(psql.Raw("id")),
			sm.From(psql.Raw("counting_rows")),
			sm.Where(psql.Raw("id = ?", 42)),
		),
		"nested query-of-query": bob.BaseQuery[bob.Query]{
			Expression: psql.Select(
				sm.Columns(psql.Raw("id")),
				sm.From(psql.Raw("counting_rows")),
			),
			Dialect: dialect.Dialect,
		},
		"raw query": psql.RawQuery("SELECT id FROM counting_rows WHERE id = ?", 42),
	}

	for name, parent := range cases {
		t.Run(name, func(t *testing.T) {
			wantSQL, wantArgs := build(t, parent)
			gotSQL, gotArgs := build(t, parentSource{parent})

			if gotSQL != wantSQL {
				t.Fatalf("parentSource SQL differs from sm.From(parent):\nraw:    %s\nwrapped: %s", wantSQL, gotSQL)
			}
			if len(gotArgs) != len(wantArgs) {
				t.Fatalf("parentSource args = %#v, want %#v", gotArgs, wantArgs)
			}
			for i := range wantArgs {
				if gotArgs[i] != wantArgs[i] {
					t.Fatalf("parentSource args = %#v, want %#v", gotArgs, wantArgs)
				}
			}
		})
	}
}

// spyQuery counts WriteSQL and WriteQuery calls separately and renders
// distinct, unmistakable text from each. Every case in
// TestParentSourceRendersIdenticallyToRawParent above happens to render
// byte-identically whichever method parentSource calls, because bob's own
// bob.BaseQuery/psql.RawQuery types implement WriteQuery by delegating to
// WriteSQL for a flat query (query.go:124-133) — so that test alone cannot
// tell "delegates to WriteSQL" apart from "delegates to WriteQuery, which
// happens to agree here" (P-17 Important, Codex review 2026-08-09: an
// earlier version of parentSource called WriteQuery instead of WriteSQL,
// and every existing test still passed against it). spyQuery makes the two
// methods disagree on purpose, so calling the wrong one is directly
// observable instead of relying on coincidental agreement.
type spyQuery struct {
	writeSQLCalls   *int
	writeQueryCalls *int
}

var _ bob.Query = spyQuery{}

func (s spyQuery) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	*s.writeSQLCalls++
	w.WriteString("(SELECT ")
	d.WriteArg(w, start)
	w.WriteString(" /* SPY_WRITESQL */)")
	return []any{"SPY_WRITESQL_ARG"}, nil
}

func (s spyQuery) WriteQuery(ctx context.Context, w io.StringWriter, start int) ([]any, error) {
	*s.writeQueryCalls++
	w.WriteString("SELECT 'SPY_WRITEQUERY'")
	return []any{"SPY_WRITEQUERY_ARG"}, nil
}

func (s spyQuery) Type() bob.QueryType { return bob.QueryTypeSelect }

// TestParentSourceCallsWriteSQLNotWriteQuery is the direct regression test
// for P-17's Important finding: parentSource.WriteSQL must delegate to the
// wrapped parent's WriteSQL, never its WriteQuery. Restoring the exact old
// implementation (WriteQuery delegation plus manually added parentheses)
// leaves the existing byte-identity cases passing but makes this test fail.
// Merely swapping WriteSQL for WriteQuery without restoring the parentheses
// also makes the byte-identity test fail, for that separate reason.
func TestParentSourceCallsWriteSQLNotWriteQuery(t *testing.T) {
	buildFrom := func(t *testing.T, parent any) (string, []any) {
		t.Helper()
		q := psql.Select(
			sm.Columns(psql.Raw("q.*")),
			sm.From(parent).As("q"),
		)
		sqlStr, args, err := bob.Build(context.Background(), q)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return sqlStr, args
	}

	var rawWriteSQLCalls, rawWriteQueryCalls int
	rawSpy := spyQuery{
		writeSQLCalls:   &rawWriteSQLCalls,
		writeQueryCalls: &rawWriteQueryCalls,
	}
	wantSQL, wantArgs := buildFrom(t, rawSpy)

	if rawWriteSQLCalls != 1 || rawWriteQueryCalls != 0 {
		t.Fatalf("sm.From(spy) calls: WriteSQL=%d, WriteQuery=%d; want 1, 0",
			rawWriteSQLCalls, rawWriteQueryCalls)
	}

	var wrappedWriteSQLCalls, wrappedWriteQueryCalls int
	wrappedSpy := spyQuery{
		writeSQLCalls:   &wrappedWriteSQLCalls,
		writeQueryCalls: &wrappedWriteQueryCalls,
	}
	gotSQL, gotArgs := buildFrom(t, parentSource{wrappedSpy})

	if wrappedWriteSQLCalls != 1 {
		t.Fatalf("sm.From(parentSource{spy}) WriteSQL calls = %d, want 1", wrappedWriteSQLCalls)
	}
	if wrappedWriteQueryCalls != 0 {
		t.Fatalf("sm.From(parentSource{spy}) WriteQuery calls = %d, want 0", wrappedWriteQueryCalls)
	}
	if gotSQL != wantSQL {
		t.Fatalf("SQL differs:\nraw:     %s\nwrapped: %s", wantSQL, gotSQL)
	}
	if len(gotArgs) != len(wantArgs) {
		t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
	}
	for i := range wantArgs {
		if gotArgs[i] != wantArgs[i] {
			t.Fatalf("args = %#v, want %#v", gotArgs, wantArgs)
		}
	}
	if !strings.Contains(gotSQL, "SPY_WRITESQL") {
		t.Fatalf("SQL does not contain WriteSQL marker:\n%s", gotSQL)
	}
	if strings.Contains(gotSQL, "SPY_WRITEQUERY") {
		t.Fatalf("SQL contains WriteQuery marker:\n%s", gotSQL)
	}
}
