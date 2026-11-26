package ormruntime

import (
	"context"

	"github.com/stephenafamo/bob"
)

// AfterModelHooks dispatches to the table's registered hooks. A nil set
// denotes an operation not supported by that table or view.
type AfterModelHooks[S any] struct {
	Select, Insert, Update, Delete, Merge *bob.Hooks[S, bob.SkipModelHooksKey]
}

func (h AfterModelHooks[S]) Run(ctx context.Context, exec bob.Executor, typ bob.QueryType, rows S) error {
	var hooks *bob.Hooks[S, bob.SkipModelHooksKey]
	switch typ {
	case bob.QueryTypeSelect:
		hooks = h.Select
	case bob.QueryTypeInsert:
		hooks = h.Insert
	case bob.QueryTypeUpdate:
		hooks = h.Update
	case bob.QueryTypeDelete:
		hooks = h.Delete
	case bob.QueryTypeMerge:
		hooks = h.Merge
	}
	if hooks == nil {
		return nil
	}
	_, err := hooks.RunHooks(ctx, exec, rows)
	return err
}
