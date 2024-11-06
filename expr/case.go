package expr

import (
	"context"
	"errors"
	"io"

	"github.com/stephenafamo/bob"
)

type Case[T bob.Expression] struct {
	Whens []When
	Else  bob.Expression
}

type When struct {
	Condition bob.Expression
	Then      bob.Expression
}

func (c Case[T]) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	var args []any

	if c.Else == nil && len(c.Whens) == 0 {
		return nil, errors.New("case must have at least one when expression")
	}

	io.WriteString(w, "CASE")
	for _, when := range c.Whens {
		io.WriteString(w, " WHEN ")
		whenArgs, err := when.Condition.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, whenArgs...)

		io.WriteString(w, " THEN ")
		thenArgs, err := when.Then.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, thenArgs...)
	}

	if c.Else != nil {
		io.WriteString(w, " ELSE ")
		elseArgs, err := c.Else.WriteSQL(ctx, w, d, start+len(args))
		if err != nil {
			return nil, err
		}
		args = append(args, elseArgs...)
	}
	io.WriteString(w, " END")

	// as

	return args, nil
}
