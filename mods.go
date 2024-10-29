package bob

import "context"

// Mod is a generic interface for modifying a query
// It is the building block for creating queries
type Mod[T any] interface {
	Apply(T)
}

type ModFunc[T any] func(T)

func (m ModFunc[T]) Apply(query T) {
	m(query)
}

type Mods[T any] []Mod[T]

func (m Mods[T]) Apply(query T) {
	for _, v := range m {
		v.Apply(query)
	}
}

// ToMods converts a slice of a type that implements Mod[T] to Mods[T]
// this is useful since a slice of structs that implement Mod[T]
// cannot be directly used as a slice of Mod[T]
func ToMods[T Mod[Q], Q any](r ...T) Mods[Q] {
	result := make([]Mod[Q], len(r))
	for i, v := range r {
		result[i] = v
	}
	return result
}

// ContextualMods are special types of mods that require a context.
// they are only applied at the point of building the query
// where possible, prefer using regular mods since they are applied once
// while contextual mods are applied every time a query is built
type ContextualMod[T any] interface {
	Apply(context.Context, T) (context.Context, error)
}

type ContextualModFunc[T any] func(context.Context, T) (context.Context, error)

func (c ContextualModFunc[T]) Apply(ctx context.Context, o T) (context.Context, error) {
	return c(ctx, o)
}

type ContextualModdable[T any] struct {
	Mods []ContextualMod[T]
}

// AppendContextualMod a hook to the set
func (h *ContextualModdable[T]) AppendContextualMod(mods ...ContextualMod[T]) {
	h.Mods = append(h.Mods, mods...)
}

// AppendContextualMod a hook to the set
func (h *ContextualModdable[T]) AppendContextualModFunc(f func(context.Context, T) (context.Context, error)) {
	h.Mods = append(h.Mods, ContextualModFunc[T](f))
}

func (h *ContextualModdable[T]) RunContextualMods(ctx context.Context, o T) (context.Context, error) {
	if len(h.Mods) == 0 {
		return ctx, nil
	}

	if skip, ok := ctx.Value(SkipContextualModsKey{}).(bool); skip && ok {
		return ctx, nil
	}

	var err error

	for _, mod := range h.Mods {
		if ctx, err = mod.Apply(ctx, o); err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}
