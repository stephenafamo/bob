package orm

import (
	"context"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/scan"
)

type ExtraLoader struct {
	Fs        []bob.LoadFunc
	OneType   reflect.Type
	SliceType reflect.Type

	collected []any
}

func (a *ExtraLoader) AppendLoader(fs ...bob.LoadFunc) {
	a.Fs = append(a.Fs, fs...)
}

func (a *ExtraLoader) Collect(v any) error {
	if reflect.TypeOf(v) != a.OneType {
		return fmt.Errorf("Expected to receive %s but got %T", a.OneType.String(), v)
	}

	a.collected = append(a.collected, v)
	return nil
}

func (a *ExtraLoader) LoadOne(ctx context.Context, exec scan.Queryer) error {
	if len(a.collected) == 0 || len(a.Fs) == 0 {
		return nil
	}

	if len(a.collected) > 1 {
		return fmt.Errorf("Called LoadOne() when there are %d values", len(a.collected))
	}

	for _, f := range a.Fs {
		if err := f(ctx, exec, a.collected[0]); err != nil {
			return err
		}
	}

	return nil
}

func (a *ExtraLoader) LoadMany(ctx context.Context, exec scan.Queryer) error {
	if len(a.collected) == 0 || len(a.Fs) == 0 {
		return nil
	}

	all := reflect.MakeSlice(a.SliceType, len(a.collected), len(a.collected))
	for k, v := range a.collected {
		all.Index(k).Set(reflect.ValueOf(v))
	}

	for _, f := range a.Fs {
		if err := f(ctx, exec, all.Interface()); err != nil {
			return err
		}
	}

	return nil
}
