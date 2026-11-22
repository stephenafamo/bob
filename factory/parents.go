package factory

import "maps"

// Parents records the available ancestors by relationship, not just by model
// type: two foreign keys to the same table can require different parents.
type Parents map[string]any

// With makes an independent branch of the creation path. Sibling factories
// must not see models created in another branch.
func (p Parents) With(model any, keys ...string) Parents {
	if len(keys) == 0 {
		return p
	}
	next := make(Parents, len(p)+len(keys))
	maps.Copy(next, p)
	for _, k := range keys {
		next[k] = model
	}
	return next
}

// Parent returns the ancestor of the requested type, or nil if absent.
func Parent[T any](p Parents, key string) *T {
	model, _ := p[key].(*T)
	return model
}
