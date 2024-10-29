package bob

import (
	"context"
	"sync"
)

type (
	// If set to true, query hooks are skipped
	SkipQueryHooksKey struct{}
	// If set to true, model hooks are skipped
	SkipModelHooksKey struct{}
	// If set to true, contextual mods are skipped
	SkipContextualModsKey struct{}
)

// SkipHooks modifies a context to prevent hooks from running for any query
// it encounters.
func SkipHooks(ctx context.Context) context.Context {
	ctx = SkipModelHooks(ctx)
	ctx = SkipQueryHooks(ctx)
	return ctx
}

// SkipModelHooks modifies a context to prevent hooks from running on models.
func SkipModelHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, SkipModelHooksKey{}, true)
}

// SkipQueryHooks modifies a context to prevent hooks from running on querys.
func SkipQueryHooks(ctx context.Context) context.Context {
	return context.WithValue(ctx, SkipQueryHooksKey{}, true)
}

// Hook is a function that can be called during lifecycle of an object
// the context can be modified and returned
// The caller is expected to use the returned context for subsequent processing
type Hook[T any] func(context.Context, Executor, T) (context.Context, error)

// Hooks is a set of hooks that can be called all at once
type Hooks[T any, K any] struct {
	mu    sync.RWMutex
	hooks []Hook[T]
	key   K
}

// AppendHooks a hook to the set
func (h *Hooks[T, K]) AppendHooks(hooks ...Hook[T]) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.hooks = append(h.hooks, hooks...)
}

// GetHooks returns all the hooks in the set
func (h *Hooks[T, K]) GetHooks() []Hook[T] {
	return h.hooks
}

// RunHooks calls all the registered hooks.
// if the context is set to skip hooks using [SkipHooks], then RunHooks simply returns the context
func (h *Hooks[T, K]) RunHooks(ctx context.Context, exec Executor, o T) (context.Context, error) {
	if len(h.hooks) == 0 {
		return ctx, nil
	}

	if skip, ok := ctx.Value(h.key).(bool); skip && ok {
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

type EmbeddedHook struct {
	Hooks []func(context.Context, Executor) (context.Context, error)
}

func (h *EmbeddedHook) SetHooks(hooks ...func(context.Context, Executor) (context.Context, error)) {
	h.Hooks = hooks
}

func (h *EmbeddedHook) AppendHooks(hooks ...func(context.Context, Executor) (context.Context, error)) {
	h.Hooks = append(h.Hooks, hooks...)
}

func (h *EmbeddedHook) RunHooks(ctx context.Context, exec Executor) (context.Context, error) {
	if len(h.Hooks) == 0 {
		return ctx, nil
	}

	for _, hook := range h.Hooks {
		var err error
		if ctx, err = hook(ctx, exec); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
