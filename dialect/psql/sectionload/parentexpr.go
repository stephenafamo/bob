package sectionload

import (
	"bytes"
	"context"
	"io"
	"regexp"
	"strings"

	"github.com/stephenafamo/bob"
)

// carriedKeySuffix returns the "_sl_k<i>" reserved-name fragment below,
// re-expressed as the part that follows the common "_sl_" prefix ("k"),
// derived from CarriedKeyPrefix (parts.go) instead of being duplicated
// as its own independent literal. Were the generator's carried-column
// name (CarriedKeyName, used by 01_edges.go.tpl) and this regexp's
// "k\d+" alternative two independently maintained copies of the same
// fact, changing one without the other would desync the generated
// column name from what this check recognizes.
func carriedKeySuffix() string {
	return strings.TrimPrefix(CarriedKeyPrefix, "_sl_")
}

// k\d+: a multi-hop edge's carried join-key column(s) land in the
// island's fixed result layout as "_sl_k0", "_sl_k1", ... (parts.go's
// Edge.Carried doc). This is a change to the parent query's ACCEPTED
// range, not merely an addition to the runtime's own internal vocabulary
// — the same "fail-closed over silently colliding" tradeoff already
// applied to cnt\d+ (_sl_cnt<set>): a parent query happening to
// select a real column literally named "_sl_k0" (unlikely, but not
// impossible) is now rejected instead of silently colliding with a
// multi-hop section's own carried column of the same name.
var reservedNameRe = regexp.MustCompile(
	`\b_sl_(?:set|ord|src|n\d+|cnt\d+|` + regexp.QuoteMeta(carriedKeySuffix()) + `\d+)\b`,
)

// parentSource wraps the parent query passed to All/Build so it is
// rendered exactly once per composed-query build — the same number of
// times bob.All itself would render it.
//
// Before this type existed, All rendered the parent query twice: once
// (discarded) to check for reserved-name collisions before composing,
// and again when the composed query, which embeds the parent query
// unmodified inside the _sl_src CTE (see compose.go), was itself built.
// Every WriteSQL call on a *dialect.SelectQuery re-runs its
// ContextualMods (dialect/psql/dialect/select.go), so a
// contextual mod that mutates shared state on the query value — such as
// a query-hint mod that rewrites SelectList.Columns[0] in place — would
// see and compound its own earlier, discarded run.
//
// parentSource closes that gap: it renders the parent's SQL into a local
// buffer exactly once, checks that buffer for reserved identifiers, and
// only then copies it into the real writer. The reserved-name check and
// the SQL that actually gets used are now the same render, so Build
// (which previously had no reserved-name check at all) gets the same
// protection as All for free.
type parentSource struct {
	q bob.Query
}

// trustedParentQuery marks a bob.Query as sectionload's OWN internally
// generated query, exempting it from parentSource's reserved-name check
// below. The check exists to catch a CALLER's
// parent query accidentally colliding with sectionload's internal
// namespace (_sl_set, _sl_ord, _sl_src, _sl_n<k>, _sl_cnt<k>, and
// _sl_k<i>). It was never meant to also police sectionload's OWN
// output: hooks.go's runThen builds a multi-hop edge's route-4 "parent
// query" (the new island's root) via buildEdgeSelect/edgeJoinChain, and
// that query legitimately SELECTs its own carried-key column(s) under
// exactly the names the check exists to protect ("_sl_k0", ...) — this
// is not a foreign collision, it is sectionload consistently using its
// own reserved names for its own machinery, the same way _sl_t/_sl_h<i>
// (scoped inside one edge's own subquery) never needed this protection
// in the first place.
//
// Exempting the WHOLE route-4 query, rather than allowlisting specific
// names, is safe for the exact failure mode this check exists to catch:
// that failure mode is a parent query's SELECT list accidentally naming
// a reserved identifier, and an edge's own caller-provided SelectOptions
// mods cannot introduce a new result column at all —
// SelectOptionsStructureError (options.go) already rejects any mod that
// changes the SELECT list, FROM/JOIN, WITH, or combine clause. So the
// only columns a route-4 query's SELECT list can ever carry are the ones
// buildEdgeSelect itself put there, all of which sectionload already
// fully controls and consistently reads back by name.
//
// parentSource keeps rendering the query exactly once either way (this
// type changes nothing about that): a trusted parent can still carry
// caller-provided WHERE/ORDER BY/LIMIT/OFFSET mods with real contextual
// side effects, so the single-render guarantee this type
// exists for still matters here.
type trustedParentQuery struct {
	bob.Query
}

// WriteSQL implements bob.Expression by delegating to the parent's own
// WriteSQL, passing through the dialect and start it was itself given.
// This is deliberate, not incidental: sm.From(parent), the path this
// type replaces, renders parent through bob.Express's Expression case
// (bob's expression.go), which is exactly v.WriteSQL(ctx, w, d,
// start). Matching that call is what makes parentSource a drop-in
// replacement — verified byte-identical against the pre-existing path
// for a flat BaseQuery, a BaseQuery wrapping another bob.Query, and a
// bare psql.RawQuery.
//
// An earlier version of this type called the parent's WriteQuery
// instead and wrapped the result in hardcoded parentheses. For every
// bob.Query built on bob.BaseQuery that divergence happens to cancel
// out — WriteQuery skips the parens BaseQuery.WriteSQL would have added
// (see query.go) — so it was not the parenthesization
// that was broken. What WriteQuery actually discarded was the dialect:
// it always renders using the parent's own stored b.Dialect, while
// WriteSQL propagates whichever dialect the caller passes in — the
// version this type discarded
// via a blank identifier. A parent nested inside a different dialect
// context, or any non-BaseQuery bob.Query whose WriteSQL and WriteQuery
// legitimately differ, would have rendered wrong. WriteSQL is simply the
// method sm.From's own contract calls; there is no case-by-case
// reasoning to redo here.
func (p parentSource) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var buf bytes.Buffer
	args, err := p.q.WriteSQL(ctx, &buf, d, start)
	if err != nil {
		return nil, err
	}

	if _, trusted := p.q.(trustedParentQuery); !trusted {
		if m := reservedNameRe.FindString(buf.String()); m != "" {
			return nil, &ReservedNameError{Name: m}
		}
	}

	w.WriteString(buf.String())

	return args, nil
}
