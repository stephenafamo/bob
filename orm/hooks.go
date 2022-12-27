package orm

import (
	"context"
	"sync"

	"github.com/stephenafamo/bob"
)

// SkipHooks modifies a context to prevent hooks from running for any query
// it encounters.
func SkipHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxSkipHooks, true)
}

// Hook is a function that can be called during lifecycle of an object
// the context can be modified and returned
// The caller is expected to use the returned context for subsequent processing
type Hook[T any] func(context.Context, bob.Executor, T) (context.Context, error)

// Hooks is a set of hooks that can be called all at once
type Hooks[T any] struct {
	mu    sync.RWMutex
	hooks []Hook[T]
}

// Add a hook to the set
func (h *Hooks[T]) Add(hook Hook[T]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.hooks = append(h.hooks, hook)
}

// Do calls all the registered hooks.
// if the context is set to skip hooks using [SkipHooks], then Do simply returns the context
func (h *Hooks[T]) Do(ctx context.Context, exec bob.Executor, o T) (context.Context, error) {
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
