package expr

// PrepareSetAssignments rewrites each expression for SET clause rendering.
// EQ equalities are unwrapped from chain/group so they render as col = val without comparison parentheses.
func PrepareSetAssignments(items []any) []any {
	if len(items) == 0 {
		return items
	}
	out := make([]any, len(items))
	for i, e := range items {
		out[i] = prepareSetAssignment(e)
	}
	return out
}

func prepareSetAssignment(e any) any {
	inner := findBaseExpression(e)
	if g, ok := inner.(group); ok && len(g) == 1 {
		inner = g[0]
	}
	if lr, ok := inner.(leftRight); ok && lr.operator == "=" {
		return lr
	}
	return e
}
