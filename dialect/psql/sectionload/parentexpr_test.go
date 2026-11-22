package sectionload

// sectionload.All and sectionload.Build must each build the parent query
// exactly once and must both reject reserved identifiers. (Building the
// parent twice, or skipping the reserved-name check, would be wrong: a
// contextual mod that mutates the parent in place, such as a hint wrapper
// rewriting SelectList.Columns[0], would merge its own hint with itself.)
// These tests are DB-free.
//
// bob's *dialect.SelectQuery.WriteSQL calls RunContextualMods on every
// invocation, so counting contextual-mod runs is equivalent to counting
// parent WriteQuery/WriteSQL calls.

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
// increments *n on every WriteSQL, mimicking how an application's query hint
// wrapper hooks into the same mechanism via AppendContextualModFunc. It also
// mutates SelectList.Columns[0] the way such a hint wrapper does, so a
// second, unwanted run is independently observable as a duplicated marker in
// the built SQL.
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

// hintMarker is a minimal stand-in for an application's query hint wrapper:
// it wraps the first SELECT column and prepends a
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

// TestHintMarkerNotDuplicated: an explicit column list plus a hint wrapper.
// newCountingParentQuery's mod mutates SelectList.Columns[0] in place, like
// a query hint wrapper; running it twice on the same query value would merge
// the marker with itself, producing a doubled marker in the composed SQL
// ("/*+ hint hint */"). The mod must run at most once, and the marker must
// appear at most once in the SQL that would actually be executed.
//
// Build (not runAll's stub-executor path) is used here specifically to
// inspect that SQL: asserting only the run count would leave the marker's
// actual text unchecked.
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
// own (TestParentArgsOrder only exercises start=1, since composeIsland
// always renders the parent first).
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
// bob.BaseQuery[*dialect.SelectQuery] sectionload normally receives.
// Covers a flat Select, a Select nested inside a Query-of-Query
// BaseQuery (exercises bob.BaseQuery.WriteSQL's Query-in-Query branch),
// and psql.RawQuery (a non-Select bob.Query).
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
// WriteSQL for a flat query — so that test alone cannot tell "delegates to
// WriteSQL" apart from "delegates to WriteQuery, which happens to agree
// here". spyQuery makes the two
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
// for that: parentSource.WriteSQL must delegate to the
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
