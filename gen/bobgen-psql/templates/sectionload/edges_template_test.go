package templates_test

import (
	"fmt"
	"strings"
	"testing"
	texttemplate "text/template"
	"text/template/parse"

	"github.com/stephenafamo/bob/gen/bobgen-psql/templates/sectionload"
)

// TestEdgesTemplateToOneBranchGuard は 01_edges.go.tpl 自体の to-one
// 分岐(Edge.IsToMany==false 側)が P-01 の typed-nil Preload 契約を
// まだ保っているかを検査する。
//
// pkg/dbsections の TestGeneratedToOneStitchUsesTypedNilPreload は
// 「今のテンプレートで実際に生成した結果」しか見られない。もしテンプレート
// 自体をこの契約が無い形へ revert したが、まだ regen していない(生成物は
// 古いまま)場合、生成物側の検査は変化を検出できない。この検査はテンプレート
// のソーステキストを直接見るため、regen を経由せずにテンプレートの巻き戻し
// そのものを検出できる。
//
// render harness が無いため、text/template のシンタックス(`{{...}}`混じり)
// を go/parser で構文解析できない(pkg/dbsections側のAST検査とは
// 異なり、ここは生文字列検査で代替する)。
//
// Codex 4回目レビュー(P-01 round 4 Important 2)指摘: 以前は
// strings.Contains による部分一致で検査していたため、コメントに期待文字列
// (`var related`, `if len(children) > 0 {` 等)を残したまま実コードだけ
// 早期return(`if len(children) < 1 { return nil }`)へ書き換えた壊れた
// テンプレートを検出できなかった(部分一致はコメント中の文字列にもマッチ
// してしまう)。検証をブロック全体との完全一致(wantToOneBranch + TrimSpace
// 比較)へ書き換え、検証処理自体を validateToOneBranch に分離した。
func TestEdgesTemplateToOneBranchGuard(t *testing.T) {
	src, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read table/01_edges.go.tpl: %v", err)
	}
	if err := validateToOneBranch(string(src)); err != nil {
		t.Fatal(err)
	}
}

// wantToOneBranch is the exact text that must appear between the
// `{{- else}}` and `{{- end}}` markers of 01_edges.go.tpl's to-one branch
// (`{{- if $rel.IsToMany}} ... {{- else}} <this> {{- end}}`), verbatim
// (including the template's own indentation — only the shared
// leading/trailing blank lines are trimmed before comparison).
const wantToOneBranch = `
			// P-01: always call Preload, even with a typed nil, so that
			// R.Loaded.{{$relAlias}} becomes true when no child was found
			// (bob Preload parity — nil/true, not nil/false).
			var related *models.{{$fAlias.UpSingular}}
			if len(children) > 0 {
				related = children[0]
			}
			return p.Preload({{quote $relAlias}}, related)
			`

// validateToOneBranch extracts the to-one branch from content (the raw
// 01_edges.go.tpl source, or a synthetic stand-in used by the negative
// tests below) using the same `{{- if $rel.IsToMany}}` / `{{- else}}` /
// `{{- end}}` structural markers as before, and asserts it matches
// wantToOneBranch exactly (strings.TrimSpace on both sides, then ==).
//
// Unlike the pre-round-4 strings.Contains checks, this rejects any
// wording drift — including a broken template that keeps the old
// contract's text in a comment while replacing the real code with an
// early return.
func validateToOneBranch(content string) error {
	const ifMarker = "{{- if $rel.IsToMany}}"
	const elseMarker = "{{- else}}"
	const endMarker = "{{- end}}"

	ifIdx := strings.Index(content, ifMarker)
	if ifIdx < 0 {
		return fmt.Errorf("could not find %q — template structure changed, update this test", ifMarker)
	}
	if strings.Count(content, ifMarker) != 1 {
		return fmt.Errorf("expected exactly one %q marker, found %d — update this test's assumptions",
			ifMarker, strings.Count(content, ifMarker))
	}

	elseIdx := strings.Index(content[ifIdx:], elseMarker)
	if elseIdx < 0 {
		return fmt.Errorf("could not find %q after the IsToMany branch", elseMarker)
	}
	elseIdx += ifIdx + len(elseMarker)

	endIdx := strings.Index(content[elseIdx:], endMarker)
	if endIdx < 0 {
		return fmt.Errorf("could not find %q closing the to-one branch", endMarker)
	}
	endIdx += elseIdx

	toOneBranch := content[elseIdx:endIdx]

	got := strings.TrimSpace(toOneBranch)
	want := strings.TrimSpace(wantToOneBranch)
	if got != want {
		return fmt.Errorf(
			"01_edges.go.tpl to-one branch changed unexpectedly\n--- got ---\n%s\n--- want ---\n%s",
			got, want,
		)
	}
	return nil
}

// TestValidateToOneBranchAcceptsCurrentTemplate pins that the real,
// current 01_edges.go.tpl still passes validateToOneBranch. This is the
// same assertion as TestEdgesTemplateToOneBranchGuard; it is kept as a
// separate, explicitly-named test alongside the two negative tests below
// so all three "validateToOneBranch behavior" cases described in the
// round-4 review are visible as one group.
func TestValidateToOneBranchAcceptsCurrentTemplate(t *testing.T) {
	src, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read table/01_edges.go.tpl: %v", err)
	}
	if err := validateToOneBranch(string(src)); err != nil {
		t.Fatalf("current template must pass validateToOneBranch: %v", err)
	}
}

// TestValidateToOneBranchRejectsCommentOnlyDisguise reproduces the exact
// broken template Codex 4回目レビュー(P-01 round 4 Important 2)pointed
// out: the P-01 contract text is left behind in a comment, but the real
// code was rewritten to the pre-P-01 early-return shape. The old
// strings.Contains-based check could not catch this because every
// "expected" substring was still present — just inside a comment.
func TestValidateToOneBranchRejectsCommentOnlyDisguise(t *testing.T) {
	broken := `
{{- if $rel.IsToMany}}
			return p.Preload({{quote $relAlias}}, models.{{$fAlias.UpSingular}}Slice(children))
			{{- else}}
			// Contract examples:
			// var related *models.{{$fAlias.UpSingular}}
			// if len(children) > 0 {
			// related = children[0]
			// return p.Preload({{quote $relAlias}}, related)
			if len(children) < 1 {
				return nil
			}
			return p.Preload({{quote $relAlias}}, children[0])
			{{- end}}
`
	if err := validateToOneBranch(broken); err == nil {
		t.Fatal("want error for comment-only-disguised early-return template, got nil")
	}
}

// TestValidateToOneBranchRejectsLenLessThanOneRephrase reproduces a
// second broken variant: the early return is rephrased as
// `len(children) < 1` (semantically the same guard as the pre-P-01
// `len(children) == 0`, just worded differently) without any disguising
// comment at all. The exact-match check must still reject it because
// the whole block, not just a keyword, must match wantToOneBranch.
// TestEdgesTemplateMultiHopSupport is P-05 Task 9's text guard: it pins
// that 01_edges.go.tpl generates a multi-hop (through) relationship
// instead of skipping it. Like TestEdgesTemplateToOneBranchGuard above,
// this exists because there is no render harness for these templates
// (text/template syntax mixed with Go, no funcmap replica available for
// a syntax-only Parse) — the actual generated OUTPUT is only checked
// once Task 11's fixture generation runs (§14 stop condition: "テンプレー
// トの正しさ" cannot be claimed from source-text checks alone; this test
// only pins that the SOURCE still says what P-05 requires).
//
// (a) the pre-P-05 `ne (len $rel.Sides) 1` skip branch — which generated
// only a comment and no Edge API at all for a multi-hop relationship —
// must be gone.
// (b) the template must emit Sides for every relationship (single-hop
// included, as a length-1 slice — P-05 §13.6(8)'s "no runtime branch
// duplication" requirement starts at generation time).
// (c) the multi-hop-only fields — Carried and NewCarriedBuf ONLY — must be
// emitted, guarded by the same $isMultiHop condition the Sides-length check
// computes. ParentKeyFromBuf used to be inside this same guard (pre-P-33);
// P-33 moved it outside so every edge (single-hop included) gets it —
// TestP33KeyFromBufTemplatePlacement (below) checks that placement
// specifically, since this test's own strings.Contains check cannot tell
// whether ParentKeyFromBuf sits inside or outside $isMultiHop's block
// (Codexレビュー1回目・指摘2、2026-08-19).
func TestEdgesTemplateMultiHopSupport(t *testing.T) {
	src, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read table/01_edges.go.tpl: %v", err)
	}
	content := string(src)

	if strings.Contains(content, "ne (len $rel.Sides) 1") {
		t.Fatalf("01_edges.go.tpl still contains the pre-P-05 multi-hop skip condition (`ne (len $rel.Sides) 1`) — P-05 must remove it, not just add Sides support alongside it")
	}
	if strings.Contains(content, "SKIPPED: multi-hop") {
		t.Fatalf("01_edges.go.tpl still contains the pre-P-05 \"SKIPPED: multi-hop\" comment")
	}

	for _, want := range []string{
		// (b): Sides is generated unconditionally (every relationship,
		// including single-hop, per P-05 §13.6(8)).
		"Sides: []sectionload.Side{",
		"{{range $s := $rel.Sides -}}",
		// (c): the multi-hop-only fields, guarded by $isMultiHop.
		"$isMultiHop := gt (len $rel.Sides) 1",
		"{{- if $isMultiHop}}",
		"Carried: []sectionload.Column{",
		"NewCarriedBuf: {{$tAlias.UpSingular}}Parts.NewBuf,",
		// ParentKeyFromBuf is generated for every edge, not only
		// multi-hop, since P-33 — presence anywhere in content is
		// necessary but not sufficient; TestP33KeyFromBufTemplatePlacement
		// checks it is NOT inside the $isMultiHop block above.
		"ParentKeyFromBuf: func(b sectionload.Buf[*models.{{$tAlias.UpSingular}}]) []any {",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("01_edges.go.tpl missing expected P-05 fragment %q", want)
		}
	}

	// ChildKey must be built from Sides[len-1].ToColumns ($lastSide), not
	// the pre-P-05 $side (which for a multi-hop edge would be the FIRST
	// intermediate table's ToColumns, not the final table's).
	if !strings.Contains(content, "ChildKey: func(o *models.{{$fAlias.UpSingular}}) []any {") {
		t.Fatalf("01_edges.go.tpl missing the ChildKey function")
	}
	if !strings.Contains(content, "range $col := $lastSide.ToColumns") {
		t.Fatalf("01_edges.go.tpl's ChildKey must range over $lastSide.ToColumns (Sides[len-1]), not $side (Sides[0])")
	}
	if !strings.Contains(content, "range $col := $firstSide.FromColumns") {
		t.Fatalf("01_edges.go.tpl's ParentKey must range over $firstSide.FromColumns (Sides[0])")
	}
}

func TestValidateToOneBranchRejectsLenLessThanOneRephrase(t *testing.T) {
	broken := `
{{- if $rel.IsToMany}}
			return p.Preload({{quote $relAlias}}, models.{{$fAlias.UpSingular}}Slice(children))
			{{- else}}
			if len(children) < 1 {
				return nil
			}
			return p.Preload({{quote $relAlias}}, children[0])
			{{- end}}
`
	if err := validateToOneBranch(broken); err == nil {
		t.Fatal("want error for `len(children) < 1` early-return rephrase, got nil")
	}
}

// p33TemplateFuncs is the minimal dummy FuncMap that lets 01_edges.go.tpl/
// 02_counts.go.tpl parse (they use quote/sub/modelsPkg/sectionloadPkg at
// generation time; only syntax matters here, never the rendered output).
func p33TemplateFuncs() texttemplate.FuncMap {
	return texttemplate.FuncMap{
		"quote":          func(any) string { return "" },
		"sub":            func(a, b int) int { return a - b },
		"modelsPkg":      func() string { return "" },
		"sectionloadPkg": func() string { return "" },
	}
}

// p33KeyFromBufMarkers are the two struct-literal fields P-33 requires on
// every row edge and every count edge (7.4節). Matched as TextNode
// substrings, the same anchor strings the pre-2回目レビュー実装 used.
var p33KeyFromBufMarkers = []string{
	"\n\t\tParentKeyFromBuf:",
	"\n\t\tChildKeyFromBuf:",
}

// p33PlacementWalker walks a parsed template's syntax tree (parse.Node),
// tracking which conditional branches (IfNode/WithNode) currently enclose
// the cursor, and records — for each marker string found in a TextNode —
// any enclosing branch that is not in the caller-supplied allow-list.
// RangeNode is never added to the ancestor stack: looping once per
// relationship is the structural mechanism P-33's "every edge" claim
// depends on, not a condition that could ever legitimately be seen as
// excluding a field (1回目レビューの了解事項)。
//
// この構文木ベースの検査は Codex レビュー2回目・重要指摘
// (2026-08-19)への対応: 生文字列の「$isMultiHop ブロック内に含まれるか」
// という1点だけを見る旧実装は、$isMultiHop 以外の新しい条件分岐
// (例: ChildKeyFromBuf だけを $isMultiHop で個別に囲む、count 側の両方を
// $hasWhere で囲む)には気付けなかった(出現回数はどちらの変異でも1件の
// まま変わらないため)。構文木を辿って「今どの条件の中にいるか」を正しく
// 積み上げることで、どんな新しい条件分岐配下に置かれても検出できる。
type p33PlacementWalker struct {
	stack      []string
	markers    []string
	allowed    map[string]bool
	counts     map[string]int
	violations map[string][]string
}

func (w *p33PlacementWalker) disallowedAncestors() []string {
	var out []string
	for _, a := range w.stack {
		if !w.allowed[a] {
			out = append(out, a)
		}
	}
	return out
}

func (w *p33PlacementWalker) recordIfMarker(text string) {
	for _, m := range w.markers {
		if !strings.Contains(text, m) {
			continue
		}
		w.counts[m]++
		if bad := w.disallowedAncestors(); len(bad) > 0 {
			if w.violations == nil {
				w.violations = map[string][]string{}
			}
			cp := append([]string(nil), bad...)
			w.violations[m] = cp
		}
	}
}

func (w *p33PlacementWalker) walk(n parse.Node) {
	if n == nil {
		return
	}
	switch x := n.(type) {
	case *parse.ListNode:
		// BranchNode.ElseList は else 節が無いとき *parse.ListNode(nil) を
		// 保持したまま渡ってくる — n はインターフェースとしては非nilだが
		// 中身のポインタがnilになる(Goの典型的な落とし穴)ため、n == nil
		// では検出できない。ここで具体型のnilを確認する。
		if x == nil {
			return
		}
		for _, c := range x.Nodes {
			w.walk(c)
		}
	case *parse.TextNode:
		w.recordIfMarker(string(x.Text))
	case *parse.IfNode:
		label := x.Pipe.String()
		w.stack = append(w.stack, label)
		w.walk(x.List)
		w.stack = w.stack[:len(w.stack)-1]

		w.stack = append(w.stack, "!"+label)
		w.walk(x.ElseList)
		w.stack = w.stack[:len(w.stack)-1]
	case *parse.WithNode:
		label := "with " + x.Pipe.String()
		w.stack = append(w.stack, label)
		w.walk(x.List)
		w.stack = w.stack[:len(w.stack)-1]

		w.stack = append(w.stack, "!"+label)
		w.walk(x.ElseList)
		w.stack = w.stack[:len(w.stack)-1]
	case *parse.RangeNode:
		// 構造そのもの(関連ごとに1回)であり、条件一覧には加えない。
		w.walk(x.List)
		w.walk(x.ElseList)
	default:
		// ActionNode/CommentNode/BreakNode/ContinueNode等、TextNodeを
		// 内包しないノードは無視する。
	}
}

// checkP33Placement parses source with p33TemplateFuncs, walks its syntax
// tree, and requires each of p33KeyFromBufMarkers to occur exactly once
// with no disallowed enclosing branch.
func checkP33Placement(name, source string, allowed map[string]bool) error {
	tmpl, err := texttemplate.New(name).Funcs(p33TemplateFuncs()).Parse(source)
	if err != nil {
		return fmt.Errorf("parse %s: %w", name, err)
	}
	w := &p33PlacementWalker{
		markers: p33KeyFromBufMarkers,
		allowed: allowed,
		counts:  make(map[string]int, len(p33KeyFromBufMarkers)),
	}
	w.walk(tmpl.Tree.Root)
	for _, m := range p33KeyFromBufMarkers {
		if w.counts[m] != 1 {
			return fmt.Errorf("%s: marker %q occurs %d times, want exactly 1", name, m, w.counts[m])
		}
	}
	if len(w.violations) > 0 {
		return fmt.Errorf("%s: marker(s) found under disallowed condition(s): %v", name, w.violations)
	}
	return nil
}

// validateP33KeyFromBufPlacement (Codexレビュー2回目・重要指摘、
// 2026-08-19) checks both templates' syntax trees for the placement rule
// 7.4節 requires:
//
//   - 行エッジ(01_edges.go.tpl)の ParentKeyFromBuf/ChildKeyFromBuf:
//     `if $rels` の配下、および関連ごとの range の配下にあることは許可する。
//     `$isMultiHop` を含むいかなる追加の条件分岐配下にもあってはならない。
//   - countエッジ(02_counts.go.tpl)の同2フィールド: 上記に加えて、
//     単一hop分岐(`else` of `if ne (len $rel.Sides) 1`)の配下にあることも
//     許可する。`$hasWhere` 等、それより内側の条件分岐配下にあっては
//     ならない。
func validateP33KeyFromBufPlacement(edgeSource, countSource string) error {
	if err := checkP33Placement("01_edges.go.tpl", edgeSource, map[string]bool{
		"$rels": true,
	}); err != nil {
		return err
	}
	if err := checkP33Placement("02_counts.go.tpl", countSource, map[string]bool{
		"$rels":                  true,
		"!ne (len $rel.Sides) 1": true,
	}); err != nil {
		return err
	}
	return nil
}

// TestP33KeyFromBufTemplatePlacement is Codexレビュー1回目・指摘2
// (2026-08-19)が要求したDB不要テストを、Codexレビュー2回目・重要指摘
// (2026-08-19)により構文木ベースの検査(validateP33KeyFromBufPlacement)
// へ強化したもの。1回目レビュー直後の生文字列版は「$isMultiHopブロックの
// 中に含まれるか」の1点しか見ておらず、$isMultiHop以外の新しい条件分岐
// (例: ChildKeyFromBufだけを個別に$isMultiHopで囲む、count側の両方を
// $hasWhereで囲む)による退行を検出できなかった(出現回数がどちらの変異
// でも1件のまま変わらないため)。否定側テスト
// (TestP33KeyFromBufPlacementRejectsChildKeyUnderMultiHop・
// TestP33KeyFromBufPlacementRejectsCountKeysUnderHasWhere)がこの2つの
// 具体的な退行を検出できることを確認する。
func TestP33KeyFromBufTemplatePlacement(t *testing.T) {
	edgeBytes, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read edge template: %v", err)
	}
	countBytes, err := templates.FS.ReadFile("table/02_counts.go.tpl")
	if err != nil {
		t.Fatalf("read count template: %v", err)
	}
	if err := validateP33KeyFromBufPlacement(string(edgeBytes), string(countBytes)); err != nil {
		t.Fatalf("validateP33KeyFromBufPlacement: %v", err)
	}
}

// TestP33KeyFromBufPlacementRejectsChildKeyUnderMultiHop is Codexレビュー
// 2回目・重要指摘の否定側テスト1: 01_edges.go.tplのChildKeyFromBuf全体
// (コメントごと)を、新しい `{{- if $isMultiHop}}...{{- end}}` で囲む
// 変異を文字列操作だけで作り(実ファイルは変更しない)、
// validateP33KeyFromBufPlacementがこれを検出することを確認する。
func TestP33KeyFromBufPlacementRejectsChildKeyUnderMultiHop(t *testing.T) {
	edgeBytes, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read edge template: %v", err)
	}
	countBytes, err := templates.FS.ReadFile("table/02_counts.go.tpl")
	if err != nil {
		t.Fatalf("read count template: %v", err)
	}
	edgeSource := string(edgeBytes)

	const marker = "\n\t\t// ChildKeyFromBuf (P-33):"
	idx := strings.Index(edgeSource, marker)
	if idx < 0 {
		t.Fatal("ChildKeyFromBufのコメント開始位置が見つからない")
	}
	closeMarker := "\n\t\tStitch: func("
	endIdx := strings.Index(edgeSource[idx:], closeMarker)
	if endIdx < 0 {
		t.Fatal("ChildKeyFromBufブロックの終端(Stitchの手前)が見つからない")
	}
	endIdx += idx

	mutated := edgeSource[:idx] +
		"\n\t\t{{- if $isMultiHop}}" +
		edgeSource[idx:endIdx] +
		"\n\t\t{{- end}}" +
		edgeSource[endIdx:]

	if err := validateP33KeyFromBufPlacement(mutated, string(countBytes)); err == nil {
		t.Fatal("ChildKeyFromBufを$isMultiHopで囲む変異を検出できなかった")
	}
}

// TestP33KeyFromBufPlacementRejectsCountKeysUnderHasWhere is Codexレビュー
// 2回目・重要指摘の否定側テスト2: 02_counts.go.tplのParentKeyFromBuf・
// ChildKeyFromBuf両方を、新しい `{{- if $hasWhere}}...{{- end}}` で囲む
// 変異を文字列操作だけで作り、validateP33KeyFromBufPlacementがこれを
// 検出することを確認する。
func TestP33KeyFromBufPlacementRejectsCountKeysUnderHasWhere(t *testing.T) {
	edgeBytes, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read edge template: %v", err)
	}
	countBytes, err := templates.FS.ReadFile("table/02_counts.go.tpl")
	if err != nil {
		t.Fatalf("read count template: %v", err)
	}
	countSource := string(countBytes)

	const marker = "\n\t\t// ParentKeyFromBuf (P-33):"
	idx := strings.Index(countSource, marker)
	if idx < 0 {
		t.Fatal("ParentKeyFromBufのコメント開始位置が見つからない")
	}
	closeMarker := "\n\t\tSetCount: func("
	endIdx := strings.Index(countSource[idx:], closeMarker)
	if endIdx < 0 {
		t.Fatal("ChildKeyFromBufブロックの終端(SetCountの手前)が見つからない")
	}
	endIdx += idx

	mutated := countSource[:idx] +
		"\n\t\t{{- if $hasWhere}}" +
		countSource[idx:endIdx] +
		"\n\t\t{{- end}}" +
		countSource[endIdx:]

	if err := validateP33KeyFromBufPlacement(string(edgeBytes), mutated); err == nil {
		t.Fatal("count側の両フィールドを$hasWhereで囲む変異を検出できなかった")
	}
}

// TestP33TemplatesParse is Codexレビュー1回目・指摘2(2026-08-19)が要求する
// DB不要テスト: 01_edges.go.tpl/02_counts.go.tplがtext/templateとして
// 構文解析可能であることを確認する。このパッケージには実レンダリング
// 基盤(render harness)は無いが、この構文解析は行う(Codexレビュー2回目・
// 注記1、2026-08-19: 「従来は生文字列比較でしか検出できなかった」は不正確
// —生文字列比較では一般的な構文エラーはそもそも検出できていなかった)。
// カスタム関数(quote/sub/modelsPkg/sectionloadPkg)はダミーのFuncMapで
// 最小限埋め、実際のレンダリング結果までは検証しない(構文解析のみ)。
func TestP33TemplatesParse(t *testing.T) {
	for _, path := range []string{
		"table/01_edges.go.tpl",
		"table/02_counts.go.tpl",
	} {
		source, err := templates.FS.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if _, err := texttemplate.New(path).Funcs(p33TemplateFuncs()).Parse(string(source)); err != nil {
			t.Fatalf("parse %s: %v", path, err)
		}
	}
}
