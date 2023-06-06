package bob

import (
	"errors"
	"fmt"

	"golang.org/x/exp/maps"
)

type NamedArgument struct {
	Name string
}

func NamedArg(name string) NamedArgument {
	return NamedArgument{Name: name}
}

func mergeNamedArguments(nargs []NamedArgument, args ...any) ([]any, error) {
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
			return nil, fmt.Errorf("named argument '%s' not found", narg.Name)
		}
	}

	return mergedArgs, nil
}

func NamesToNamedArguments(names ...string) []any {
	args := make([]any, len(names))
	for idx, name := range names {
		args[idx] = NamedArg(name)
	}
	return args
}
