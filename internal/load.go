package internal

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
)

// NewAfterPreloader returns a new AfterPreloader based on the given types
func NewAfterPreloader[T any, Ts ~[]T]() *AfterPreloader {
	var one T
	var slice Ts
	return &AfterPreloader{
		oneType:   reflect.TypeOf(one),
		sliceType: reflect.TypeOf(slice),
	}
}

// AfterPreloader is embedded in a Preloader to chain loading
// whenever a preloaded object is scanned, it should be collected with the Collect method
// The loading functions should be added with AppendLoader
// later, when this object is called like any other [bob.Loader], it
// calls the appended loaders with the collected objects
type AfterPreloader struct {
	oneType   reflect.Type
	sliceType reflect.Type

	funcs     []bob.Loader
	collected []any
}

func (a *AfterPreloader) AppendLoader(fs ...bob.Loader) {
	a.funcs = append(a.funcs, fs...)
}

func (a *AfterPreloader) Collect(v any) error {
	if len(a.funcs) == 0 {
		return nil
	}

	if reflect.TypeOf(v) != a.oneType {
		return fmt.Errorf("Expected to receive %s but got %T", a.oneType.String(), v)
	}

	a.collected = append(a.collected, v)
	return nil
}

func (a *AfterPreloader) Load(ctx context.Context, exec bob.Executor, _ any) error {
	if len(a.collected) == 0 || len(a.funcs) == 0 {
		return nil
	}

	obj := a.collected[0]

	if len(a.collected) > 1 {
		all := reflect.MakeSlice(a.sliceType, len(a.collected), len(a.collected))
		for k, v := range a.collected {
			all.Index(k).Set(reflect.ValueOf(v))
		}

		obj = all.Interface()
	}

	for _, f := range a.funcs {
		if err := f.Load(ctx, exec, obj); err != nil {
			return err
		}
	}

	return nil
}
