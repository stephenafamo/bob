package expr

import (
	"context"
	"errors"
	"io"

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

func (c caseExpr) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	if len(c.whens) == 0 {
		return nil, errors.New("case must have at least one when expression")
	}

	w.Write([]byte("CASE"))
	for _, when := range c.whens {
		w.Write([]byte(" WHEN "))
		whenArgs, err := when.condition.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, whenArgs...)

		w.Write([]byte(" THEN "))
		thenArgs, err := when.then.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, thenArgs...)
	}

	if c.elseExpr != nil {
		w.Write([]byte(" ELSE "))
		elseArgs, err := c.elseExpr.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, elseArgs...)
	}
	w.Write([]byte(" END"))

	return args, nil
}

type CaseChain[T bob.Expression, B builder[T]] func() caseExpr

func NewCase[T bob.Expression, B builder[T]]() CaseChain[T, B] {
	return CaseChain[T, B](func() caseExpr { return caseExpr{} })
}

func (cc CaseChain[T, B]) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return cc().WriteSQL(ctx, w, d, start)
}

func (cc CaseChain[T, B]) When(condition, then bob.Expression) CaseChain[T, B] {
	c := cc()
	c.whens = append(c.whens, when{condition: condition, then: then})
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
