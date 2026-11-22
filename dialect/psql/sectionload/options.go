package sectionload

import (
	"context"
	"io"
	"reflect"
	"strings"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
)

// SelectOptions carries per-relationship query mods and column selection:
// query mods go through Mods (WHERE,
// ORDER BY, LIMIT, OFFSET, or any other bob.Mod[*dialect.SelectQuery]);
// model column restriction goes through Only or Except. Only and Except are
// mutually exclusive; calling either with zero column names, or naming an
// unknown column, is rejected when the tree is resolved (sectionload.All/
// Build) — see resolveColumnPlans in columnselect.go. It holds nothing
// else — no internal aliases, no PreloadWhere-style from/to filters —
// because those would expose sectionload's internal alias (_sl_t) as a
// public contract.
//
// buildEdgeSelect never lets the mods (Mods) touch the query it actually builds
// (which already carries sectionload's own join condition): it applies
// them to an isolated, empty query instead, and only the fields extracted
// by extractOptionsFragment — WHERE/ORDER BY/LIMIT/OFFSET/GROUP BY/HAVING —
// ever reach the real query (the old "apply to the real query, then
// check nothing else moved" design let a mod delete the join condition
// itself, since Where was never one of the checked fields).
//
// LIMIT/OFFSET apply to this edge's SELECT as a whole — across every
// matched parent combined, the same "N rows total" meaning bob's own
// ThenLoad gives a slice-relationship LIMIT — never
// "N rows per parent". sm.Limit(5) on a to-many edge loaded for 100
// parents can return as few as 5 children total, not up to 5 each.
type SelectOptions struct {
	mods []bob.Mod[*pdialect.SelectQuery]

	// only/except/onlySet/exceptSet: the caller's column selection
	// for this edge's own SELECT, kept separate from mods (never a
	// bob.Mod) so it can be turned into a ColumnPlan via NewColumnPlan
	// instead of being checked by extractOptionsFragment — see
	// resolveColumnPlans in columnselect.go. onlySet/exceptSet distinguish
	// "never called" (both false, no restriction) from "called with a
	// possibly-empty list" (Only()/Except() with zero args is a caller
	// mistake, rejected as ColumnSelectionEmptyError — a variadic call
	// with no arguments produces a nil slice in Go, indistinguishable
	// from "never called" without a separate flag).
	only      []string
	except    []string
	onlySet   bool
	exceptSet bool
}

// Mods builds a SelectOptions from one or more bob SELECT mods (sm.Where,
// sm.OrderBy, sm.Limit, sm.Offset, or any other bob.Mod[*dialect.SelectQuery]),
// applied to the edge's own SELECT in the given order.
func Mods(mods ...bob.Mod[*pdialect.SelectQuery]) SelectOptions {
	return SelectOptions{mods: mods}
}

// Only restricts this edge's own columns to exactly cols: every other
// column is excluded from the model, except join key columns, which are
// always fetched for stitching (and stay in the model only when named in
// cols explicitly — see NewColumnPlan's doc in parts.go). Mutually
// exclusive with Except; calling both, or calling this with zero cols, is
// rejected when the tree is resolved (sectionload.All/Build), not here —
// see resolveColumnPlans.
func (o SelectOptions) Only(cols ...string) SelectOptions {
	o.only = cols
	o.onlySet = true
	return o
}

// Except restricts this edge's own columns to every column EXCEPT cols.
// A join key column named in cols is rejected (KeyColumnExcludedError):
// sectionload always fetches join key columns for stitching regardless of
// this selection. A join key column NOT named in cols is fetched AND
// kept in the model — matching bob's own psql.PreloadExcept, which never
// special-cases key columns (see NewColumnPlan's doc in parts.go for
// why). Mutually exclusive with Only; calling both, or calling this with
// zero cols, is rejected when the tree is resolved — see
// resolveColumnPlans.
func (o SelectOptions) Except(cols ...string) SelectOptions {
	o.except = cols
	o.exceptSet = true
	return o
}

// Only is sugar for SelectOptions{}.Only(cols...), for callers that want
// only a column restriction and no other WHERE/ORDER BY/LIMIT/OFFSET mods
// (package-level sugar alongside the
// existing package-level Mods).
func Only(cols ...string) SelectOptions {
	return SelectOptions{}.Only(cols...)
}

// Except is sugar for SelectOptions{}.Except(cols...). To combine a
// column restriction with WHERE/ORDER BY/LIMIT/OFFSET mods, start from
// Mods(...) instead and chain .Except(...)/.Only(...) onto it — e.g.
// Mods(sm.Where(...)).Except("col") — since Mods itself is not, and will
// not become, a SelectOptions method (composition is one-directional,
// Mods-first only).
func Except(cols ...string) SelectOptions {
	return SelectOptions{}.Except(cols...)
}

// permittedFragment holds the only parts of a caller's SelectOptions that
// ever reach the query sectionload actually builds: WHERE, ORDER BY,
// LIMIT, OFFSET, GROUP BY, HAVING. extractOptionsFragment produces it by
// applying the caller's mods to an isolated, empty *pdialect.SelectQuery —
// one that never carries sectionload's own join condition — so a mod has
// no way to see, read, or delete that condition, and no field outside
// this fragment can leak through even silently.
type permittedFragment struct {
	where   clause.Where
	orderBy clause.OrderBy
	limit   clause.Limit
	offset  clause.Offset
	groupBy clause.GroupBy
	having  clause.Having
}

// applyTo merges the fragment onto q: WHERE conditions are appended (so
// they combine with q's existing WHERE via AND, same as bob's own
// sm.Where called twice), the rest are set outright since buildEdgeSelect
// never populates them itself.
func (f permittedFragment) applyTo(q *pdialect.SelectQuery) {
	if len(f.where.Conditions) > 0 {
		q.Where.Conditions = append(q.Where.Conditions, f.where.Conditions...)
	}
	if len(f.orderBy.Expressions) > 0 {
		q.OrderBy = f.orderBy
	}
	if f.limit.Count != nil {
		q.Limit = f.limit
	}
	if f.offset.Count != nil {
		q.Offset = f.offset
	}
	if len(f.groupBy.Groups) > 0 {
		q.GroupBy = f.groupBy
	}
	if len(f.having.Conditions) > 0 {
		q.Having = f.having
	}
}

// extractOptionsFragment applies opts to a fresh, empty *pdialect.SelectQuery
// — never the query buildEdgeSelect is actually building — and returns the
// permitted fragment of it. Any field outside WHERE/ORDER BY/LIMIT/OFFSET/
// GROUP BY/HAVING that ends up non-zero means a mod reached outside its
// allowed territory (e.g. sm.Distinct, sm.Window, sm.Fetch, sm.ForUpdate, a
// contextual mod, an embedded hook, a loader, or a combine/CombinedX
// field); that is rejected with a typed *SelectOptionsStructureError before
// this fragment is ever built, let alone applied to a real query.
//
// This isolation is what makes the rejection meaningful: the earlier
// design applied mods straight to the real query (which already had
// sectionload's join condition in its own WHERE) and only compared
// SelectList/TableRef/With/Combines before vs. after — Where was never
// checked, so a mod could delete the join condition outright and pass.
// Because the query here starts empty and is discarded after this call,
// there is nothing for a mod to delete or overwrite except what this
// function then hands back deliberately.
//
// Invariant: pure — the same opts must yield the same fragment. Every
// production call site extracts a given opts value exactly once and
// shares the resulting fragment with whatever else needs to agree with it:
// unionBranch (compose.go, route 2) shares one
// extraction between unionBranchOrdColumn and buildEdgeSelect via
// edgeSelectSpec.fragment; composeIsland's hasKids CTE-body loop (route 1)
// shares one extraction between that CTE's buildEdgeSelect call and route
// 3's `ordered` flag (hasKidsOrdered, passed into unionBranch); runThen
// (hooks.go) shares one extraction between its route-4 buildEdgeSelect
// call, its parentOrdCol decision, and its post-sort check. hasOrderBy
// (below) is no longer called from any production path — only options_test.go
// exercises it directly — so purity is now load-bearing solely for each of
// the three call sites above to see one, and only one, evaluation of a
// stateful opts.mods per section/edge.
func extractOptionsFragment(opts SelectOptions, relName string) (permittedFragment, error) {
	q := &pdialect.SelectQuery{}
	for _, m := range opts.mods {
		m.Apply(q)
	}

	var zero pdialect.SelectQuery
	switch {
	case !reflect.DeepEqual(q.With, zero.With):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "WITH (CTE)"}
	case !reflect.DeepEqual(q.SelectList, zero.SelectList):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "select columns"}
	case !reflect.DeepEqual(q.Distinct, zero.Distinct):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "DISTINCT"}
	case !reflect.DeepEqual(q.TableRef, zero.TableRef):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "FROM/JOIN"}
	case !reflect.DeepEqual(q.Windows, zero.Windows):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "WINDOW"}
	case !reflect.DeepEqual(q.Combines, zero.Combines):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "combine (UNION/INTERSECT/EXCEPT)"}
	case !reflect.DeepEqual(q.Fetch, zero.Fetch):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "FETCH"}
	case !reflect.DeepEqual(q.Locks, zero.Locks):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "locking clause (FOR UPDATE etc.)"}
	case !reflect.DeepEqual(q.Load, zero.Load):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "loaders"}
	case !reflect.DeepEqual(q.EmbeddedHook, zero.EmbeddedHook):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "hooks"}
	case !reflect.DeepEqual(q.ContextualModdable, zero.ContextualModdable):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "contextual mods"}
	case !reflect.DeepEqual(q.CombinedOrder, zero.CombinedOrder),
		!reflect.DeepEqual(q.CombinedLimit, zero.CombinedLimit),
		!reflect.DeepEqual(q.CombinedFetch, zero.CombinedFetch),
		!reflect.DeepEqual(q.CombinedOffset, zero.CombinedOffset):
		return permittedFragment{}, &SelectOptionsStructureError{RelName: relName, Field: "combined ORDER BY/LIMIT/FETCH/OFFSET"}
	}

	return permittedFragment{
		where:   q.Where,
		orderBy: q.OrderBy,
		limit:   q.Limit,
		offset:  q.Offset,
		groupBy: q.GroupBy,
		having:  q.Having,
	}, nil
}

// hasOrderBy reports whether opts would add an ORDER BY. It used to decide
// this, at zero cost to callers that pass no ORDER BY mod, for every
// caller that needed to know whether a section needs its own row-ordinal
// column. No production call site calls it anymore:
// unionBranch's route 3, and runThen's
// parentOrdCol/post-sort decisions, now reuse the SAME extraction their
// caller already performed once for another purpose (see
// extractOptionsFragment's own Invariant doc), instead of calling this
// function for a second, independent extraction of the same opts. Kept
// only because options_test.go still exercises it directly. Any error
// here is deliberately swallowed: a real call site extracting for itself
// would instead surface the actionable error.
func hasOrderBy(opts SelectOptions) bool {
	if len(opts.mods) == 0 {
		return false
	}
	frag, err := extractOptionsFragment(opts, "")
	if err != nil {
		return false
	}
	return len(frag.orderBy.Expressions) > 0
}

// scanSingleQuotedLiteral scans a standard single-quoted SQL string
// literal starting at sql[start] (which must be a '\”), treating a
// doubled ” as an escaped literal quote (not a terminator), and returns
// the index just past the closing quote. An unterminated literal (no
// closing quote found) consumes to the end of sql.
func scanSingleQuotedLiteral(sql string, start int) int {
	i := start + 1
	for i < len(sql) {
		if sql[i] == '\'' {
			if i+1 < len(sql) && sql[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		}
		i++
	}
	return i
}

// scanEscapeStringLiteral scans a PostgreSQL E'...'/e'...' escape-string
// literal starting at sql[start] (which must be 'E' or 'e', with sql[start+1]
// == '\”): inside it, a backslash escapes whatever single byte follows
// (including a '\”), so that byte never terminates the literal, and a
// doubled ” is likewise an escaped literal quote. Returns the index just
// past the closing quote; an unterminated literal consumes to the end of
// sql.
func scanEscapeStringLiteral(sql string, start int) int {
	i := start + 2 // skip the leading E/e and the opening '
	for i < len(sql) {
		switch sql[i] {
		case '\\':
			i += 2
			continue
		case '\'':
			if i+1 < len(sql) && sql[i+1] == '\'' {
				i += 2
				continue
			}
			return i + 1
		default:
			i++
		}
	}
	return i
}

// isDollarTagByte reports whether b may appear inside a dollar-quote tag
// ($tag$...$tag$) after its first byte: letters, digits, and underscore.
func isDollarTagByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z') || (b >= '0' && b <= '9')
}

// isDollarTagStartByte reports whether b may begin a dollar-quote tag
// ($tag$...$tag$): PostgreSQL requires the first byte of a tag to follow
// the same rule as an unquoted identifier's first byte — letters and
// underscore only, no digits. This is what tells a tagged dollar-quote
// apart from a `$1`-style bind placeholder: a
// digit right after '$' can only be a bind parameter, never the start of
// a dollar-quote tag.
func isDollarTagStartByte(b byte) bool {
	return b == '_' || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

// scanDollarQuotedLiteral scans a PostgreSQL dollar-quoted literal
// ($$...$$ or $tag$...$tag$) starting at sql[start] (which must be '$').
// It returns ok=false when sql[start] does not begin a valid
// dollar-quote delimiter (e.g. a `$1` bind placeholder), in which case
// the caller must treat the '$' as an ordinary byte. When ok is true, it
// returns the index just past the matching closing delimiter; an
// unterminated literal (no matching closing delimiter found) consumes to
// the end of sql.
func scanDollarQuotedLiteral(sql string, start int) (int, bool) {
	j := start + 1
	if j < len(sql) && sql[j] != '$' {
		if !isDollarTagStartByte(sql[j]) {
			return start, false
		}
		j++
	}
	for j < len(sql) && sql[j] != '$' && isDollarTagByte(sql[j]) {
		j++
	}
	if j >= len(sql) || sql[j] != '$' {
		return start, false
	}
	delim := sql[start : j+1] // "$$" or "$tag$"
	rest := sql[j+1:]
	idx := strings.Index(rest, delim)
	if idx < 0 {
		return len(sql), true
	}
	return j + 1 + idx + len(delim), true
}

// scanQuotedIdentifier reads a double-quoted SQL identifier starting at
// sql[start] (which must be a '"'), unescaping a doubled "" into a single
// literal '"' byte in the returned name. It returns the unescaped name
// and the index just past the closing quote; ok is false when the
// identifier is unterminated (no closing quote found), in which case name
// and end are meaningless.
func scanQuotedIdentifier(sql string, start int) (string, int, bool) {
	var b strings.Builder
	i := start + 1
	for i < len(sql) {
		if sql[i] == '"' {
			if i+1 < len(sql) && sql[i+1] == '"' {
				b.WriteByte('"')
				i += 2
				continue
			}
			return b.String(), i + 1, true
		}
		b.WriteByte(sql[i])
		i++
	}
	return "", 0, false
}

// rewriteTwoPartQualifiedIdentifier reads a quoted identifier starting at
// sql[pos] (which must be a '"') and, only when it is immediately
// followed by exactly one more quoted identifier segment (a
// `"<name>"."<col>"` reference with no third segment following), rewrites
// it to `"<alias>".<col>` when <name> is in m. Any other shape — a bare
// quoted identifier, or a qualified reference with a third segment
// following (schema.table.col, 3+ parts) — is returned byte-identical to
// the input. It returns the text to emit and the index just past what it
// consumed.
func rewriteTwoPartQualifiedIdentifier(sql string, pos int, m map[string]string) (string, int) {
	name1, end1, ok := scanQuotedIdentifier(sql, pos)
	if !ok {
		return sql[pos:], len(sql)
	}
	if end1 >= len(sql) || sql[end1] != '.' || end1+1 >= len(sql) || sql[end1+1] != '"' {
		// bare quoted identifier: no qualifier shape at all
		return sql[pos:end1], end1
	}
	name2, end2, ok := scanQuotedIdentifier(sql, end1+1)
	if !ok {
		return sql[pos:], len(sql)
	}
	if end2 < len(sql) && sql[end2] == '.' && end2+1 < len(sql) && sql[end2+1] == '"' {
		// a third segment follows: schema.table.col (3+ parts), out of
		// scope — leave this pair untouched and let the caller continue
		// scanning from the '.' before the third segment.
		return sql[pos:end2], end2
	}
	alias, found := m[name1]
	if !found {
		return sql[pos:end2], end2
	}
	return quoteIdent(alias) + "." + quoteIdent(name2), end2
}

// rewriteQualifiers replaces `"<name>".` with `"<alias>".` for every name
// in m, for a qualified reference of exactly two quoted segments
// (`"<name>"."<col>"`). It recognizes and skips over SQL lexical
// structure it must never rewrite inside: single-quoted string literals
// (scanSingleQuotedLiteral, doubled ” as an escaped quote), E'...'/e'...'
// escape-string literals (scanEscapeStringLiteral, backslash escapes the
// next byte), dollar-quoted literals (scanDollarQuotedLiteral, $$...$$ or
// $tag$...$tag$), and doubled "" inside a quoted identifier
// (scanQuotedIdentifier). bob passes values as bind parameters, so a
// literal only appears here when a caller wrote expr.Raw with one in it;
// skipping keeps such a literal byte-identical.
//
// Schema-qualified references (psql.Quote(schema, table, col), 3 args —
// e.g. "public"."user_teams"."user_id") are OUT OF SCOPE and left
// untouched: for a reference of exactly three quoted segments,
// rewriteTwoPartQualifiedIdentifier returns the first pair (segments 1
// and 2) byte-identical, which is guaranteed by the implementation, not
// merely a documented restriction. This guarantee extends only to that first pair: for a chain
// of four or more segments, the caller loop in rewriteQualifiers resumes
// scanning right after the pair it returned, so the next quoted segment
// starts a fresh, independent two-part evaluation — a name in a later
// pair that happens to match an entry in m would still be rewritten.
// Generated code never emits such a chain: every column expression
// uses the 2-arg psql.Quote form, so no schema qualifier appears at
// all. This case is therefore unreachable in practice, and this
// function should be re-verified if generated code ever starts
// emitting a schema qualifier.
func rewriteQualifiers(sql string, m map[string]string) string {
	if len(m) == 0 || !strings.Contains(sql, `".`) {
		return sql
	}
	var out strings.Builder
	out.Grow(len(sql))
	for i := 0; i < len(sql); {
		c := sql[i]
		switch {
		case c == '\'':
			end := scanSingleQuotedLiteral(sql, i)
			out.WriteString(sql[i:end])
			i = end
		case (c == 'E' || c == 'e') && i+1 < len(sql) && sql[i+1] == '\'':
			end := scanEscapeStringLiteral(sql, i)
			out.WriteString(sql[i:end])
			i = end
		case c == '$':
			if end, ok := scanDollarQuotedLiteral(sql, i); ok {
				out.WriteString(sql[i:end])
				i = end
			} else {
				out.WriteByte(c)
				i++
			}
		case c == '"':
			text, end := rewriteTwoPartQualifiedIdentifier(sql, i, m)
			out.WriteString(text)
			i = end
		default:
			out.WriteByte(c)
			i++
		}
	}
	return out.String()
}

// aliasRewrite renders inner and rewrites every qualified reference
// to a table name in m into that table's alias inside this section's own
// SELECT. Rewriting happens at render time, on the SQL text inner itself
// produced, because bob's own column expression type (expr.Quote's
// return, expr/quote.go:23) is unexported: there is no way for this
// package to inspect or rebuild it structurally.
type aliasRewrite struct {
	inner bob.Expression
	m     map[string]string // table name -> alias inside this SELECT
}

func (a aliasRewrite) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var buf strings.Builder
	args, err := a.inner.WriteSQL(ctx, &buf, d, start)
	if err != nil {
		return nil, err
	}
	if _, err := w.WriteString(rewriteQualifiers(buf.String(), a.m)); err != nil {
		return nil, err
	}
	return args, nil
}

// ShouldOmitParens delegates to inner. bob's expression builders wrap any
// expression that does NOT implement bob.ParensOmitter in parentheses, and
// every column reference bob produces implements it and returns true.
// Without this method the wrapper would turn `ORDER BY "id" DESC` into
// `ORDER BY ("id") DESC`, changing the SQL text of every wrapped
// expression even when no rewrite applies.
func (a aliasRewrite) ShouldOmitParens() bool {
	if o, ok := a.inner.(bob.ParensOmitter); ok {
		return o.ShouldOmitParens()
	}
	return false
}

// rewriteQualifiedRefs wraps every expression in the permitted
// fragment so that, when it renders, a column qualified with a table name
// this section reads under an alias is rewritten to that alias. Only the
// fragment's own expressions are wrapped: sectionload's own join
// condition, FROM clause and JOIN chain are built after this and never
// pass through here.
func rewriteQualifiedRefs(f permittedFragment, m map[string]string) permittedFragment {
	if len(m) == 0 {
		return f
	}
	for i, c := range f.where.Conditions {
		if e, ok := c.(bob.Expression); ok {
			f.where.Conditions[i] = aliasRewrite{inner: e, m: m}
		}
	}
	for i, e := range f.orderBy.Expressions {
		f.orderBy.Expressions[i] = aliasRewrite{inner: e, m: m}
	}
	for i, g := range f.groupBy.Groups {
		if e, ok := g.(bob.Expression); ok {
			f.groupBy.Groups[i] = aliasRewrite{inner: e, m: m}
		}
	}
	for i, c := range f.having.Conditions {
		if e, ok := c.(bob.Expression); ok {
			f.having.Conditions[i] = aliasRewrite{inner: e, m: m}
		}
	}
	return f
}
