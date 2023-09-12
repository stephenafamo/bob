package bob

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/stephenafamo/bob/internal/mappings"
	"github.com/stephenafamo/scan"
)

var mapperSource, _ = scan.NewStructMapperSource()

type MismatchedArgsError struct {
	Expected int
	Got      int
}

func (e MismatchedArgsError) Error() string {
	return fmt.Sprintf("expected %d args, got %d", e.Expected, e.Got)
}

type DuplicateArgError struct{ Name string }

func (e DuplicateArgError) Error() string {
	return fmt.Sprintf("duplicate arg %s", e.Name)
}

type MissingArgError struct{ Name string }

func (e MissingArgError) Error() string {
	return fmt.Sprintf("missing arg %s", e.Name)
}

type binder[T any] interface {
	ToArgs(T) []any
}

type BoundQuery[T any] struct {
	query  []byte
	binder binder[T]
	start  int
}

func (b BoundQuery[T]) Bind(args T) BaseQuery[*cached] {
	realArgs := b.binder.ToArgs(args)

	return BaseQuery[*cached]{
		Expression: &cached{
			query: b.query,
			args:  realArgs,
			start: b.start,
		},
	}
}

func BindNamed[T any](q Query) (BoundQuery[T], error) {
	return BindNamedN[T](q, 1)
}

func BindNamedN[T any](q Query, start int) (BoundQuery[T], error) {
	query, args, err := BuildN(q, start)
	if err != nil {
		return BoundQuery[T]{}, err
	}

	binder, err := makeStructBinder[T](args)
	if err != nil {
		return BoundQuery[T]{}, err
	}

	return BoundQuery[T]{
		query:  []byte(query),
		binder: binder,
		start:  start,
	}, nil
}

type structBinder[T any] struct {
	args   []string
	fields []string
}

func (b structBinder[T]) Inspect() []string {
	names := make([]string, len(b.args))
	for _, name := range b.args {
		if name == "" {
			continue
		}

		names = append(names, name)
	}

	return names
}

func (b structBinder[T]) ToArgs(arg T) []any {
	val := reflect.ValueOf(arg)
	if val.Kind() == reflect.Pointer {
		if val.IsNil() {
			return make([]any, len(b.args))
		}
		val = val.Elem()
	}

	values := make([]any, len(b.args))

ArgLoop:
	for index, argName := range b.args {
		for _, fieldName := range b.fields {
			if fieldName == argName {
				field := val.Field(index)
				values[index] = field.Interface()
				continue ArgLoop
			}
		}
	}

	return values
}

func makeStructBinder[Arg any](args []any) (structBinder[Arg], error) {
	var x Arg
	typ := reflect.TypeOf(x)

	isStruct := typ.Kind() == reflect.Struct
	if typ.Kind() == reflect.Ptr {
		isStruct = typ.Elem().Kind() == reflect.Struct
	}

	if !isStruct {
		return structBinder[Arg]{}, errors.New("bind type must be a struct")
	}

	givenArgs := make([]any, len(args))
	argPositions := make([]string, len(args))
	for pos, arg := range args {
		if name, ok := arg.(namedArg); ok {
			argPositions[pos] = string(name)
			continue
		}

		if name, ok := arg.(named); ok && len(name.names) == 1 {
			argPositions[pos] = string(name.names[0])
			continue
		}

		givenArgs[pos] = arg
	}

	fieldPositions := mappings.GetMappings(reflect.TypeOf(x)).All
	fmt.Println(fieldPositions)

	// check if all positions have matching fields
ArgLoop:
	for _, name := range argPositions {
		for _, field := range fieldPositions {
			if field == name {
				continue ArgLoop
			}
		}
		return structBinder[Arg]{}, MissingArgError{Name: name}
	}

	return structBinder[Arg]{
		args:   argPositions,
		fields: fieldPositions,
	}, nil
}
