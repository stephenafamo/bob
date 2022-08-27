package orm

import (
	"context"
	"sync"

	"github.com/stephenafamo/scan"
)

// SkipHooks modifies a context to prevent hooks from running for any query
// it encounters.
func SkipHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxSkipHooks, true)
}

type Hook[T any] func(context.Context, scan.Queryer, T) (context.Context, error)

type Hooks[T any] struct {
	mu    sync.RWMutex
	hooks []Hook[T]
}

func (h *Hooks[T]) Add(hook Hook[T]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.hooks = append(h.hooks, hook)
}

func (h *Hooks[T]) Do(ctx context.Context, exec scan.Queryer, o T) (context.Context, error) {
	if skip, ok := ctx.Value(ctxSkipHooks).(bool); skip && ok {
		return ctx, nil
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	var err error

	for _, hook := range h.hooks {
		if ctx, err = hook(ctx, exec, o); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
