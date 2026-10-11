package templates_test

import (
	"fmt"
	"strings"
	"testing"
	texttemplate "text/template"
	"text/template/parse"

	"github.com/stephenafamo/bob/gen/bobgen-psql/templates/sectionload"
)

// TestEdgesTemplateToOneBranchGuard checks that the to-one branch of
// 01_edges.go.tpl itself (the Edge.IsToMany==false side) still keeps the
// typed-nil Preload contract.
//
// A check on a generated package can only see the result generated from
// the current template. If the template itself were reverted to a form
// without this contract but not yet regenerated (the generated output
// stays old), a check on the generated side would not notice. This check
// reads the template's source text directly, so it detects a rollback of
// the template itself without going through regeneration.
//
// There is no render harness, so a template mixed with text/template
// syntax (`{{...}}`) cannot be parsed with go/parser (unlike an AST check
// on the generated package, this falls back to a raw-string check).
//
// An earlier version matched with strings.Contains, which could not
// detect a broken template that left the expected strings (`var
// related`, `if len(children) > 0 {`, ...) in comments while rewriting
// the real code into an early return (`if len(children) < 1 { return nil
// }`), because a substring match also matches strings inside comments.
// The check now requires an exact match against the whole block
// (wantToOneBranch compared after TrimSpace), and the validation logic is
// separated into validateToOneBranch.
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
			// Always call Preload, even with a typed nil, so that
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
// Unlike a strings.Contains check, this rejects any
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
// so all three validateToOneBranch behavior cases are visible as one
// group.
func TestValidateToOneBranchAcceptsCurrentTemplate(t *testing.T) {
	src, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read table/01_edges.go.tpl: %v", err)
	}
	if err := validateToOneBranch(string(src)); err != nil {
		t.Fatalf("current template must pass validateToOneBranch: %v", err)
	}
}

// TestValidateToOneBranchRejectsCommentOnlyDisguise reproduces a broken
// template: the contract text is left behind in a comment, but the real
// code was rewritten to an early-return shape that skips the typed-nil
// Preload. A strings.Contains-based check could not catch this because
// every "expected" substring was still present — just inside a comment.
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

// TestEdgesTemplateMultiHopSupport is a text guard: it pins
// that 01_edges.go.tpl generates a multi-hop (through) relationship
// instead of skipping it. Like TestEdgesTemplateToOneBranchGuard above,
// this exists because there is no render harness for these templates
// (text/template syntax mixed with Go, no funcmap replica available for
// a syntax-only Parse) — the actual generated OUTPUT can only be checked
// by running generation on a fixture, so the correctness of the template
// cannot be claimed from source-text checks alone; this test only pins
// that the SOURCE still says what multi-hop support requires.
//
// (a) the old `ne (len $rel.Sides) 1` skip branch — which generated
// only a comment and no Edge API at all for a multi-hop relationship —
// must be gone.
// (b) the template must emit Sides for every relationship (single-hop
// included, as a length-1 slice — the requirement that the runtime have
// no branch duplication starts at generation time).
// (c) the multi-hop-only fields — Carried and NewCarriedBuf ONLY — must be
// emitted, guarded by the same $isMultiHop condition the Sides-length check
// computes. ParentKeyFromBuf is NOT inside this guard: every edge
// (single-hop included) gets it, and TestP33KeyFromBufTemplatePlacement
// (below) checks that placement specifically, since this test's own
// strings.Contains check cannot tell whether ParentKeyFromBuf sits
// inside or outside $isMultiHop's block.
func TestEdgesTemplateMultiHopSupport(t *testing.T) {
	src, err := templates.FS.ReadFile("table/01_edges.go.tpl")
	if err != nil {
		t.Fatalf("read table/01_edges.go.tpl: %v", err)
	}
	content := string(src)

	if strings.Contains(content, "ne (len $rel.Sides) 1") {
		t.Fatalf("01_edges.go.tpl still contains the old multi-hop skip condition (`ne (len $rel.Sides) 1`) — it must be removed, not just supplemented with Sides support alongside it")
	}
	if strings.Contains(content, "SKIPPED: multi-hop") {
		t.Fatalf("01_edges.go.tpl still contains the old \"SKIPPED: multi-hop\" comment")
	}

	for _, want := range []string{
		// (b): Sides is generated unconditionally (every relationship,
		// including single-hop).
		"Sides: []sectionload.Side{",
		"{{range $s := $rel.Sides -}}",
		// (c): the multi-hop-only fields, guarded by $isMultiHop.
		"$isMultiHop := gt (len $rel.Sides) 1",
		"{{- if $isMultiHop}}",
		"Carried: []sectionload.Column{",
		"NewCarriedBuf: {{$tAlias.UpSingular}}Parts.NewBuf,",
		// ParentKeyFromBuf is generated for every edge, not only
		// multi-hop — presence anywhere in content is
		// necessary but not sufficient; TestP33KeyFromBufTemplatePlacement
		// checks it is NOT inside the $isMultiHop block above.
		"ParentKeyFromBuf: func(b sectionload.Buf[*models.{{$tAlias.UpSingular}}]) []any {",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("01_edges.go.tpl missing expected multi-hop fragment %q", want)
		}
	}

	// ChildKey must be built from Sides[len-1].ToColumns ($lastSide), not
	// the old $side (which for a multi-hop edge would be the FIRST
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

// TestValidateToOneBranchRejectsLenLessThanOneRephrase reproduces a
// second broken variant: the early return is rephrased as
// `len(children) < 1` (semantically the same guard as the original
// `len(children) == 0`, just worded differently) without any disguising
// comment at all. The exact-match check must still reject it because
// the whole block, not just a keyword, must match wantToOneBranch.
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

// p33KeyFromBufMarkers are the two struct-literal fields required on
// every row edge and every count edge. Matched as TextNode substrings.
var p33KeyFromBufMarkers = []string{
	"\n\t\tParentKeyFromBuf:",
	"\n\t\tChildKeyFromBuf:",
}

// p33PlacementWalker walks a parsed template's syntax tree (parse.Node),
// tracking which conditional branches (IfNode/WithNode) currently enclose
// the cursor, and records — for each marker string found in a TextNode —
// any enclosing branch that is not in the caller-supplied allow-list.
// RangeNode is never added to the ancestor stack: looping once per
// relationship is the structural mechanism the "every edge" claim
// depends on, not a condition that could ever legitimately be seen as
// excluding a field.
//
// This syntax-tree-based check exists because a raw-string check of
// only "is it inside the $isMultiHop block" would not notice a new
// conditional other than $isMultiHop (for example wrapping only
// ChildKeyFromBuf in $isMultiHop, or wrapping both fields on the count
// side in $hasWhere), since the occurrence count stays at one under
// either mutation. Walking the tree and stacking up "which conditions
// we are currently inside" detects placement under any new conditional
// branch.
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
		// BranchNode.ElseList holds a nil *parse.ListNode when there is no
		// else clause — n is non-nil as an interface but the pointer inside
		// is nil (a classic Go pitfall), so n == nil cannot detect it. Check
		// the concrete type for nil here.
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
		// The structure itself (once per relationship), not a condition; it
		// is not added to the condition stack.
		w.walk(x.List)
		w.walk(x.ElseList)
	default:
		// ActionNode/CommentNode/BreakNode/ContinueNode etc. do not contain a
		// TextNode, so they are ignored.
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

// validateP33KeyFromBufPlacement checks both templates' syntax trees for
// the placement rule:
//
//   - Row edge (01_edges.go.tpl) ParentKeyFromBuf/ChildKeyFromBuf: being
//     under `if $rels` and under the per-relationship range is allowed.
//     They must not be under any additional conditional branch,
//     `$isMultiHop` included.
//   - The same two fields of a count edge (02_counts.go.tpl): in addition
//     to the above, being under the single-hop branch (`else` of `if ne
//     (len $rel.Sides) 1`) is allowed. They must not be under a more
//     deeply nested conditional branch such as `$hasWhere`.
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

// TestP33KeyFromBufTemplatePlacement is a DB-free test that checks the
// placement with a syntax-tree-based check
// (validateP33KeyFromBufPlacement). A raw-string version would only look
// at whether the fields are inside the `$isMultiHop` block, and could not
// detect a regression through a new conditional other than $isMultiHop
// (for example wrapping only ChildKeyFromBuf in $isMultiHop, or wrapping
// both fields on the count side in $hasWhere), because the occurrence
// count stays at one under either mutation. The negative tests
// (TestP33KeyFromBufPlacementRejectsChildKeyUnderMultiHop and
// TestP33KeyFromBufPlacementRejectsCountKeysUnderHasWhere) confirm that
// these two concrete regressions are detected.
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

// TestP33KeyFromBufPlacementRejectsChildKeyUnderMultiHop is negative
// test 1: it builds, with string manipulation only (the real file is
// not modified), a mutation that wraps the whole ChildKeyFromBuf of
// 01_edges.go.tpl (comment included) in a new `{{- if $isMultiHop}}...
// {{- end}}`, and confirms that validateP33KeyFromBufPlacement detects
// it.
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

	const marker = "\n\t\t// ChildKeyFromBuf:"
	idx := strings.Index(edgeSource, marker)
	if idx < 0 {
		t.Fatal("could not find the start of the ChildKeyFromBuf comment")
	}
	closeMarker := "\n\t\tStitch: func("
	endIdx := strings.Index(edgeSource[idx:], closeMarker)
	if endIdx < 0 {
		t.Fatal("could not find the end of the ChildKeyFromBuf block (just before Stitch)")
	}
	endIdx += idx

	mutated := edgeSource[:idx] +
		"\n\t\t{{- if $isMultiHop}}" +
		edgeSource[idx:endIdx] +
		"\n\t\t{{- end}}" +
		edgeSource[endIdx:]

	if err := validateP33KeyFromBufPlacement(mutated, string(countBytes)); err == nil {
		t.Fatal("did not detect the mutation that wraps ChildKeyFromBuf in $isMultiHop")
	}
}

// TestP33KeyFromBufPlacementRejectsCountKeysUnderHasWhere is negative
// test 2: it builds, with string manipulation only, a mutation that
// wraps both ParentKeyFromBuf and ChildKeyFromBuf of 02_counts.go.tpl in
// a new `{{- if $hasWhere}}...{{- end}}`, and confirms that
// validateP33KeyFromBufPlacement detects it.
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

	const marker = "\n\t\t// ParentKeyFromBuf:"
	idx := strings.Index(countSource, marker)
	if idx < 0 {
		t.Fatal("could not find the start of the ParentKeyFromBuf comment")
	}
	closeMarker := "\n\t\tSetCount: func("
	endIdx := strings.Index(countSource[idx:], closeMarker)
	if endIdx < 0 {
		t.Fatal("could not find the end of the ChildKeyFromBuf block (just before SetCount)")
	}
	endIdx += idx

	mutated := countSource[:idx] +
		"\n\t\t{{- if $hasWhere}}" +
		countSource[idx:endIdx] +
		"\n\t\t{{- end}}" +
		countSource[endIdx:]

	if err := validateP33KeyFromBufPlacement(string(edgeBytes), mutated); err == nil {
		t.Fatal("did not detect the mutation that wraps both count-side fields in $hasWhere")
	}
}

// TestP33TemplatesParse is a DB-free test that checks that
// 01_edges.go.tpl/02_counts.go.tpl can be parsed as text/template. This
// package has no real render harness, but this syntax check is done (a
// raw-string comparison could not detect general syntax errors at all).
// The custom functions (quote/sub/modelsPkg/sectionloadPkg) are filled
// in minimally with a dummy FuncMap, and the actual rendered result is
// not verified (syntax parsing only).
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
