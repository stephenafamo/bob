// tools/bob-sectionload/sectionload/countpreload_shim.go (暫定。
// orm.CountPreloadColumn がbob本体にマージされ次第、このファイルを削除し
// declaredCountPreloads 内の1行を型アサーション版へ差し替える)
package sectionload

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/stephenafamo/bob"
)

// countPreloadAliasRe matches the tail every aliasedExpr.WriteSQL
// (gen/templates/counts/bob_counts.bob.go.tpl, byte-identical across bob
// v0.49.0/v0.50.0 — P-13-調査メモ.md §0) produces: "<expr> AS "<alias>"".
// dialect.WriteQuoted (dialect/psql/dialect/dialect.go:18-22) writes a
// literal `"` + s + `"` with NO escaping of embedded quotes — confirmed
// by reading the function body, not assumed — so a plain [^"]* capture
// between the two literal quotes is exact; there is no doubled-quote
// escaping to undo. The pattern is anchored at the end of the string
// ($), and this function calls WriteSQL on exactly ONE PreloadColumns
// element at a time (never the whole built SQL), so the P-13 round-5
// characterization failure class (a UNION ALL or wildcard elsewhere in
// a large SQL string confusing a substring search) does not apply here.
var countPreloadAliasRe = regexp.MustCompile(` AS "([^"]*)"$`)

// declaredCountPreloadsShim renders each SelectList.PreloadColumns
// element on its own and reads back the "__count_<name>" alias
// aliasedExpr.WriteSQL produces, without any typed API from bob.
//
// 訂正(2026-08-19、advisorレビューで指摘。2026-08-19 Opusレビュー M-3で
// ファイル名も訂正): この関数がWriteSQLするのは
// 「PreloadColumnsの1要素」であり、その実体は
// psql.Group(psql.Select(subqueryMods...).Expression) という相関サブクエリ
// 自身(gen/templates/counts/table/115_counts.go.tpl:99。旧記述の
// bob_counts.bob.go.tpl:99はc.applyCount(q, "")の行であり誤り)。
// SelectQuery.WriteSQLがcontextual modsを
// 走らせるのは「レンダリングされるクエリ自身が持つcontextual mods」であり、
// ここでレンダリングしているのは親クエリではなく相関サブクエリの方なので、
// 走るとしてもその**サブクエリ自身**のcontextual mods(既定では無い)であって
// 親クエリのcontextual modsではない。したがって「親のcontextual modsが2回
// 走る」という懸念は当たらない — この要素レンダリングは親クエリの
// contextual modsから独立している。
//
// args は使い捨てる(破棄する)。正当化(2026-08-19 Opusレビュー M-7):
// countExpr の呼び出し元 mods(PreloadCount.Task.TaskComments(mods...) の
// caller mods)に psql.Arg(...) が含まれていれば、このレンダリング結果は
// $1 等のプレースホルダと args を伴う文字列になり得る。しかし args を
// 捨ててよいのは、この描画結果を実際のSQLとして使うのではなく、末尾の
// AS "<alias>" というヘッダ部分だけを読むためであり、args(プレース
// ホルダの実引数)はalias抽出に一切関与しない。
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
