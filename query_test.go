package bob

import (
	"context"
	"io"
	"testing"
)

var (
	_ Expression = &cached{}

	_ Query = BaseQuery[Expression]{}
	_ Query = BoundQuery[Expression]{}

	_ Loadable = BaseQuery[Expression]{}
	_ Loadable = BoundQuery[Expression]{}
	_ Loadable = &cached{}

	_ MapperModder = BaseQuery[Expression]{}
	_ MapperModder = BoundQuery[Expression]{}
	_ MapperModder = &cached{}

	_ HookableQuery = BaseQuery[Expression]{}
	_ HookableQuery = BoundQuery[Expression]{}
)

// allQueryTypes lists every defined QueryType value (query.go:22-28), used to
// table-test BaseQuery.Clone() across both of its branches (P-19).
var allQueryTypes = []QueryType{
	QueryTypeUnknown,
	QueryTypeSelect,
	QueryTypeInsert,
	QueryTypeUpdate,
	QueryTypeDelete,
	QueryTypeValues,
	QueryTypeMerge,
}

// cloneOwnExpr is an Expression with its own Clone() method, so
// BaseQuery.Clone() takes the "custom Clone" branch (query.go:73-77).
// tag is a pointer so a real independence check (not just Go value
// semantics) can distinguish "copied the pointer" from "copied the value".
type cloneOwnExpr struct {
	tag *string
}

func (cloneOwnExpr) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	return nil, nil
}

func (e cloneOwnExpr) Clone() cloneOwnExpr {
	cp := *e.tag
	return cloneOwnExpr{tag: &cp}
}

// cloneReprintExpr has no Clone() method, so BaseQuery.Clone() falls back to
// reprint.This (query.go:80-83). tag is a slice so reprint's deep-copy can be
// distinguished from a shallow field copy that would alias the backing array.
type cloneReprintExpr struct {
	tag []string
}

func (cloneReprintExpr) WriteSQL(ctx context.Context, w io.StringWriter, d Dialect, start int) ([]any, error) {
	return nil, nil
}

// TestBaseQueryCloneKeepsQueryType is the P-19 characterization/regression
// test: before the fix, BaseQuery.Clone() drops QueryType in both branches,
// so a cloned query's Type() always reports QueryTypeUnknown regardless of
// the original QueryType.
func TestBaseQueryCloneKeepsQueryType(t *testing.T) {
	t.Run("custom Clone() expression", func(t *testing.T) {
		for _, qt := range allQueryTypes {
			t.Run(qt.String(), func(t *testing.T) {
				tag := "own"
				original := BaseQuery[cloneOwnExpr]{
					Expression: cloneOwnExpr{tag: &tag},
					Dialect:    d,
					QueryType:  qt,
				}

				cloned := original.Clone()

				if got := cloned.Type(); got != qt {
					t.Fatalf("Clone() changed QueryType: original %s (%d), cloned %s (%d)", qt, qt, got, got)
				}
			})
		}
	})

	t.Run("reprint fallback expression", func(t *testing.T) {
		for _, qt := range allQueryTypes {
			t.Run(qt.String(), func(t *testing.T) {
				original := BaseQuery[cloneReprintExpr]{
					Expression: cloneReprintExpr{tag: []string{"reprint"}},
					Dialect:    d,
					QueryType:  qt,
				}

				cloned := original.Clone()

				if got := cloned.Type(); got != qt {
					t.Fatalf("Clone() changed QueryType: original %s (%d), cloned %s (%d)", qt, qt, got, got)
				}
			})
		}
	})
}

// TestBaseQueryCloneExpressionIndependence guards the pre-existing
// contract (unrelated to QueryType) that Clone() must not let the clone's
// Expression alias the original's mutable state, for both Clone() branches.
func TestBaseQueryCloneExpressionIndependence(t *testing.T) {
	t.Run("custom Clone() expression", func(t *testing.T) {
		tag := "own"
		original := BaseQuery[cloneOwnExpr]{
			Expression: cloneOwnExpr{tag: &tag},
			Dialect:    d,
			QueryType:  QueryTypeSelect,
		}

		cloned := original.Clone()
		*cloned.Expression.tag = "mutated"

		if *original.Expression.tag != "own" {
			t.Fatalf("Clone() leaked the original's tag pointer: original tag now %q", *original.Expression.tag)
		}
	})

	t.Run("reprint fallback expression", func(t *testing.T) {
		original := BaseQuery[cloneReprintExpr]{
			Expression: cloneReprintExpr{tag: []string{"reprint"}},
			Dialect:    d,
			QueryType:  QueryTypeSelect,
		}

		cloned := original.Clone()
		cloned.Expression.tag[0] = "mutated"

		if original.Expression.tag[0] != "reprint" {
			t.Fatalf("Clone() leaked the original's tag backing array: original tag now %q", original.Expression.tag[0])
		}
	})
}
