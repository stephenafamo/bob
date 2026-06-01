package expr

// PrepareSetAssignments rewrites each expression for SET clause rendering.
// EQ equalities are returned as leftRight with omitParens set so they render without comparison parentheses.
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
		return lr.withOmitParens(true)
	}
	return e
}
