package sectionload

import (
	"context"
	"fmt"

	"github.com/stephenafamo/bob"
	pdialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	"github.com/stephenafamo/bob/expr"
)

// declaredCountPreload is one bob PreloadCount declaration read off the
// caller's parentQuery: the relationship name PreloadCount(name, count)
// matches against, and the SQL column alias ("__count_" + name) the
// correlated subquery is aliased to in the parent's own SELECT list.
type declaredCountPreload struct {
	RelName string
	Alias   string
}

// countPreloadable is satisfied by every generated model with a to-many
// relationship (gen/templates/counts/table/115_counts.go.tpl's
// PreloadCount method) — see §6.7 for why a failed assertion is an error,
// not a silent skip.
type countPreloadable interface {
	PreloadCount(name string, count int64) error
}

// declaredCountPreloads reads every bob PreloadCount declaration off
// parentQuery's own SELECT list, using the SAME psqlSelectModdable +
// no-op-mod trick installBuildTimeMapperModCheck (run.go:88-132) uses to
// reach the concrete *dialect.SelectQuery — clause.SelectList.PreloadColumns
// (clause/select.go:14) is a public field, but only reachable through the
// concrete type, never through the opaque bob.Query interface. Capturing
// the pointer this way is read-only (the no-op mod only assigns a local
// variable) and harmless to call a second time before the real bob.Build
// later in All/Build.
//
// lenient (2026-08-19 Opusレビュー I-2、Fable裁定=前者の案): true のとき、
// parentQuery が psqlSelectModdable でも bob.BaseQuery[expr.Clause] でも
// ない(=宣言を読み取れない)場合に *UncheckableParentQueryError を返さず、
// 黙って宣言0件(nil, nil)を返す。Build から呼ぶときは true を渡す —
// Build は MapperMod 検査を一切しない設計(issue P-06 A-06-3、
// run.go:355-360 のdocコメント「Build は mapper mod を持つ親クエリを
// 拒否しない」)のままであり、宣言読み取りの都合で新たに親クエリ型を
// 拒否してしまうと、この既存契約を壊す(この設計書の旧稿はこの区別が
// 無く、Build が不要に厳しくなる回帰をI-2として指摘された)。All から
// 呼ぶときは false を渡し、従来どおり *UncheckableParentQueryError で
// 拒否する([1.6]の`installBuildTimeMapperModCheck`が既に同じ型に対して
// 同じ判定をしているので、二重に見えても実害は無い)。
//
// totalPreloadColumns(第2戻り値、Codexレビュー1回目・防御強化案(a)、
// 2026-08-20)は、宣言の中身とは別に取れる追加の防御条件: 親クエリの
// PreloadColumns自体の総数(count宣言以外のPreloadColumns、例えば呼び出し
// 側が独自に足した別のJOIN Preload由来の列も含む)。checkMapperMods
// (run.go)がこれを「宣言数と一致するか」まで見ることで、「宣言数と
// GetMapperMods()数がたまたま一致するが、PreloadColumns自体に宣言外の
// 列が混ざっている」ケースを保守的に拒否できる。ただし、これは
// MapperMod自体の由来を型的に証明するものではなく、あくまで追加の代理
// 条件であることに変わりはない(§6.5の残存リスク参照)。
func declaredCountPreloads(ctx context.Context, parentQuery bob.Query, lenient bool) ([]declaredCountPreload, int, error) {
	moddable, ok := parentQuery.(psqlSelectModdable)
	if !ok {
		switch parentQuery.(type) {
		case bob.BaseQuery[expr.Clause], *bob.BaseQuery[expr.Clause]:
			return nil, 0, nil
		default:
			if lenient {
				return nil, 0, nil
			}
			return nil, 0, &UncheckableParentQueryError{Type: fmt.Sprintf("%T", parentQuery)}
		}
	}

	var selectQuery *pdialect.SelectQuery
	moddable.Apply(bob.ModFunc[*pdialect.SelectQuery](
		func(q *pdialect.SelectQuery) { selectQuery = q },
	))
	if selectQuery == nil {
		if lenient {
			return nil, 0, nil
		}
		return nil, 0, &UncheckableParentQueryError{Type: fmt.Sprintf("%T", parentQuery)}
	}

	// bob側API(orm.CountPreloadColumn、§6.2)がマージされるまでは、シム版
	// (declaredCountPreloadsShim、下記)を呼ぶ。マージ後はこの1行を型
	// アサーション版へ差し替える(§7.1手順3)。
	declared, err := declaredCountPreloadsShim(ctx, selectQuery.PreloadColumns, pdialect.Dialect)
	if err != nil {
		return nil, 0, err
	}
	return declared, len(selectQuery.PreloadColumns), nil
}
