package bob

import (
	"context"
	"io"
	"slices"
	"testing"
)

type cloneableExpr struct {
	parts []string
}

func (e *cloneableExpr) Clone() *cloneableExpr {
	return &cloneableExpr{
		parts: append([]string(nil), e.parts...),
	}
}

func (e *cloneableExpr) WriteSQL(context.Context, io.StringWriter, Dialect, int) ([]any, error) {
	return nil, nil
}

type appendExprMod string

func (m appendExprMod) Apply(e *cloneableExpr) {
	e.parts = append(e.parts, string(m))
}

func TestBaseQueryWithDoesNotMutateOriginal(t *testing.T) {
	base := BaseQuery[*cloneableExpr]{
		Expression: &cloneableExpr{parts: []string{"base"}},
		QueryType:  QueryTypeSelect,
	}

	derived := base.With(appendExprMod("derived"))

	if !slices.Equal(base.Expression.parts, []string{"base"}) {
		t.Fatalf("base query changed unexpectedly: %#v", base.Expression.parts)
	}

	if !slices.Equal(derived.Expression.parts, []string{"base", "derived"}) {
		t.Fatalf("derived query mismatch: %#v", derived.Expression.parts)
	}
}

func TestBaseQueryApplyDoesNotMutateOriginal(t *testing.T) {
	base := BaseQuery[*cloneableExpr]{
		Expression: &cloneableExpr{parts: []string{"base"}},
		QueryType:  QueryTypeSelect,
	}

	derived := base.Apply(appendExprMod("derived"))

	if !slices.Equal(base.Expression.parts, []string{"base"}) {
		t.Fatalf("base query changed unexpectedly: %#v", base.Expression.parts)
	}

	if !slices.Equal(derived.Expression.parts, []string{"base", "derived"}) {
		t.Fatalf("derived query mismatch: %#v", derived.Expression.parts)
	}
}
