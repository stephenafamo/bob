package sectionload

// resolveColumnPlans walks the declared tree once, turning each
// row edge's caller-provided Options.Only/Except column selection into a
// concrete ColumnPlan attached to a COPY of that edge (edgeSpec.
// withColumnPlan) — never mutating the Edge value the caller's generated
// tree-builder call already produced, so the same Node value can safely
// be reused across multiple All()/Build() calls.
//
// It must run after the parent query's hooks (the hook-first
// contract: hooks always run even when the declared tree turns out to be
// invalid) and before validatePlan (validate.go), which needs to see the
// resolved ColumnPlan to require the matching *KeyFromBuf overrides —
// see run.go's call sites for exactly where this sits in All/Build.
//
// A count node's Options.Only/Except is rejected outright
// (CountColumnRestrictionUnsupportedError): a count section's SELECT is
// always keyCols+count(*) (CountEdge has no ColumnPlan field at all —
// parts.go), so there is no model column to restrict, and silently
// ignoring the caller's instruction would violate sectionload's
// never-silently-no-op rule.
//
// resolveNode recurses into n.children regardless of the Then flag (same
// as validatePlan's own walk), so a Then(...) boundary's subtree is
// ALREADY resolved by the time planIsland splits it into a thenPlan.
// runThen (hooks.go) only calls planIsland(tp.children) on that
// already-resolved subtree — it must NOT call resolveColumnPlans again.
// Implementers must not add a second resolveColumnPlans call inside
// runThen.
func resolveColumnPlans(nodes []Node) ([]Node, error) {
	out := make([]Node, len(nodes))
	for i, n := range nodes {
		r, err := resolveNode(n)
		if err != nil {
			return nil, err
		}
		out[i] = r
	}
	return out, nil
}

func resolveNode(n Node) (Node, error) {
	if n.count != nil {
		opts := n.count.options()
		if opts.onlySet || opts.exceptSet {
			return Node{}, &CountColumnRestrictionUnsupportedError{RelName: n.count.relName()}
		}
		// A count node never has children (NewCountNode takes none), so
		// there is nothing further to walk.
		return n, nil
	}

	e := n.edge
	opts := e.options()
	if opts.onlySet || opts.exceptSet {
		plan, err := resolveEdgeColumnPlan(e, opts)
		if err != nil {
			return Node{}, err
		}
		e = e.withColumnPlan(plan)
	}
	n.edge = e

	if len(n.children) > 0 {
		children := make([]Node, len(n.children))
		for i, c := range n.children {
			rc, err := resolveNode(c)
			if err != nil {
				return Node{}, err
			}
			children[i] = rc
		}
		n.children = children
	}
	return n, nil
}

// resolveEdgeColumnPlan validates one edge's Only/Except selection and
// builds its ColumnPlan. Checks run in this order, all before any SQL is
// built:
//  1. both Only and Except set → ColumnSelectionConflictError (existing
//     type, parts.go/errors.go — reused as-is; it carries no RelName,
//     a limitation of that type).
//  2. Only or Except set with zero names → ColumnSelectionEmptyError.
//  3. a multi-hop edge (len(sides()) > 1) →
//     MultiHopColumnRestrictionUnsupportedError (rejected outright —
//     see the type's own doc comment in errors.go for why an empty
//     keyCols was considered and rejected).
//     Checked before the column-name checks below because it is a
//     structural precondition, not an input-content mistake.
//  4. a named column that is not one of the child table's own columns →
//     UnknownColumnError. NewColumnPlan itself does not check this for
//     only/except (only for keyCols — parts.go), so this check must
//     happen here, before NewColumnPlan is called.
//  5. the same column named twice in one call (e.g. Except("a", "a")) →
//     DuplicateColumnError.
//  6. delegates to NewColumnPlan for the join-key handling
//     (KeyColumnExcludedError, UnknownKeyColumnError — unaffected by
//     this function).
func resolveEdgeColumnPlan(e edgeSpec, opts SelectOptions) (ColumnPlan, error) {
	if opts.onlySet && opts.exceptSet {
		return ColumnPlan{}, &ColumnSelectionConflictError{}
	}
	field, names := "Only", opts.only
	if opts.exceptSet {
		field, names = "Except", opts.except
	}
	if len(names) == 0 {
		return ColumnPlan{}, &ColumnSelectionEmptyError{RelName: e.relName(), Field: field}
	}

	// A multi-hop edge is rejected
	// outright, not attempted with an empty keyCols. e.toCols() (==
	// sides()[0].ToCols) names a column of the FIRST INTERMEDIATE table
	// for a multi-hop edge, never the final child table this ColumnPlan
	// would restrict, so there is no correct join-key column to protect
	// on that final table — computing a plan without one would silently
	// open a path where the caller could exclude a column stitching
	// secretly depends on (childKeyFrom reads the carried buffer for a
	// multi-hop edge, never this ColumnPlan's Model/Internal split —
	// node.go's childKeyFrom doc).
	if len(e.sides()) > 1 {
		return ColumnPlan{}, &MultiHopColumnRestrictionUnsupportedError{RelName: e.relName()}
	}

	all := e.child().cols()
	byName := make(map[string]bool, len(all))
	for _, c := range all {
		byName[c.Name] = true
	}
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		if !byName[name] {
			return ColumnPlan{}, &UnknownColumnError{RelName: e.relName(), Field: field, Column: name}
		}
		if seen[name] {
			return ColumnPlan{}, &DuplicateColumnError{RelName: e.relName(), Field: field, Column: name}
		}
		seen[name] = true
	}

	// e.sides() has exactly 1 element at this point (multi-hop already
	// rejected above), so e.toCols() == sides()[0].ToCols names the
	// CHILD table's own join key columns — the same table `all`
	// enumerates.
	keyCols := e.toCols()

	return NewColumnPlan(all, opts.only, opts.except, keyCols)
}
