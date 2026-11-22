// Package sectionload implements a third, explicitly chosen eager-loading
// strategy for bob-generated models: sectioned UNION ALL queries.
// See https://github.com/stephenafamo/bob/discussions/738 for the proposal.
//
// This file defines the DATA CONTRACT between generated code
// and the runtime layer. Generated code only fills these values; all logic
// lives in the runtime layer.
package sectionload

import "fmt"

// CarriedKeyPrefix is the fixed prefix generated code uses for a
// multi-hop edge's carried join-key column names ("_sl_k0", "_sl_k1",
// ...). It is a single constant because the prefix is needed in two
// places — the generator (templates/table/01_edges.go.tpl, which builds
// the name via CarriedKeyName) and the runtime's own reserved-name check
// (parentexpr.go's reservedNameRe, which recognizes it via a regexp
// fragment). Were it a string literal duplicated in both, changing one
// without the other would desync the generated column name from what the
// reserved-name check recognizes and silently reopen a column-name
// collision. Both sites derive from this constant (the template calls
// CarriedKeyName directly; parentexpr.go builds its regexp fragment from
// CarriedKeyPrefix via regexp.QuoteMeta).
const CarriedKeyPrefix = "_sl_k"

// ContractVersion is the data-contract version this runtime implements.
// Generated code pins the version it was generated for and
// asserts equality at compile time (see templates/bob_sectionload.bob.go.tpl).
const ContractVersion = 1

// CarriedKeyName returns the SQL-layout name for the i-th carried
// join-key column of a multi-hop edge (Edge.Carried's doc comment below):
// "_sl_k0", "_sl_k1", and so on, positionally matching Sides[0].FromCols.
func CarriedKeyName(i int) string {
	return fmt.Sprintf("%s%d", CarriedKeyPrefix, i)
}

// Column describes one SQL column of a table: its name and its database
// type. DBType is used verbatim for NULL padding casts (NULL::uuid etc.),
// mirroring how bob's own templates cast with Column.DBType.
type Column struct {
	Name   string
	DBType string
}

// TableParts is everything the runtime needs to emit and scan one table's
// section. One value per table is generated into the app (e.g. PostParts).
type TableParts[T any] struct {
	// Schema and Table are kept separate so the runtime can quote them.
	// Schema may be empty (bob's psql driver omits "public").
	Schema string
	Table  string
	// Columns in the exact order of the generated Buf targets.
	// The generator guarantees: Columns order == Targets() order.
	Columns []Column
	// NewBuf returns a fresh scan buffer for one row of this section.
	NewBuf func() Buf[T]
	// AsSlice converts to the named slice type (e.g. models.PostSlice).
	// Used to fire AfterQueryHook with bob-identical semantics.
	AsSlice func([]T) any
}

// Buf is a scan destination for one row. Every field is null-tolerant:
// when a row belongs to another section, ALL columns of this section are
// NULL (padding), even those that are NOT NULL in the database.
type Buf[T any] interface {
	// TargetsFor returns scan destinations for exactly the columns in
	// cols, in that order. An empty cols means "every column of the
	// table, in TableParts.Columns order" — the original behavior, and
	// still the only form used wherever no Edge.ColumnPlan restricts
	// columns.
	TargetsFor(cols []Column) []any
	// BuildFor converts the scanned values into the bob model, copying
	// only modelCols. A nil modelCols means "every declared model field".
	// A non-nil, empty modelCols means a restricted plan with zero
	// model-visible columns, and copies none (nil and empty must not
	// collide). Columns scanned into the buffer but absent from modelCols
	// (Internal columns) are never copied into the model. It must only be
	// called for rows that belong to this section.
	BuildFor(modelCols []Column) T
}

// ColumnPlan selects which columns of a table's row an edge fetches, and
// which of those are copied into the model. A zero ColumnPlan (nil Model,
// nil Internal) means "no restriction": every column of the table is
// fetched and copied into the model.
//
// Internal columns are fetched (they appear in the generated SQL and are
// scanned into the row buffer) but are never copied into the model, even
// though the model has no field reserved for them. They exist so a join
// key can still be read via Edge.ParentKeyFromBuf/ChildKeyFromBuf after
// the caller has excluded it from the columns they want to see.
type ColumnPlan struct {
	Model    []Column
	Internal []Column
}

// All returns every column this plan fetches from SQL: Model, then
// Internal. A zero ColumnPlan returns nil; callers that need "every
// column of the table" when All() is empty fall back to
// TableParts.Columns themselves — ColumnPlan has no reference to it.
func (p ColumnPlan) All() []Column {
	if len(p.Internal) == 0 {
		return p.Model
	}
	out := make([]Column, 0, len(p.Model)+len(p.Internal))
	out = append(out, p.Model...)
	out = append(out, p.Internal...)
	return out
}

// NewColumnPlan builds a ColumnPlan for one edge's child columns from the
// caller's column selection and the join key columns this edge's
// stitching depends on (keyCols — normally Edge.ToCols on the child
// side):
//
//   - only, except: mutually exclusive selection modes. Both empty means
//     "no restriction" — NewColumnPlan returns the zero ColumnPlan.
//   - only mode: a key column the caller did not name is added to
//     Internal automatically (fetched for stitching, never exposed on
//     the model); one the caller DID name in only stays in Model.
//   - except mode: a key column not named in except lands in
//     Model like any other unexcluded column, matching bob's own
//     psql.PreloadExcept (a plain set difference with no key-column
//     special case). A key column NAMED in except is a
//     KeyColumnExcludedError (below) — sectionload always fetches join
//     key columns for stitching regardless of the caller's selection, so
//     naming one for exclusion is rejected rather than silently kept,
//     silently dropped (breaking stitching), or silently rerouted to
//     Internal (which would produce a different result set than bob's
//     PreloadExcept for the same call).
func NewColumnPlan(all []Column, only, except, keyCols []string) (ColumnPlan, error) {
	if len(only) > 0 && len(except) > 0 {
		return ColumnPlan{}, &ColumnSelectionConflictError{}
	}

	byName := make(map[string]Column, len(all))
	for _, c := range all {
		byName[c.Name] = c
	}

	// A keyCols entry missing from the full column set is a
	// caller/generator error regardless of the selection mode, so it is
	// checked first — before the unrestricted early return and before
	// except-mode's KeyColumnExcludedError.
	for _, k := range keyCols {
		if _, ok := byName[k]; !ok {
			return ColumnPlan{}, &UnknownKeyColumnError{Column: k}
		}
	}

	if len(only) == 0 && len(except) == 0 {
		return ColumnPlan{}, nil
	}

	// onlyMode selects which of the two branches below builds `selected`:
	// Only names columns the caller explicitly wants (so a key column
	// appearing in Only really was asked for by name); Except names
	// columns to drop, so `selected` there is built as "all columns minus
	// the excluded ones" instead.
	//
	// Note on except mode: a key column that is not named in except lands
	// in Model like any other unexcluded column, to match bob's own
	// psql.PreloadExcept (a plain set difference with no key-column special
	// case). This deliberately differs from only mode, where a key column
	// must be explicitly named to stay in Model — see the branch below and
	// this function's doc comment above for the contract.
	onlyMode := len(only) > 0

	var selected map[string]bool
	if onlyMode {
		selected = make(map[string]bool, len(only))
		for _, n := range only {
			selected[n] = true
		}
	} else {
		excluded := make(map[string]bool, len(except))
		for _, n := range except {
			excluded[n] = true
		}
		for _, k := range keyCols {
			if excluded[k] {
				return ColumnPlan{}, &KeyColumnExcludedError{Column: k}
			}
		}
		selected = make(map[string]bool, len(all)-len(except))
		for _, c := range all {
			// A key column NOT named in except lands in Model like any
			// other unexcluded column — matching bob's own
			// psql.PreloadExcept (internal.Except, orm/load.go), which
			// performs a plain set difference with no key-column special
			// case. A key column that IS named in except is rejected
			// above (KeyColumnExcludedError), so this branch only ever
			// sees keys the caller did not try to exclude.
			if !excluded[c.Name] {
				selected[c.Name] = true
			}
		}
	}

	plan := ColumnPlan{
		// A restricted plan's Model is never nil: nil means "unrestricted,
		// every column", a non-nil empty slice means "restricted with zero
		// model-visible columns".
		Model: make([]Column, 0, len(all)),
	}
	for _, c := range all {
		if selected[c.Name] {
			plan.Model = append(plan.Model, c)
		}
	}
	for _, k := range keyCols {
		// Existence is guaranteed by the check at the top of this function.
		c := byName[k]
		if selected[k] {
			continue // already in Model: only-mode named it explicitly,
			// or except-mode did not exclude it
		}
		plan.Internal = append(plan.Internal, c)
	}
	return plan, nil
}

// EdgeWhere is a fixed condition attached to a relationship definition
// (bob's relationship "where"). On tells which side it constrains.
type EdgeWhere struct {
	On       string // "from" or "to"
	Column   string
	SQLValue string
}

// Side is one hop of a relationship's join chain: bob's own
// orm.Relationship.Sides, translated into the generator's data contract.
// A single-hop relationship is one Side; a through/multi-hop relationship
// (e.g. User -> user_roles -> Role) is Sides[0..n-1], where
// Sides[i-1].ToTable == Sides[i].FromTable for every i>=1 (bob's own
// Relationship.Validate() guarantees this — see orm/relationship.go). The
// generator emits table names/schemas at generation time; the runtime
// resolves them to SQL via tableExpr (compose.go), the same helper bob's
// own generated views use.
//
// FromSchema/ToSchema may be empty (bob's psql driver omits "public",
// mirroring TableParts.Schema).
type Side struct {
	FromSchema string
	FromTable  string
	FromCols   []string
	ToSchema   string
	ToTable    string
	ToCols     []string
	// Where is this hop's own fixed condition(s), not the whole edge's.
	// On == "from" applies to Sides[i].From, "to" to Sides[i].To — see
	// compose.go's edgeJoinChain/matchWhere/parentKeySubquery for where
	// each lands in the rendered SQL.
	Where []EdgeWhere
}

// Edge is one relationship edge (parent -> child), typed by the two model
// types. The generator emits one Edge per relationship, single-hop or
// multi-hop (through an intermediate table) alike.
type Edge[P, C any] struct {
	// RelName is the exact name accepted by the generated
	// (*P).Preload(name, rel) stitcher method.
	RelName string
	// FromCols/ToCols/Where are the single-hop form of the contract. They
	// are kept for generated code that sets only these, and the runtime's
	// sides() (node.go) normalizes them into a length-1 Sides whenever
	// Sides itself is empty. Current generated code sets Sides instead and
	// leaves these three empty.
	FromCols []string
	ToCols   []string
	ToMany   bool
	Where    []EdgeWhere
	// Sides is the ordered parent -> ... -> child hop chain.
	// Sides[0].FromCols are the parent's own join columns (same values
	// fromCols()/FromCols always described); Sides[len-1].ToCols are the
	// final child table's join columns. A single-hop relationship is
	// generated as a length-1 Sides, never as the empty/flat form above —
	// only older generated code leaves Sides empty and relies on the
	// FromCols/ToCols/Where fallback. Runtime code must
	// always read sides() (node.go), never Sides directly, so the two
	// forms stay indistinguishable past that one normalization point.
	Sides []Side
	// Carried is non-empty only for a multi-hop edge (len(Sides) > 1): the
	// column(s) this edge's own SQL adds, beyond the child model's own
	// columns, to carry the parent's join key value alongside each row —
	// the final child table has no column holding it (a through
	// relationship's last table does not contain the
	// first hop's join key). Each entry's Name is the SQL-layout name
	// ("_sl_k0", "_sl_k1", ... — reserved against parent-query collisions
	// by reservedNameRe in parentexpr.go) and DBType is copied from the
	// PARENT's corresponding Sides[0].FromCols[i] column, since that is
	// the type NULL-padding casts it as everywhere else in the layout
	// (section0List/unionBranch/checkWidth, compose.go). This SQL-layout
	// name is deliberately NOT the parent's own column name: were the
	// parent's real column name reused (e.g. "id") and the final child
	// table happened to also have an "id" column, route 3's read-back
	// (psql.Quote(srcAlias, name)) would collide — hence two separate
	// name lists (the SQL-layout name here, the parent's real column name
	// in carriedScanCols). Nil for a single-hop edge; the rendered SQL and
	// layout are then identical to before Carried existed.
	Carried []Column
	// NewCarriedBuf returns a fresh scan buffer for reading the Carried
	// columns back out of a row — the SAME buffer type the parent table's
	// own TableParts[P].NewBuf produces, because the carried value IS the
	// parent's own join key value and must come back as the same Go type
	// ParentKey/ParentKeyFromBuf already return (no new
	// buffer type, so a driver-vs-model type mismatch cannot occur by
	// construction). Nil for a single-hop edge.
	NewCarriedBuf func() Buf[P]
	Child         TableParts[C]
	// ColumnPlan restricts which of Child's columns this edge fetches and
	// exposes on the model. The zero value means "no restriction": every
	// column in Child.Columns is fetched and copied into the model,
	// identical to the unrestricted behavior. Build with
	// NewColumnPlan, which enforces that join key columns are always
	// fetched regardless of this selection.
	ColumnPlan ColumnPlan
	// Options carries the query mods (WHERE/ORDER BY/LIMIT/OFFSET, via Mods)
	// and Only/Except column selection to apply to this edge's own
	// rows. The zero value means no mods and no restriction — identical SQL
	// to before this field existed.
	Options SelectOptions
	// ParentKey extracts the join key values from a parent row.
	// Returns nil when the (nullable) key is not set — no link.
	ParentKey func(P) []any
	// ChildKey extracts the join key values from a child row.
	ChildKey func(C) []any
	// ParentKeyFromBuf extracts the join key values from the parent's scan
	// buffer instead of the built model. Optional: nil means "the join key
	// is always present on the model", and the runtime falls back to
	// ParentKey(buf.Build()). Generated code sets this when the
	// caller may have excluded the join key column from the parent's
	// selected columns — the key is still scanned into the buffer (an
	// Internal column, never copied into the model), so it must be read
	// there before the buffer's next reuse, not from the model afterward.
	ParentKeyFromBuf func(Buf[P]) []any
	// ChildKeyFromBuf is ChildKey's counterpart to ParentKeyFromBuf.
	ChildKeyFromBuf func(Buf[C]) []any
	// Stitch connects the collected children into one parent, converting
	// to the named slice type the generated Preload method expects.
	// For a non-nil parent key, Stitch is called exactly once for both
	// to-one and to-many edges, including when no matching child exists.
	// For a nil parent key, to-one is stitched once with no children so
	// Loaded becomes true; to-many is skipped entirely.
	Stitch func(parent P, children []C) error
}

// CountEdge is one relationship edge loaded for its COUNT only: no child
// model is ever built, only the number of matching child rows per parent.
// It is the sectionload counterpart of bob's SelectThenLoadCount /
// LoadCount<Rel>, and it reproduces the same contract: EVERY parent gets a
// non-nil *int64, zero included (nil means "not loaded", &0 means "loaded,
// no children").
//
// A count edge is always a leaf. NewCountNode takes no children at all, so
// nesting under a count node is impossible to express.
type CountEdge[P, C any] struct {
	// RelName is the exact name accepted by the generated
	// (*P).PreloadCount(name, count) setter method.
	RelName  string
	FromCols []string
	ToCols   []string
	Where    []EdgeWhere
	// Sides mirrors Edge.Sides: the ordered parent -> ... -> child
	// hop chain. Length-1 for a single-hop count; length>1 for a
	// multi-hop (through) count. sides() (node.go) falls back to
	// FromCols/ToCols/Where when Sides is empty, exactly like
	// Edge.sides().
	Sides []Side
	// KeyCols (multi-hop only) is the pair of Sides[0].ToCols (the real
	// column names of the first intermediate table) and the DBTypes
	// copied from the PARENT's corresponding Sides[0].FromCols[i] columns.
	// No TableParts is generated for the first intermediate table
	// (CountEdge.Child is only the final child table), so there is no way
	// to introspect the DBTypes at runtime; they are embedded at
	// generation time. Nil for a single-hop count (countKeyCols resolves
	// the columns from e.child().cols() instead).
	KeyCols []Column
	// NewCarriedBuf (multi-hop only) returns a fresh Buf[P] used ONLY to
	// scan a count row's grouped key value back with the exact Go type
	// ParentKey/ParentKeyFromBuf already return — the same reasoning as
	// Edge.NewCarriedBuf, but here it adds NO extra SQL SELECT column.
	// Nil for a single-hop count.
	NewCarriedBuf func() Buf[P]
	// Child is the child table this edge counts rows in. Only its schema,
	// table name, column list and scan buffer are used: the count section
	// selects nothing but ToCols and count(*), and never builds a model.
	Child TableParts[C]
	// Options carries caller-provided SELECT mods for this count's own
	// aggregate SELECT. WHERE narrows the rows counted, the
	// same way bob's own LoadCount<Rel> mods do. LIMIT/OFFSET apply to the
	// GROUPED rows — "the first N distinct parent keys", not "N children"
	// — so a parent outside that window counts 0 (still non-nil), again
	// matching bob. ORDER BY passes through to the aggregate SELECT
	// verbatim: alone it cannot change any count (count rows are consumed
	// through a key→count map), but combined with LIMIT/OFFSET it decides
	// WHICH parent-key groups are counted — the same meaning bob's own
	// LoadCount<Rel> gives the same mods.
	//
	// Only/Except cannot apply here: a count section's SELECT is
	// always exactly the join key columns plus count(*) (this struct has no
	// ColumnPlan field at all), so there is no model column to restrict.
	// Setting Only/Except on a count node's SelectOptions is rejected
	// outright with CountColumnRestrictionUnsupportedError when the tree is
	// resolved (sectionload.All/Build) — see resolveColumnPlans in
	// columnselect.go.
	Options SelectOptions
	// ParentKey extracts the join key values from a parent row.
	// Returns nil when the (nullable) key is not set — such a parent still
	// gets a count of 0, unlike a row edge, which leaves it untouched.
	ParentKey func(P) []any
	// ParentKeyFromBuf reads the same key out of the parent's scan buffer
	// instead of the built model. Optional; see Edge.ParentKeyFromBuf.
	ParentKeyFromBuf func(Buf[P]) []any
	// ChildKey extracts the grouped key values from a count row. Only the
	// ToCols columns are ever scanned into the buffer this is applied to;
	// every other field stays at its zero value.
	ChildKey func(C) []any
	// ChildKeyFromBuf is ChildKey's counterpart to ParentKeyFromBuf.
	ChildKeyFromBuf func(Buf[C]) []any
	// SetCount stores one parent's count. Generated code delegates to the
	// model's own PreloadCount(name, count), then verifies the C field is
	// no longer nil — bob's PreloadCount switch has no default case, so a
	// stale generated contract would otherwise fail silently.
	SetCount func(parent P, count int64) error
}

// Node is one node of a loading tree, produced by the generated
// tree-builder methods (e.g. generated.Post.Author(...) or
// generated.Post.CommentsCount(...)).
//
// Exactly one of edge (a row-loading relationship) and count (a
// count-only relationship) is set.
type Node struct {
	edge     edgeSpec
	count    countEdgeSpec
	children []Node
	then     bool
}

// NewNode wraps an Edge and its nested children into a tree node.
// Called by generated code only.
func NewNode[P, C any](e Edge[P, C], children ...Node) Node {
	return Node{edge: e, children: children}
}

// NewCountNode wraps a CountEdge into a tree node. Called by generated
// code only. It deliberately takes NO children: a count node loads no
// models, so there is nothing to nest under it, and leaving the parameter
// out makes that a compile error rather than a runtime one.
func NewCountNode[P, C any](e CountEdge[P, C]) Node {
	return Node{count: e}
}

// Then marks a node as a query boundary: the subtree rooted at this node
// is loaded by its own sectioned query (one extra round trip), exactly
// like an explicit ThenLoad. Use it to split very wide trees or to keep
// an existing ThenLoad boundary.
func Then(n Node) Node {
	n.then = true
	return n
}
