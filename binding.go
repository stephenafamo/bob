package bob

import (
	"errors"
	"fmt"

	"golang.org/x/exp/maps"
)

type ArgumentBinding struct {
	Name string
}

func ArgBinding(name string) ArgumentBinding {
	return ArgumentBinding{Name: name}
}

func replaceArgumentBindings(nargs []ArgumentBinding, args ...any) ([]any, error) {
	allArgs := map[string]any{}
	for _, arg := range args {
		var sourceArgs map[string]any
		switch a := arg.(type) {
		case map[string]any:
			sourceArgs = a
		}

		// must try struct also

		if sourceArgs == nil {
			return nil, errors.New("unknown arguments type")
		}

		maps.Copy(allArgs, sourceArgs)
	}

	mergedArgs := make([]any, len(nargs))
	for idx, narg := range nargs {
		if carg, ok := allArgs[narg.Name]; ok {
			mergedArgs[idx] = carg
		} else {
			return nil, fmt.Errorf("argument binding '%s' not found", narg.Name)
		}
	}

	return mergedArgs, nil
}

func NamesToArgumentBindings(names ...string) []any {
	args := make([]any, len(names))
	for idx, name := range names {
		args[idx] = ArgBinding(name)
	}
	return args
}

func FailIfArgumentBindings(args []any) error {
	for _, arg := range args {
		if _, ok := arg.(ArgumentBinding); ok {
			return errors.New("some argument bindings were not processed")
		}
	}
	return nil
}

func FailIfMixedArgumentBindings(args []any) error {
	hasBinding := false
	hasNonBinding := false
	for _, arg := range args {
		if _, ok := arg.(ArgumentBinding); ok {
			hasBinding = true
		} else {
			hasNonBinding = true
		}
	}
	if hasBinding && hasNonBinding {
		return fmt.Errorf("cannot mix argument bindings with other arguments")
	}
	return nil
}
