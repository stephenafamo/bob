package antlrhelpers

import (
	"fmt"
	"strings"
)

type Functions map[string]Function

type Function struct {
	RequiredArgs         int
	Variadic             bool
	Args                 []string
	ReturnType           string
	CalcReturnType       func(...string) string // If present, will be used to calculate the return type
	ShouldArgsBeNullable bool
	CalcNullable         func(...func() bool) func() bool // will be provided with the nullability of the args
}

func (f Function) ArgType(i int) string {
	if i >= len(f.Args) {
		return f.Args[len(f.Args)-1]
	}

	return f.Args[i]
}

func GetFunctionType(functions Functions, funcName string, argTypes []string) (Function, error) {
	f, ok := functions[funcName]
	if !ok {
		return Function{}, fmt.Errorf("function %q not found", funcName)
	}

	if len(argTypes) < f.RequiredArgs {
		return Function{}, fmt.Errorf("too few arguments for function %q, %d/%d", funcName, len(argTypes), f.RequiredArgs)
	}

	if !f.Variadic && len(argTypes) > len(f.Args) {
		return Function{}, fmt.Errorf("too many arguments for function %q, %d/%d", funcName, len(argTypes), len(f.Args))
	}

	for i, arg := range argTypes {
		// We don't know the type of the given argument
		if arg == "" {
			continue
		}

		argID := i
		if f.Variadic && i >= len(f.Args) {
			argID = len(f.Args) - 1
		}

		// means the func can take any type in this position
		if f.Args[argID] == "" {
			continue
		}

		if !strings.EqualFold(f.Args[argID], arg) {
			return Function{}, fmt.Errorf("function %q(%s) expects %s at position %d, got %s", funcName, strings.Join(argTypes, ", "), f.Args[argID], i+1, arg)
		}
	}

	if f.CalcReturnType != nil {
		f.ReturnType = f.CalcReturnType(argTypes...)
	}

	return f, nil
}
