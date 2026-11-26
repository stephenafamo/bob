package bob

import (
	"context"
	"slices"
	"sync"

	"github.com/stephenafamo/scan"
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
	list hookList[Hook[T]]
	key  K
}

// Lists are append-only: a run can read the existing entries without holding
// the lock while calling user code.
type hookList[F any] struct {
	mu    sync.RWMutex
	hooks []F
}

func (h *hookList[F]) append(hooks []F) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.hooks = append(h.hooks, hooks...)
}

func (h *hookList[F]) registered() []F {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.hooks
}

// AppendHooks registers callbacks in the order supplied.
func (h *Hooks[T, K]) AppendHooks(hooks ...Hook[T]) { h.list.append(hooks) }

// GetHooks returns a copy of the registered hooks.
func (h *Hooks[T, K]) GetHooks() []Hook[T] { return slices.Clone(h.list.registered()) }

// RunHooks calls the hooks registered when the call starts.
// if the context is set to skip hooks using [SkipHooks], then RunHooks simply returns the context
func (h *Hooks[T, K]) RunHooks(ctx context.Context, exec Executor, o T) (context.Context, error) {
	hooks := h.list.registered()

	if len(hooks) == 0 {
		return ctx, nil
	}
	if skip, ok := ctx.Value(h.key).(bool); skip && ok {
		return ctx, nil
	}
	var err error

	for _, hook := range hooks {
		if ctx, err = hook(ctx, exec, o); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

// HasHooks reports whether the registry has any callbacks.
func (h *Hooks[T, K]) HasHooks() bool { return len(h.list.registered()) != 0 }

// QueryHook edits a query for one build. exec is the executor passed to Build
// or to the execution method, and can be nil. Do not retain the editor.
type QueryHook[E any] func(context.Context, Executor, E) (context.Context, error)

// QueryHooks registers transformations for new queries. Its zero value is ready for use.
type QueryHooks[Q any] struct{ list hookList[QueryHook[Q]] }

// AppendHooks registers query hooks in the order supplied.
func (h *QueryHooks[Q]) AppendHooks(hooks ...QueryHook[Q]) { h.list.append(hooks) }

// GetHooks returns a copy of the registered query hooks.
func (h *QueryHooks[Q]) GetHooks() []QueryHook[Q] { return slices.Clone(h.list.registered()) }

// HasHooks reports whether any query hooks are registered.
func (h *QueryHooks[Q]) HasHooks() bool { return len(h.list.registered()) != 0 }

// RunHooks passes each hook's result to the next hook, in registration order.
func (h *QueryHooks[Q]) RunHooks(ctx context.Context, exec Executor, q Q) (context.Context, error) {
	if skip, _ := ctx.Value(SkipQueryHooksKey{}).(bool); skip {
		return ctx, nil
	}
	return runHooks(ctx, exec, q, h.list.registered())
}

func runHooks[Q any](ctx context.Context, exec Executor, q Q, hooks []QueryHook[Q]) (context.Context, error) {
	var err error
	for _, hook := range hooks {
		ctx, err = hook(ctx, exec, q)
		if err != nil {
			return ctx, err
		}
	}
	return ctx, nil
}

// HookableQuery returns the context and statement to use for one build.
// The returned statement may have a different concrete type.
type HookableQuery interface {
	Query
	RunHooks(context.Context, Executor) (context.Context, Query, error)
}

// RunQueryHooks resolves any Query for a build. Queries without hooks pass
// through unchanged. Building or executing the result does not repeat its query
// hooks. Model callbacks still run when the query is prepared for execution.
func RunQueryHooks(ctx context.Context, exec Executor, q Query) (context.Context, Query, error) {
	h, ok := q.(HookableQuery)
	if !ok {
		return ctx, q, nil
	}
	ctx, stmt, err := h.RunHooks(ctx, exec)
	if err != nil {
		return ctx, stmt, err
	}
	// Resolution selects the SQL and scan callbacks, not a new model
	// lifecycle. Executing a separately resolved query must use the same
	// before-model callbacks as executing the original directly.
	before, _ := q.(embeddedHookRunner)
	return ctx, completedQuery{Query: stmt, before: before}, nil
}

// A caller that immediately renders the result has no need to wrap it to
// prevent a second hook pass.
func resolveForBuild(ctx context.Context, exec Executor, q Query) (context.Context, Query, error) {
	if h, ok := q.(HookableQuery); ok {
		return h.RunHooks(ctx, exec)
	}
	return ctx, q, nil
}

type embeddedHookRunner interface {
	RunEmbeddedHooks(context.Context, Executor) (context.Context, error)
}

// Hide completed query hooks while forwarding the optional callbacks.
type completedQuery struct {
	Query
	before embeddedHookRunner
}

func (q completedQuery) RunEmbeddedHooks(ctx context.Context, exec Executor) (context.Context, error) {
	if q.before != nil {
		return q.before.RunEmbeddedHooks(ctx, exec)
	}
	return ctx, nil
}

func runEmbeddedHooks(ctx context.Context, exec Executor, q any) (context.Context, error) {
	if h, ok := q.(embeddedHookRunner); ok {
		return h.RunEmbeddedHooks(ctx, exec)
	}
	return ctx, nil
}

// Before-model callbacks can supply values used by query hooks and by SQL
// rendering. Only run them when preparing a query for execution.
func resolveForExecution(ctx context.Context, exec Executor, q Query) (context.Context, Query, error) {
	ctx, err := runEmbeddedHooks(ctx, exec, q)
	if err != nil {
		return ctx, q, err
	}
	return resolveForBuild(ctx, exec, q)
}

func (q completedQuery) GetLoaders(ctx context.Context) []Loader {
	return loadersFor(ctx, q.Query)
}

func (q completedQuery) GetMapperMods(ctx context.Context) []scan.MapperMod {
	return mapperModsFor(ctx, q.Query)
}

type EmbeddedHook struct {
	Hooks []func(context.Context, Executor) (context.Context, error)
}

func (h *EmbeddedHook) SetHooks(hooks ...func(context.Context, Executor) (context.Context, error)) {
	h.Hooks = slices.Clone(hooks)
}

func (h *EmbeddedHook) AppendHooks(hooks ...func(context.Context, Executor) (context.Context, error)) {
	h.Hooks = append(h.Hooks, hooks...)
}

// RunEmbeddedHooks runs the lifecycle callbacks attached to a statement's AST.
func (h *EmbeddedHook) RunEmbeddedHooks(ctx context.Context, exec Executor) (context.Context, error) {
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
