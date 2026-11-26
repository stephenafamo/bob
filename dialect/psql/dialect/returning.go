package dialect

import (
	"context"

	"github.com/stephenafamo/bob"
)

type returningQuery interface {
	HasReturning() bool
	SetReturning(...any)
}

// DefaultReturning supplies a model's projection. It only replaces a slice
// header on the renderer's local query; it never writes to an existing list.
type DefaultReturning[Q returningQuery] struct {
	Columns bob.Expression
	Enabled func(context.Context) bool
}

func (r DefaultReturning[Q]) Edit(ctx context.Context, q Q) (context.Context, error) {
	if !q.HasReturning() && (r.Enabled == nil || r.Enabled(ctx)) {
		q.SetReturning(r.Columns)
	}
	return ctx, nil
}
