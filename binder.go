package bob

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/stephenafamo/bob/internal/mappings"
)

//nolint:gochecknoglobals
var (
	ErrBadArgType       = errors.New("bind type of multiple named args must be a struct, pointer to struct or map with ~string keys")
	ErrTooManyNamedArgs = errors.New("too many named args for single arg binder")
	driverValuerIntf    = reflect.TypeFor[driver.Valuer]()
	timeType            = reflect.TypeFor[time.Time]()
)

type MissingArgError struct{ Name string }

func (e MissingArgError) Error() string {
	return fmt.Sprintf("missing arg %s", e.Name)
}

type binder[T any] interface {
	// list returns the names of the args that the binder expects
	list() []string
	// Return the args to be run in the query
	// this should also include any non-named args in the original query
	toArgs(T) []any
}

func bindArgs[Arg any](args []any, named Arg) ([]any, error) {
	binder, err := makeBinder[Arg](args)
	if err != nil {
		return nil, err
	}

	return binder.toArgs(named), nil
}

func makeBinder[Arg any](args []any) (binder[Arg], error) {
	namedArgs := countNamedArgs(args)

	switch namedArgs {
	case 0: // no named args
		return emptyBinder[Arg](args), nil
	case 1: // only one named arg
		return makeSingleArgBinder[Arg](args)
	default:
		return makeMultiArgBinder[Arg](args)
	}
}

func canUseAsSingleValue(typ reflect.Type) bool {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Bool, reflect.String,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	case reflect.Slice:
		return typ.Elem().Kind() == reflect.Uint8
	}

	if typ == timeType {
		return true
	}

	if typ.Implements(driverValuerIntf) {
		return true
	}

	return false
}

func makeSingleArgBinder[Arg any](args []any) (binder[Arg], error) {
	typ := reflect.TypeFor[Arg]()
	if !canUseAsSingleValue(typ) {
		return makeMultiArgBinder[Arg](args)
	}

	givenArg := make([]any, len(args))
	copy(givenArg, args)

	b := singleBinder[Arg]{givenArg: givenArg}

	for pos, arg := range args {
		if name, ok := arg.(namedArg); ok {
			b.argIndexs = append(b.argIndexs, pos)
			b.name = string(name)
		}
	}

	return b, nil
}

func makeMultiArgBinder[Arg any](args []any) (binder[Arg], error) {
	typ := reflect.TypeFor[Arg]()

	switch typ.Kind() {
	case reflect.Map:
		if typ.Key().Kind() != reflect.String {
			return nil, ErrBadArgType
		}

		return makeMapBinder[Arg](args), nil

	case reflect.Struct:
		return makeStructBinder[Arg](args)

	case reflect.Ptr:
		if typ.Elem().Kind() == reflect.Struct {
			return makeStructBinder[Arg](args)
		}
	}

	return nil, ErrBadArgType
}

type emptyBinder[Arg any] []any

func (b emptyBinder[Arg]) list() []string {
	return nil
}

func (b emptyBinder[Arg]) toArgs(arg Arg) []any {
	return b
}

func makeStructBinder[Arg any](args []any) (binder[Arg], error) {
	typ := reflect.TypeFor[Arg]()

	isStruct := typ.Kind() == reflect.Struct
	if typ.Kind() == reflect.Ptr {
		isStruct = typ.Elem().Kind() == reflect.Struct
	}

	if !isStruct {
		return structBinder[Arg]{}, errors.New("bind type must be a struct")
	}

	givenArg := make([]any, len(args))
	argPositions := make([]string, len(args))
	for pos, arg := range args {
		if name, ok := arg.(namedArg); ok {
			argPositions[pos] = string(name)
			continue
		}

		givenArg[pos] = arg
	}

	fieldNames := mappings.GetMappings(typ).All
	fieldPositions := make([]int, len(argPositions))

	// check if all positions have matching fields
ArgLoop:
	for argIndex, name := range argPositions {
		if name == "" {
			continue
		}

		for fieldIndex, field := range fieldNames {
			if field == name {
				fieldPositions[argIndex] = fieldIndex
				continue ArgLoop
			}
		}
		return structBinder[Arg]{}, MissingArgError{Name: name}
	}

	return structBinder[Arg]{
		args:     argPositions,
		fields:   fieldPositions,
		givenArg: givenArg,
	}, nil
}

type structBinder[Arg any] struct {
	args     []string
	fields   []int
	givenArg []any
}

func (b structBinder[Arg]) list() []string {
	names := make([]string, len(b.args))
	for _, name := range b.args {
		if name == "" {
			continue
		}

		names = append(names, name)
	}

	return names
}

func (b structBinder[Arg]) toArgs(arg Arg) []any {
	isNil := false
	val := reflect.ValueOf(arg)
	if val.Kind() == reflect.Pointer {
		isNil = val.IsNil()
		val = val.Elem()
	}

	values := make([]any, len(b.args))

	for index, argName := range b.args {
		if argName == "" {
			values[index] = b.givenArg[index]
			continue
		}

		if isNil {
			continue
		}

		values[index] = val.Field(b.fields[index]).Interface()
	}

	return values
}

func makeMapBinder[Arg any](args []any) binder[Arg] {
	givenArg := make([]any, len(args))
	argPositions := make([]string, len(args))
	for pos, arg := range args {
		if name, ok := arg.(namedArg); ok {
			argPositions[pos] = string(name)
			continue
		}

		givenArg[pos] = arg
	}

	return mapBinder[Arg]{
		args:     argPositions,
		givenArg: givenArg,
	}
}

type mapBinder[Arg any] struct {
	args     []string
	givenArg []any
}

func (b mapBinder[Arg]) list() []string {
	names := make([]string, len(b.args))
	for _, name := range b.args {
		if name == "" {
			continue
		}

		names = append(names, name)
	}

	return names
}

func (b mapBinder[Arg]) toArgs(args Arg) []any {
	values := make([]any, len(b.args))

	for index, argName := range b.args {
		if argName == "" {
			values[index] = b.givenArg[index]
			continue
		}

		val := reflect.ValueOf(args).MapIndex(reflect.ValueOf(argName))
		if !val.IsValid() {
			continue
		}

		values[index] = val.Interface()
	}

	return values
}

type singleBinder[Arg any] struct {
	givenArg  []any
	argIndexs []int
	name      string
}

func (b singleBinder[Arg]) list() []string {
	return []string{b.name}
}

func (b singleBinder[Arg]) toArgs(arg Arg) []any {
	values := make([]any, len(b.givenArg))
	copy(values, b.givenArg)

	for _, i := range b.argIndexs {
		values[i] = arg
	}

	return values
}

func countNamedArgs(args []any) int {
	names := map[string]struct{}{}
	for _, arg := range args {
		if name, ok := arg.(namedArg); ok {
			names[string(name)] = struct{}{}
			continue
		}

		if name, ok := arg.(named); ok && len(name.names) == 1 {
			names[name.names[0]] = struct{}{}
			continue
		}
	}

	return len(names)
}
