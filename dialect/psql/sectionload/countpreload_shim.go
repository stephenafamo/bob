// This file is a stopgap. Once orm.CountPreloadColumn exists in bob
// itself, delete this file and replace one line in declaredCountPreloads
// with a type-assertion version.

package sectionload

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/stephenafamo/bob"
)

// countPreloadAliasRe matches the tail every aliasedExpr.WriteSQL
// (gen/templates/counts/bob_counts.bob.go.tpl) produces:
// "<expr> AS "<alias>"". dialect.WriteQuoted
// (dialect/psql/dialect/dialect.go) writes a literal `"` + s + `"` with
// NO escaping of embedded quotes, so a plain [^"]* capture between the
// two literal quotes is exact; there is no doubled-quote escaping to
// undo. The pattern is anchored at the end of the string ($), and this
// function calls WriteSQL on exactly ONE PreloadColumns element at a
// time (never the whole built SQL), so a UNION ALL or wildcard elsewhere
// in a large SQL string cannot confuse a substring search here.
var countPreloadAliasRe = regexp.MustCompile(` AS "([^"]*)"$`)

// declaredCountPreloadsShim renders each SelectList.PreloadColumns
// element on its own and reads back the "__count_<name>" alias
// aliasedExpr.WriteSQL produces, without any typed API from bob.
//
// What this function renders with WriteSQL is ONE PreloadColumns
// element, which is the correlated subquery itself:
// psql.Group(psql.Select(subqueryMods...).Expression) (see the
// PreloadCount method in gen/templates/counts/table/115_counts.go.tpl).
// SelectQuery.WriteSQL runs the contextual mods of the query being
// rendered, and what is rendered here is the correlated subquery, not
// the parent query, so any contextual mods that run are the
// subquery's own (none by default), never the parent query's. The
// concern that the parent's contextual mods would run twice therefore
// does not apply: rendering this element is independent of the parent
// query's contextual mods.
//
// args is discarded. If the caller's mods passed to countExpr
// (PreloadCount.Post.Comments(mods...)) contain psql.Arg(...), the
// rendered result can be a string with placeholders such as $1 plus
// args. Discarding args is safe because the rendered result is not used
// as actual SQL; only the trailing AS "<alias>" header is read, and the
// args (the actual values of the placeholders) play no part in the alias
// extraction.
func declaredCountPreloadsShim(ctx context.Context, cols []any, d bob.Dialect) ([]declaredCountPreload, error) {
	var out []declaredCountPreload
	for i, c := range cols {
		e, ok := c.(bob.Expression)
		if !ok {
			continue
		}
		var sb strings.Builder
		if _, err := e.WriteSQL(ctx, &sb, d, 1); err != nil {
			return nil, fmt.Errorf("sectionload: rendering preload column %d: %w", i, err)
		}
		m := countPreloadAliasRe.FindStringSubmatch(sb.String())
		if m == nil {
			continue // some other PreloadColumns entry (e.g. a JOIN Preload's own column) — not a count preload
		}
		const prefix = "__count_"
		alias := m[1]
		if !strings.HasPrefix(alias, prefix) {
			continue
		}
		out = append(out, declaredCountPreload{
			RelName: strings.TrimPrefix(alias, prefix),
			Alias:   alias,
		})
	}
	return out, nil
}
