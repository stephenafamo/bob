package expr

import (
	"context"
	"errors"
	"io"
	"slices"

	"github.com/stephenafamo/bob"
)

type (
	caseExpr struct {
		whens    []when
		elseExpr bob.Expression
	}
	when struct {
		condition bob.Expression
		then      bob.Expression
	}
)

func (c caseExpr) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any

	if len(c.whens) == 0 {
		return nil, errors.New("case must have at least one when expression")
	}

	w.WriteString("CASE")
	for _, when := range c.whens {
		w.WriteString(" WHEN ")
		whenArgs, err := bob.Express(ctx, w, d, start+len(args), when.condition)
		if err != nil {
			return nil, err
		}
		args = append(args, whenArgs...)

		w.WriteString(" THEN ")
		thenArgs, err := bob.Express(ctx, w, d, start+len(args), when.then)
		if err != nil {
			return nil, err
		}
		args = append(args, thenArgs...)
	}

	if c.elseExpr != nil {
		w.WriteString(" ELSE ")
		elseArgs, err := bob.Express(ctx, w, d, start+len(args), c.elseExpr)
		if err != nil {
			return nil, err
		}
		args = append(args, elseArgs...)
	}
	w.WriteString(" END")

	return args, nil
}

type CaseChain[T bob.Expression, B builder[T]] func() caseExpr

func NewCase[T bob.Expression, B builder[T]]() CaseChain[T, B] {
	return CaseChain[T, B](func() caseExpr { return caseExpr{} })
}

func (cc CaseChain[T, B]) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return cc().WriteSQL(ctx, w, d, start)
}

func (cc CaseChain[T, B]) When(condition, then bob.Expression) CaseChain[T, B] {
	c := cc()
	c.whens = append(slices.Clip(c.whens), when{condition: condition, then: then})
	return CaseChain[T, B](func() caseExpr { return c })
}

func (cc CaseChain[T, B]) Else(then bob.Expression) T {
	c := cc()
	c.elseExpr = then
	return X[T, B](c)
}

func (cc CaseChain[T, B]) End() T {
	return X[T, B](cc())
}
