package sectionload

import (
	"fmt"

	"github.com/stephenafamo/bob"
)

// Untyped views over the generic contract types. The runtime works on
// these; the generic-to-any adaptation happens here, without reflection.

type bufAny interface {
	targetsFor(cols []Column) []any
	buildFor(modelCols []Column) any
}

type bufAdapter[T any] struct{ b Buf[T] }

func (a bufAdapter[T]) targetsFor(cols []Column) []any  { return a.b.TargetsFor(cols) }
func (a bufAdapter[T]) buildFor(modelCols []Column) any { return a.b.BuildFor(modelCols) }

// rowBufs is the pair of scan buffers one row of a section (or a
// Then(...) boundary's root row) may have: model is always set (the
// child table's own buffer, exactly what childKeyFromBufAny already
// read); carried is set only when the row belongs to a multi-hop edge's
// section (nil for a single-hop edge, or for the root of an island whose
// edge is single-hop or absent). It exists so childKeyFrom (below) can
// pick which buffer to read from without exec.go/hooks.go needing to
// know the single-hop/multi-hop distinction themselves — see edgeSpec's
// childKeyFrom doc for why the branch lives in exactly one place.
type rowBufs struct {
	model   bufAny
	carried bufAny
}

type partsSpec interface {
	schemaName() string
	tableName() string
	// cols is the SQL column list this root actually fetches: always
	// concrete (never nil), so compose.go/exec.go can build SELECT lists
	// and expectedColumns from it directly.
	cols() []Column
	// modelCols is the subset of cols copied into the built model. nil
	// means "no restriction — copy every column", the fast-path sentinel
	// Buf.BuildFor and (when also unrestricted) Buf.TargetsFor expect.
	// partsAdapter always returns nil (the true top-level parent is never
	// column-restricted); edgeRootParts defers to its
	// edge's ColumnPlan, because runThen's child-as-root (hooks.go) is the
	// SAME edge whose ColumnPlan already restricts thenChildQuery's SQL.
	modelCols() []Column
	newBufAny() bufAny
	asSliceAny(items []any) any
	// scanCols/carriedCols/carriedScanCols/newCarriedBufAny exist
	// so execIsland can scan a Then(...) boundary's root the same
	// two-buffer way an ordinary multi-hop section is scanned, when that
	// boundary's own edge happens to be multi-hop (edgeRootParts defers
	// all four to its edge; see node.go's edgeSpec doc for what each
	// returns). partsAdapter (the TRUE top-level root — never itself
	// multi-hop; it is the caller's own parent query) returns cols() for
	// scanCols and nil/nil/nil for the rest — the single-hop shape.
	scanCols() []Column
	carriedCols() []Column
	carriedScanCols() []Column
	newCarriedBufAny() bufAny
}

type partsAdapter[T any] struct{ p TableParts[T] }

func (a partsAdapter[T]) schemaName() string        { return a.p.Schema }
func (a partsAdapter[T]) tableName() string         { return a.p.Table }
func (a partsAdapter[T]) cols() []Column            { return a.p.Columns }
func (a partsAdapter[T]) modelCols() []Column       { return nil }
func (a partsAdapter[T]) newBufAny() bufAny         { return bufAdapter[T]{a.p.NewBuf()} }
func (a partsAdapter[T]) scanCols() []Column        { return a.p.Columns }
func (a partsAdapter[T]) carriedCols() []Column     { return nil }
func (a partsAdapter[T]) carriedScanCols() []Column { return nil }
func (a partsAdapter[T]) newCarriedBufAny() bufAny  { return nil }

func (a partsAdapter[T]) asSliceAny(items []any) any {
	ts := make([]T, len(items))
	for i, it := range items {
		ts[i] = it.(T)
	}
	if a.p.AsSlice == nil {
		return ts
	}
	return a.p.AsSlice(ts)
}

// edgeRootParts adapts an edge's child table as the root of its own
// recursive island: runThen (hooks.go) treats a Then(...) boundary's
// child as a fresh island's "parent" (set 0), but it is still the SAME
// edge whose ColumnPlan already restricts thenChildQuery's actual SQL
// SELECT list. Without this wrapper, composeIsland/execIsland would
// build the outer SELECT and expectedColumns from the child table's full
// TableParts.Columns (via a plain partsAdapter) while the physical query
// only returns the edge's restricted childCols() — a column-layout
// mismatch that appears once an edge is both column-restricted and a
// Then boundary target.
type edgeRootParts struct {
	partsSpec
	edge edgeSpec
}

func (p edgeRootParts) cols() []Column      { return p.edge.childCols() }
func (p edgeRootParts) modelCols() []Column { return p.edge.childModelCols() }

// scanCols/carriedCols/carriedScanCols/newCarriedBufAny all defer to the
// edge, not to the embedded partsSpec (which is e.child(), the child
// TABLE's own unrestricted partsAdapter and knows nothing about this
// edge's ColumnPlan or Carried) — the same reasoning cols()/modelCols()
// above already apply. When e is multi-hop, this is
// what lets execIsland (exec.go) scan a Then(...) boundary's root with
// the same two-buffer (model + carried) shape an ordinary in-island
// multi-hop section uses, so its own rootChildKeyFn can call
// e.childKeyFrom the same way a section row does (hooks.go's runThen).
func (p edgeRootParts) scanCols() []Column        { return p.edge.childScanCols() }
func (p edgeRootParts) carriedCols() []Column     { return p.edge.carriedCols() }
func (p edgeRootParts) carriedScanCols() []Column { return p.edge.carriedScanCols() }
func (p edgeRootParts) newCarriedBufAny() bufAny  { return p.edge.newCarriedBufAny() }

// edgeCore is what BOTH a row edge (Edge) and a count edge (CountEdge)
// provide: the relationship's identity, its join columns, its fixed
// conditions, its caller options, its child table, and buffer-based key
// extraction. buildEdgeSelect/matchWhere/parentKeySubquery need nothing
// beyond this, which is why a count section's SELECT goes through exactly
// the same builder — and therefore the same SelectOptions structure check
// — as every row section.
type edgeCore interface {
	relName() string
	// sides is the normalized hop chain: length 1 for a single-hop edge
	// (whether from a length-1 Sides or the legacy FromCols/ToCols/Where
	// fallback — see Edge.sides()'s doc), length >1 for a through
	// relationship. fromCols/toCols/edgeWhere below are derived from
	// sides()[0] and exist only because so much existing code already
	// calls them by name; new code should call sides() directly whenever
	// it cares about more than the first hop.
	sides() []Side
	fromCols() []string
	toCols() []string
	edgeWhere() []EdgeWhere
	options() SelectOptions
	child() partsSpec
	parentKeyAny(parent any) []any
	// parentKeyFromBufAny/childKeyFromBufAny extract join key values from a
	// scan buffer (bufAny, produced by this edge's own parent/child
	// partsSpec) instead of a built model — see Edge.ParentKeyFromBuf.
	// buf must be the bufAdapter[P]/bufAdapter[C] this edge's own
	// TableParts produces; passing a buffer from an unrelated table panics.
	parentKeyFromBufAny(buf bufAny) []any
	childKeyFromBufAny(buf bufAny) []any
}

// edgeSpec is a row-loading edge: it additionally builds child models and
// stitches them into their parents.
type edgeSpec interface {
	edgeCore
	isToMany() bool
	// childScanCols is every table-native column this edge fetches from
	// SQL for its child section: Edge.ColumnPlan.All() when ColumnPlan
	// restricts columns, else child().cols() (every column of the table).
	// This does NOT include a multi-hop edge's Carried columns — those are
	// a layout-only addition, never part of the child table itself.
	childScanCols() []Column
	// carriedCols is a multi-hop edge's Carried field: the
	// SQL-layout columns (name "_sl_k<i>", DBType from the parent's own
	// join column) that carry the parent's join key alongside a row that
	// has no such column of its own. nil for a single-hop edge.
	carriedCols() []Column
	// carriedScanCols is carriedCols() re-expressed for reading the
	// carried buffer back out: the PARENT's own column names (positionally
	// matching Carried) with Carried's DBTypes — see Edge.Carried's doc
	// for why these are two different name lists over the same values.
	// nil for a single-hop edge.
	carriedScanCols() []Column
	// newCarriedBufAny adapts Edge.NewCarriedBuf (a Buf[P] factory) to the
	// untyped bufAny view. nil for a single-hop edge.
	newCarriedBufAny() bufAny
	// childCols is every result column this edge's child section occupies
	// in the island's fixed layout: childScanCols() followed by
	// carriedCols() (empty for a single-hop edge, so childCols() ==
	// childScanCols() exactly).
	childCols() []Column
	// childModelCols is the subset of childCols copied into the built
	// child model: Edge.ColumnPlan.Model. Empty means "same as
	// childCols(), copy everything" — the BuildFor nil-sentinel.
	childModelCols() []Column
	childKeyAny(child any) []any
	// childKeyFrom is "the join key value THIS row exposes to its own
	// parent", read from whichever of r's two buffers actually holds it.
	// For a single-hop edge this is exactly childKeyFromBufAny(r.model).
	// For a multi-hop edge, the final child table has no column holding
	// the parent's join key at all, so the value instead comes from
	// r.carried, the buffer that scanned this edge's own Carried columns.
	// The single-hop/multi-hop branch lives ONLY inside this one method —
	// every caller (exec.go's section-row/root-row handling, hooks.go's
	// rootChildKeyFn) calls childKeyFrom uniformly and never re-derives
	// which buffer to read.
	childKeyFrom(r rowBufs) []any
	stitchAny(parent any, children []any) error
	// columnPlanRestricts, hasParentKeyFromBuf, hasChildKeyFromBuf back
	// validatePlan (validate.go): whether this edge's ColumnPlan is
	// non-zero, and whether the corresponding *FromBuf overrides are set.
	// validatePlan uses these to require the overrides wherever the
	// *KeyFromBuf fallback below would otherwise run against a
	// partially-scanned buffer.
	columnPlanRestricts() bool
	hasParentKeyFromBuf() bool
	hasChildKeyFromBuf() bool
	// withColumnPlan returns a COPY of this edge with ColumnPlan
	// replaced by cp — never mutates the edge resolveColumnPlans read
	// this from. Edge[P, C] is a value type, so a value-receiver method
	// naturally returns a fresh copy; see columnselect.go's
	// resolveColumnPlans doc for why this matters (a caller may reuse the
	// same Node/Edge value across multiple All()/Build() calls with
	// different SelectOptions each time).
	withColumnPlan(cp ColumnPlan) edgeSpec
}

// countEdgeSpec is a count-only edge. It has no childCols/childModelCols
// (a count section selects join keys and count(*), never a model's
// columns) and no stitchAny (there is nothing to stitch) — which is also
// why CountEdge does not accidentally satisfy edgeSpec.
type countEdgeSpec interface {
	edgeCore
	setCountAny(parent any, count int64) error
	// hasParentKeyFromBuf backs validatePlan (validate.go) the same way
	// edgeSpec's does: a count node nested under a ColumnPlan-restricted
	// row edge reads that edge's partially-scanned parent buffer via
	// parentKeyFromBufAny, whose BuildFor(nil) fallback panics on any
	// excluded NOT NULL column — so the override must be required there.
	hasParentKeyFromBuf() bool
	// carriedCols/carriedScanCols/newCarriedBufAny/declaredKeyCols/
	// childKeyFrom let edgeJoinChain, countKeyCols and execIsland
	// treat a multi-hop count edge the same way they already treat a
	// multi-hop row edge (edgeSpec's identically-named methods). All five
	// are always nil/no-op for a single-hop count (len(sides())==1).
	carriedCols() []Column
	carriedScanCols() []Column
	newCarriedBufAny() bufAny
	// declaredKeyCols is the generator-supplied Sides[0].ToCols/DBType pair
	// (CountEdge.KeyCols) used ONLY for a multi-hop count, since the first
	// intermediate table has no TableParts to introspect at runtime. nil
	// for a single-hop count.
	declaredKeyCols() []Column
	// childKeyFrom reads the grouped key value out of a count row's scan
	// buffer. It follows the same idea as Edge.childKeyFrom, except that
	// it takes a single bufAny (a count row always scans one buffer, so
	// there is no model/carried pair in rowBufs). A single-hop count reads
	// like childKeyFromBufAny(buf). A multi-hop count interprets buf as a
	// bufAdapter[P] and reads it via ParentKeyFromBuf when that is set.
	childKeyFrom(buf bufAny) []any
}

func (e Edge[P, C]) relName() string { return e.RelName }

// sides normalizes Edge.Sides: when the generator (or a hand-written
// test fixture) left Sides empty, synthesize a length-1 Side from the
// legacy flat fields instead. This is the ONLY place either form is read
// directly — every other method and every runtime call site goes
// through sides(), so the single-hop/multi-hop distinction never needs a
// second branch anywhere else.
func (e Edge[P, C]) sides() []Side {
	if len(e.Sides) > 0 {
		return e.Sides
	}
	return []Side{{FromCols: e.FromCols, ToCols: e.ToCols, Where: e.Where}}
}

func (e Edge[P, C]) fromCols() []string     { return e.sides()[0].FromCols }
func (e Edge[P, C]) toCols() []string       { return e.sides()[0].ToCols }
func (e Edge[P, C]) edgeWhere() []EdgeWhere { return e.sides()[0].Where }
func (e Edge[P, C]) isToMany() bool         { return e.ToMany }
func (e Edge[P, C]) options() SelectOptions { return e.Options }
func (e Edge[P, C]) child() partsSpec       { return partsAdapter[C]{e.Child} }

func (e Edge[P, C]) childScanCols() []Column {
	if all := e.ColumnPlan.All(); len(all) > 0 {
		return all
	}
	return e.Child.Columns
}

func (e Edge[P, C]) carriedCols() []Column { return e.Carried }

// carriedScanCols pairs the PARENT's own join-column names (sides()[0]
// .FromCols, positionally) with Carried's DBTypes — see Edge.Carried's
// doc comment for why the SQL-layout name ("_sl_k<i>") and the
// scan-buffer name (the parent's real column name) must differ. Indexing
// sides()[0].FromCols[i] without a bounds check is deliberate: a
// generator/fixture that emits a Carried shorter or longer than
// Sides[0].FromCols is exactly the contract violation validatePlan
// (validate.go) rejects before All() ever reaches here; Build()
// makes no such promise for any of Edge's fields (see run.go's own doc),
// so this is consistent with the rest of this file's un-guarded indexing
// of generator-supplied slices.
func (e Edge[P, C]) carriedScanCols() []Column {
	if len(e.Carried) == 0 {
		return nil
	}
	fromCols := e.sides()[0].FromCols
	out := make([]Column, len(e.Carried))
	for i, c := range e.Carried {
		out[i] = Column{Name: fromCols[i], DBType: c.DBType}
	}
	return out
}

func (e Edge[P, C]) newCarriedBufAny() bufAny {
	if e.NewCarriedBuf == nil {
		return nil
	}
	return bufAdapter[P]{e.NewCarriedBuf()}
}

// childCols is childScanCols() followed by carriedCols(). When carried is
// non-empty (a multi-hop edge), the concatenation is built into a freshly
// allocated slice — never by appending directly onto childScanCols()'s own
// backing array (which may be ColumnPlan.All() or Child.Columns), since
// that array can have spare capacity and appending into it would silently
// corrupt whichever of those two the caller reused it from. A
// single-hop edge has an empty carriedCols(), so this returns
// childScanCols() itself, unmodified, including slice identity.
func (e Edge[P, C]) childCols() []Column {
	scan := e.childScanCols()
	carried := e.carriedCols()
	if len(carried) == 0 {
		return scan
	}
	out := make([]Column, 0, len(scan)+len(carried))
	out = append(out, scan...)
	out = append(out, carried...)
	return out
}

func (e Edge[P, C]) childModelCols() []Column { return e.ColumnPlan.Model }

func (e Edge[P, C]) parentKeyAny(parent any) []any { return e.ParentKey(parent.(P)) }
func (e Edge[P, C]) childKeyAny(child any) []any   { return e.ChildKey(child.(C)) }

// parentKeyFromBufAny/childKeyFromBufAny's fallback (no *FromBuf override)
// always builds with BuildFor(nil) ("every column"), never the edge's own
// childModelCols() (or, for parentKeyFromBufAny, the parent edge's actual
// scanned columns). This is safe ONLY when the buffer was scanned with
// every column of its table — i.e. ColumnPlan does not restrict — because
// a column TargetsFor never scanned leaves its buffer field at the Go
// zero value, and generated Buf.BuildFor(nil) calls .MustGet() on every
// NOT NULL column unconditionally. An unset null.Val's .MustGet()
// panics; it does not "yield zero" the way plain zero-valued fields
// do. This is why validatePlan (validate.go) requires the matching
// *FromBuf override to be set whenever ColumnPlan restricts an edge —
// All (run.go) enforces it once, after the parent query hooks and before
// sectionload builds or executes its own query, rather than leaving the
// fallback to panic mid-scan. (Build does not execute anything, so it
// does not run this check.)
func (e Edge[P, C]) parentKeyFromBufAny(buf bufAny) []any {
	b := buf.(bufAdapter[P]).b
	if e.ParentKeyFromBuf != nil {
		return e.ParentKeyFromBuf(b)
	}
	return e.ParentKey(b.BuildFor(nil))
}

func (e Edge[P, C]) childKeyFromBufAny(buf bufAny) []any {
	b := buf.(bufAdapter[C]).b
	if e.ChildKeyFromBuf != nil {
		return e.ChildKeyFromBuf(b)
	}
	return e.ChildKey(b.BuildFor(nil))
}

// childKeyFrom is the single branch point: single-hop reads r.model
// exactly as childKeyFromBufAny always has; multi-hop reads r.carried
// instead, via ParentKeyFromBuf when the generator set it (the override
// path, extended here to the parent's own carried buffer) or
// ParentKey(b.BuildFor(carriedScanCols())) otherwise.
// BuildFor(carriedScanCols()) — never BuildFor(nil) — matters: nil means
// "every declared field", and the parent model likely has NOT NULL
// fields this edge's own carried scan never touched; BuildFor(nil) would
// call MustGet() on those and panic (the exact failure mode
// parentKeyFromBufAny's own doc comment already documents for the
// unrelated all-columns case).
func (e Edge[P, C]) childKeyFrom(r rowBufs) []any {
	if len(e.sides()) > 1 {
		b := r.carried.(bufAdapter[P]).b
		if e.ParentKeyFromBuf != nil {
			return e.ParentKeyFromBuf(b)
		}
		return e.ParentKey(b.BuildFor(e.carriedScanCols()))
	}
	return e.childKeyFromBufAny(r.model)
}

func (e Edge[P, C]) columnPlanRestricts() bool {
	return len(e.ColumnPlan.Model) > 0 || len(e.ColumnPlan.Internal) > 0
}

func (e Edge[P, C]) hasParentKeyFromBuf() bool { return e.ParentKeyFromBuf != nil }
func (e Edge[P, C]) hasChildKeyFromBuf() bool  { return e.ChildKeyFromBuf != nil }

func (e Edge[P, C]) withColumnPlan(cp ColumnPlan) edgeSpec {
	e.ColumnPlan = cp
	return e
}

func (e Edge[P, C]) stitchAny(parent any, children []any) error {
	cs := make([]C, len(children))
	for i, c := range children {
		cs[i] = c.(C)
	}
	return e.Stitch(parent.(P), cs)
}

// --- CountEdge's untyped view (countEdgeSpec) ------------------------

func (e CountEdge[P, C]) relName() string { return e.RelName }

// sides normalizes CountEdge.Sides the same way Edge.sides() does: when
// the generator (or a hand-written test fixture) left Sides empty,
// synthesize a length-1 Side from the flat FromCols/ToCols/Where fields
// instead.
func (e CountEdge[P, C]) sides() []Side {
	if len(e.Sides) > 0 {
		return e.Sides
	}
	return []Side{{FromCols: e.FromCols, ToCols: e.ToCols, Where: e.Where}}
}

// carriedCols always returns nil: a count section's SQL never adds a
// Carried-style SQL column (the count SELECT stays "key columns +
// count(*)" regardless of hop count). This exists so CountEdge
// satisfies hopChainSpec and edgeJoinChain can be reused verbatim for a
// multi-hop count's JOIN chain.
func (e CountEdge[P, C]) carriedCols() []Column { return nil }

// carriedScanCols (multi-hop only) pairs the PARENT's own join-column
// names (sides()[0].FromCols, positionally) with KeyCols' DBTypes — the
// same two-name-lists-over-the-same-values reasoning as
// Edge.carriedScanCols, because the SQL SELECT/GROUP BY column is the
// first intermediate table's own real column name (Sides[0].ToCols),
// while the scan buffer that reads the value back is the PARENT's own
// Buf[P], addressed by the parent's own field names. Indexing
// fromCols[i] without a bounds check is deliberate, the same way
// Edge.carriedScanCols' unguarded indexing of Sides[0].FromCols[i] is:
// a generator/fixture that emits KeyCols shorter or longer than
// Sides[0].FromCols is exactly the contract violation
// validateMultiHopCountEdge (validate.go) rejects before All() ever
// reaches here — the count-edge counterpart of the row-edge
// Carried-length check validatePlan enforces. Build() makes no such
// promise for any of CountEdge's fields (see run.go's own doc, same as
// Edge), so this is consistent with the rest of this file's un-guarded
// indexing of generator-supplied slices.
func (e CountEdge[P, C]) carriedScanCols() []Column {
	if len(e.KeyCols) == 0 {
		return nil
	}
	fromCols := e.sides()[0].FromCols
	out := make([]Column, len(e.KeyCols))
	for i, c := range e.KeyCols {
		out[i] = Column{Name: fromCols[i], DBType: c.DBType}
	}
	return out
}

// newCarriedBufAny (multi-hop only) adapts CountEdge.NewCarriedBuf
// (a Buf[P] factory) to the untyped bufAny view — the same adaptation
// Edge.newCarriedBufAny already does for row edges. nil for a single-hop
// count.
func (e CountEdge[P, C]) newCarriedBufAny() bufAny {
	if e.NewCarriedBuf == nil {
		return nil
	}
	return bufAdapter[P]{e.NewCarriedBuf()}
}

// declaredKeyCols (multi-hop only) exposes CountEdge.KeyCols, the
// generator-supplied {Name, DBType} pairs for the first intermediate
// table's join key columns. Used by countKeyCols (below) instead of
// runtime introspection against e.child().cols(), because the first
// intermediate table has no TableParts of its own to introspect
// (CountEdge.Child is only the FINAL child table). nil for a single-hop
// count.
func (e CountEdge[P, C]) declaredKeyCols() []Column { return e.KeyCols }

func (e CountEdge[P, C]) fromCols() []string     { return e.sides()[0].FromCols }
func (e CountEdge[P, C]) toCols() []string       { return e.sides()[0].ToCols }
func (e CountEdge[P, C]) edgeWhere() []EdgeWhere { return e.sides()[0].Where }
func (e CountEdge[P, C]) options() SelectOptions { return e.Options }
func (e CountEdge[P, C]) child() partsSpec       { return partsAdapter[C]{e.Child} }

func (e CountEdge[P, C]) parentKeyAny(parent any) []any { return e.ParentKey(parent.(P)) }

func (e CountEdge[P, C]) parentKeyFromBufAny(buf bufAny) []any {
	b := buf.(bufAdapter[P]).b
	if e.ParentKeyFromBuf != nil {
		return e.ParentKeyFromBuf(b)
	}
	return e.ParentKey(b.BuildFor(nil))
}

// childKeyFromBufAny reads the grouped key out of a count row's buffer.
//
// Without ChildKeyFromBuf, the BuildFor(nil) fallback is kept only as a
// compatibility path for hand-written edges. A count row scans nothing
// but ToCols, so with a generated buffer this path can panic when it
// reads a required (NOT NULL) column that was never scanned: BuildFor(nil)
// means "build every column", which is unsafe for a buffer that was
// scanned for key columns only (the same concern as in
// Edge.childKeyFromBufAny). The generated template (02_counts.go.tpl)
// must always set ChildKeyFromBuf. The fallback is only safe with a
// hand-written fake buffer whose BuildFor is a plain value copy and does
// not panic like MustGet.
func (e CountEdge[P, C]) childKeyFromBufAny(buf bufAny) []any {
	b := buf.(bufAdapter[C]).b
	if e.ChildKeyFromBuf != nil {
		return e.ChildKeyFromBuf(b)
	}
	return e.ChildKey(b.BuildFor(nil))
}

// childKeyFrom is CountEdge's counterpart to Edge.childKeyFrom, but
// takes a single bufAny (count always scans exactly one buffer per row —
// there is no rowBufs model/carried pair to choose between the way a row
// section has). Single-hop reads buf via childKeyFromBufAny exactly as
// before. Multi-hop reads buf as the PARENT's own Buf[P] (produced by
// newCarriedBufAny, scanned with carriedScanCols()) via ParentKeyFromBuf
// when the generator set it, or ParentKey(b.BuildFor(carriedScanCols()))
// otherwise — never BuildFor(nil), for the same NOT-NULL-panic reason
// Edge.childKeyFrom's own doc comment explains. ChildKeyFromBuf is
// structurally unreachable here: it takes Buf[C] (the final child table's
// buffer), but the multi-hop group key was never scanned into one.
func (e CountEdge[P, C]) childKeyFrom(buf bufAny) []any {
	if len(e.sides()) > 1 {
		b := buf.(bufAdapter[P]).b
		if e.ParentKeyFromBuf != nil {
			return e.ParentKeyFromBuf(b)
		}
		return e.ParentKey(b.BuildFor(e.carriedScanCols()))
	}
	return e.childKeyFromBufAny(buf)
}

func (e CountEdge[P, C]) setCountAny(parent any, count int64) error {
	return e.SetCount(parent.(P), count)
}

func (e CountEdge[P, C]) hasParentKeyFromBuf() bool { return e.ParentKeyFromBuf != nil }

// ---------------------------------------------------------------------
// Island planning: flatten the declared tree into sections (this island)
// and deferred sub-islands (Then boundaries).

type flatNode struct {
	set       int // _sl_set number (>= 1)
	parentSet int // 0 = the island's parent (src); otherwise a node set
	edge      edgeSpec
	hasKids   bool // true → materialized as its own CTE (_sl_n<set>)
}

// thenPlan is a Then(...) boundary: unlike flatNode, it never becomes a
// section (it is loaded as its own recursive island by runThen, never
// merged into this island's UNION ALL), so it has no _sl_set number of
// its own. id identifies it anyway, drawn from its own negative id space
// (see planIsland) — never touches the `next` counter section/count sets
// share, so a Then boundary can never collide with, or steal a slot from,
// any section's own set number regardless of where it is declared in the
// tree. It exists solely so execIsland can hand back a parentKeys-shaped
// map keyed by something that cannot collide with a section's set number:
// before this field existed, callers indexed parentKeys by parentSet
// itself, which — for any Then nested under a non-root section — collided
// with whatever unrelated section happened to share that same numeric
// set.
type thenPlan struct {
	id        int // unique within this island; never a valid section set
	parentSet int // whose items provide the keys
	edge      edgeSpec
	children  []Node
}

// countDBType is the SQL type of every _sl_cnt<set> column: count(*)
// returns bigint in PostgreSQL.
const countDBType = "bigint"

// cntColName is the result column name a count section's count(*) lands
// in. It is internal, never a public API, and reserved against
// parent-query collisions by reservedNameRe in parentexpr.go.
func cntColName(set int) string {
	return fmt.Sprintf("_sl_cnt%d", set)
}

// countPlan is a count-only section of this island: unlike flatNode it
// builds no models, has no children (a count node is always a leaf) and
// never becomes a CTE. It occupies len(keyCols)+1 columns at the END of
// the island's fixed layout, after every row section.
//
// keyCols/countCol are resolved once here, at plan time, so that every
// later consumer (section0List, checkWidth, expectedColumns, unionBranch,
// countBranch, execIsland) can read them as plain fields instead of
// re-resolving ToCols against the child table and threading an error
// through six call sites.
type countPlan struct {
	set       int // _sl_set number, from the same counter as section sets
	parentSet int // 0 = the island's parent (src); otherwise a node set
	edge      countEdgeSpec
	keyCols   []Column // child-side join key columns (edge.toCols()), with DBTypes
	countCol  Column   // {Name: "_sl_cnt<set>", DBType: "bigint"}
}

// cols is every result column this count section occupies in the island's
// fixed layout, in order.
func (c countPlan) cols() []Column {
	out := make([]Column, 0, len(c.keyCols)+1)
	out = append(out, c.keyCols...)
	out = append(out, c.countCol)
	return out
}

type islandPlan struct {
	sections []flatNode // DFS (declaration) order — deterministic
	counts   []countPlan
	thens    []thenPlan
	// declaredCounts is a root-only extra set of scalar columns: the bob
	// PreloadCount declarations attached to the caller's parentQuery,
	// listed as-is. They never touch TableParts.Columns /
	// partsSpec.cols(), which structurally avoids propagating to every
	// edge that has that table as its Child. With an empty slice every
	// loop below runs zero times, so the column layout of a call that
	// does not use PreloadCount is unchanged.
	declaredCounts []declaredCountPreload
	// When parentOuterOrd is non-nil, _sl_src adds row_number() outside
	// the parent subquery (the parent's LIMIT applies first, and the
	// window function only sees the LIMITed rows, so a top-N heapsort can
	// be used). When nil, the parentOrdCol path
	// (installParentOrdColumn) adds a "_sl_row_ord" column inside the
	// parent SELECT instead.
	parentOuterOrd   []bob.Expression
	parentOuterAlias string
}

// countKeyCols resolves an edge's grouped-by join key column names to
// their {Name, DBType}. For a single-hop count, this is the child table's
// own column list, introspected at runtime. For a multi-hop count, the
// grouped key belongs to the first intermediate table, which has no
// TableParts to introspect at runtime — so the generator-supplied
// declaredKeyCols() (CountEdge.KeyCols) is used instead. Beyond the
// length check (a generator-error safety net, not runtime
// introspection), each KeyCols[i].Name is also checked against
// toCols()[i] positionally: countBranch/buildEdgeSelect (compose.go)
// use KeyCols[i].Name verbatim as the SQL SELECT/GROUP BY column against
// the first intermediate table, so a same-length KeyCols with a wrong
// name or reversed order would still build and run a valid query — just
// one that groups/matches on the WRONG column, silently misattributing
// counts to the wrong parent (or zeroing every parent out) instead of
// failing loudly.
func countKeyCols(e countEdgeSpec) ([]Column, error) {
	if len(e.sides()) > 1 {
		kc := e.declaredKeyCols()
		toCols := e.toCols()

		if len(kc) != len(toCols) {
			return nil, &CountKeyColsMismatchError{
				RelName: e.relName(), Declared: len(kc), ToCols: len(toCols),
			}
		}

		for i, c := range kc {
			if c.Name != toCols[i] {
				return nil, &CountKeyColsMismatchError{
					RelName:  e.relName(),
					Declared: len(kc),
					ToCols:   len(toCols),
					Reason: fmt.Sprintf(
						"KeyCols[%d].Name (%q) must equal Sides[0].ToCols[%d] (%q)",
						i, c.Name, i, toCols[i],
					),
				}
			}
		}

		return kc, nil
	}
	all := e.child().cols()
	byName := make(map[string]Column, len(all))
	for _, c := range all {
		byName[c.Name] = c
	}
	out := make([]Column, 0, len(e.toCols()))
	for _, name := range e.toCols() {
		c, ok := byName[name]
		if !ok {
			return nil, &UnknownKeyColumnError{Column: name}
		}
		out = append(out, c)
	}
	return out, nil
}

func planIsland(nodes []Node) (islandPlan, error) {
	var plan islandPlan
	next := 1
	var walk func(parentSet int, n Node) error
	walk = func(parentSet int, n Node) error {
		if n.count != nil {
			if n.then {
				return &CountNodeThenError{RelName: n.count.relName()}
			}
			set := next
			next++
			keyCols, err := countKeyCols(n.count)
			if err != nil {
				return err
			}
			plan.counts = append(plan.counts, countPlan{
				set:       set,
				parentSet: parentSet,
				edge:      n.count,
				keyCols:   keyCols,
				countCol:  Column{Name: cntColName(set), DBType: countDBType},
			})
			return nil
		}
		if n.then {
			// A from-side fixed condition cannot cross this
			// boundary. Inside one island it narrows the parent-key
			// semi-join subquery (parentKeySubquery in compose.go, which
			// matchWhere reaches only when semiFromCTE is set); a Then
			// boundary instead matches an already-collected list of
			// literal parent-key tuples (matchWhere's keys branch), where
			// no parent CTE exists to constrain. matchWhere would drop the
			// condition silently and stitch children onto parents that do
			// not satisfy it, so reject the tree here — the same place,
			// and the same fail-closed shape, as CountNodeThenError above.
			// Only "from" is rejected: "to" conditions are ANDed in on
			// every route, this one included.
			//
			// n.edge.edgeWhere() already means sides()[0].Where — see
			// Edge.edgeWhere()'s own doc — so this loop only ever sees
			// Sides[0]'s own fixed condition(s), never a multi-hop edge's
			// side i>=1 conditions. That is deliberately correct: a side
			// i>=1 "from" condition lands in that hop's own JOIN ON clause
			// (edgeJoinChain, compose.go), which route 4 (a Then boundary's
			// own parent query) renders identically to routes 1/2 — it is
			// never routed through parentKeySubquery and so is never dropped
			// the way this check exists to catch. Widening this loop to walk
			// every side would reject perfectly correct multi-hop Then
			// boundaries for no reason.
			for _, w := range n.edge.edgeWhere() {
				if w.On == "from" {
					return &ThenBoundaryFromWhereError{RelName: n.edge.relName()}
				}
			}
			id := -(len(plan.thens) + 1)
			plan.thens = append(plan.thens, thenPlan{
				id:        id,
				parentSet: parentSet,
				edge:      n.edge,
				children:  n.children,
			})
			return nil
		}
		set := next
		next++
		// hasKids means "something in THIS island reads my rows": a nested
		// row section, or a count section semi-joining against my CTE.
		// Both are non-Then children, so the test is unchanged by counts —
		// but composeIsland only materializes _sl_n<set> for hasKids
		// nodes, so a count child MUST keep flipping this to true.
		hasKids := false
		for _, c := range n.children {
			if !c.then {
				hasKids = true
				break
			}
		}
		plan.sections = append(plan.sections, flatNode{
			set:       set,
			parentSet: parentSet,
			edge:      n.edge,
			hasKids:   hasKids,
		})
		for _, c := range n.children {
			if err := walk(set, c); err != nil {
				return err
			}
		}
		return nil
	}
	for _, n := range nodes {
		if err := walk(0, n); err != nil {
			return islandPlan{}, err
		}
	}
	return plan, nil
}
