package scanto

import (
	"database/sql"
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

// Uses reflection to create a mapping function for a struct type
// using the default options
func StructMapper[T any](c cols) func(*Values) (T, error) {
	var x T
	typ := reflect.TypeOf(x)
	m, err := defaultStructMapper.getMapping(reflect.TypeOf(x))
	if err != nil {
		return ErrorMapper[T](err)
	}

	return mapperFromMapping[T](m, typ, false)(c)
}

// Uses reflection to create a mapping function for a struct type
// using with custom options
func CustomStructMapper[T any](opts ...MappingOption) func(c cols) func(*Values) (T, error) {
	mapper, err := NewStructMapper(opts...)
	return func(c cols) func(*Values) (T, error) {
		if err != nil {
			return ErrorMapper[T](err)
		}

		var x T
		typ := reflect.TypeOf(x)
		m, err := mapper.getMapping(reflect.TypeOf(x))
		if err != nil {
			return ErrorMapper[T](err)
		}

		return mapperFromMapping[T](m, typ, mapper.allowUnknownColumns)(c)
	}
}

func mapperFromMapping[T any](m mapping, typ reflect.Type, allowUnknown bool) func(cols) func(*Values) (T, error) {
	var isPointer bool
	if typ.Kind() == reflect.Pointer {
		isPointer = true
		typ = typ.Elem()
	}

	return func(c cols) func(*Values) (T, error) {
		// Filter the mapping so we only ask for the available columns
		filtered := make(mapping)
		for name := range c {
			v, ok := m[name]
			if !ok {
				if !allowUnknown {
					err := fmt.Errorf("No destination for column %q", name)
					return ErrorMapper[T](err, "no destination", name)
				}
				continue
			}

			filtered[name] = v
		}

		return func(v *Values) (T, error) {
			row := reflect.New(typ).Elem()

			for name, info := range filtered {
				for _, v := range info.init {
					pv := row.FieldByIndex(v)
					if !pv.IsZero() {
						continue
					}

					pv.Set(reflect.New(pv.Type().Elem()))
				}

				fv := row.FieldByIndex(info.position)
				val := GetType(v, name, fv.Type())
				fv.Set(reflect.ValueOf(val))
			}

			if isPointer {
				row = row.Addr()
			}

			return row.Interface().(T), nil
		}
	}
}

// NameMapperFunc is a function type that maps a struct field name to the database column name.
type NameMapperFunc func(string) string

var (
	matchFirstCapRe = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllCapRe   = regexp.MustCompile("([a-z0-9])([A-Z])")
)

// SnakeCaseFieldFunc is a NameMapperFunc that maps struct field to snake case.
func SnakeCaseFieldFunc(str string) string {
	snake := matchFirstCapRe.ReplaceAllString(str, "${1}_${2}")
	snake = matchAllCapRe.ReplaceAllString(snake, "${1}_${2}")
	return strings.ToLower(snake)
}

// MappingOption is a function type that changes Mapping configuration.
type MappingOption func(api *structMapper) error

// NewStructMapper creates a new Mapping object with provided list of options.
func NewStructMapper(opts ...MappingOption) (*structMapper, error) {
	api := defaultStructMapper
	for _, o := range opts {
		if err := o(&api); err != nil {
			return nil, err
		}
	}
	return &api, nil
}

// WithStructTagKey allows to use a custom struct tag key.
// The default tag key is `db`.
func WithStructTagKey(tagKey string) MappingOption {
	return func(api *structMapper) error {
		api.structTagKey = tagKey
		return nil
	}
}

// WithColumnSeparator allows to use a custom separator character for column name when combining nested structs.
// The default separator is "." character.
func WithColumnSeparator(separator string) MappingOption {
	return func(api *structMapper) error {
		api.columnSeparator = separator
		return nil
	}
}

// WithFieldNameMapper allows to use a custom function to map field name to column names.
// The default function is SnakeCaseFieldFunc.
func WithFieldNameMapper(mapperFn NameMapperFunc) MappingOption {
	return func(api *structMapper) error {
		api.fieldMapperFn = mapperFn
		return nil
	}
}

// WithScannableTypes specifies a list of interfaces that underlying database library can scan into.
// In case the destination type passed to dbscan implements one of those interfaces,
// dbscan will handle it as primitive type case i.e. simply pass the destination to the database library.
// Instead of attempting to map database columns to destination struct fields or map keys.
// In order for reflection to capture the interface type, you must pass it by pointer.
//
// For example your database library defines a scanner interface like this:
// type Scanner interface {
//     Scan(...) error
// }
// You can pass it to dbscan this way:
// dbscan.WithScannableTypes((*Scanner)(nil)).
func WithScannableTypes(scannableTypes ...interface{}) MappingOption {
	return func(api *structMapper) error {
		for _, stOpt := range scannableTypes {
			st := reflect.TypeOf(stOpt)
			if st == nil {
				return fmt.Errorf("scannable type must be a pointer, got %T", st)
			}
			if st.Kind() != reflect.Ptr {
				return fmt.Errorf("scannable type must be a pointer, got %s: %s",
					st.Kind(), st.String())
			}
			st = st.Elem()
			if st.Kind() != reflect.Interface {
				return fmt.Errorf("scannable type must be a pointer to an interface, got %s: %s",
					st.Kind(), st.String())
			}
			api.scannableTypes = append(api.scannableTypes, st)
		}
		return nil
	}
}

// WithAllowUnknownColumns allows the scanner to ignore db columns that doesn't exist at the destination.
// The default function is to throw an error when a db column ain't found at the destination.
func WithAllowUnknownColumns(allowUnknownColumns bool) MappingOption {
	return func(api *structMapper) error {
		api.allowUnknownColumns = allowUnknownColumns
		return nil
	}
}

// structMapper is the core type in dbscan. It implements all the logic and exposes functionality available in the package.
// With structMapper type users can create a custom structMapper instance and override default settings hence configure dbscan.
type structMapper struct {
	structTagKey        string
	columnSeparator     string
	fieldMapperFn       NameMapperFunc
	scannableTypes      []reflect.Type
	allowUnknownColumns bool
	maxDepth            int
}

func (s structMapper) getMapping(typ reflect.Type) (mapping, error) {
	if typ == nil {
		return nil, fmt.Errorf("Nil type passed to StructMapper")
	}

	var structTyp reflect.Type

	switch {
	case typ.Kind() == reflect.Struct:
		structTyp = typ
	case typ.Kind() == reflect.Pointer:
		structTyp = typ.Elem()

		if structTyp.Kind() != reflect.Struct {
			return nil, fmt.Errorf("Type %q is not a struct or pointer to a struct", typ.String())
		}
	default:
		return nil, fmt.Errorf("Type %q is not a struct or pointer to a struct", typ.String())
	}

	m := make(mapping)
	s.setMappings(structTyp, "", make(visited), m, nil)

	return m, nil
}

func (s structMapper) setMappings(typ reflect.Type, prefix string, v visited, m mapping, inits [][]int, position ...int) {
	count := v[typ]
	if count > s.maxDepth {
		return
	}
	v[typ] = count + 1

	var hasExported bool

	// If it implements a scannable type, then it can be used
	// as a value itself. Return it
	for _, scannable := range s.scannableTypes {
		if reflect.PtrTo(typ).Implements(scannable) {
			m[prefix] = mapinfo{
				position: position,
				init:     inits,
			}
			return
		}
	}

	// Go through the struct fields and populate the map.
	// Recursively go into any child structs, adding a prefix where necessary
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)

		// Don't consider unexported fields
		if !field.IsExported() {
			continue
		}

		// Skip columns that have the tag "-"
		tag := field.Tag.Get(s.structTagKey)
		if tag == "-" {
			continue
		}

		hasExported = true

		key := prefix

		if !field.Anonymous {
			var sep string
			if prefix != "" {
				sep = s.columnSeparator
			}

			name := field.Name
			if tag != "" {
				name = tag
			}

			key = strings.Join([]string{key, s.fieldMapperFn(name)}, sep)
		}

		currentIndex := append(position, i)
		fieldType := field.Type
		if fieldType.Kind() == reflect.Pointer {
			inits = append(inits, currentIndex)
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			s.setMappings(fieldType, key, v.copy(), m, inits, currentIndex...)
			continue
		}

		m[key] = mapinfo{
			position: currentIndex,
			init:     inits,
		}
	}

	// If it has no exported field (such as time.Time) then we attempt to
	// directly scan into it
	if !hasExported {
		m[prefix] = mapinfo{
			position: position,
			init:     inits,
		}
	}
}

//nolint:gochecknoglobals
var defaultStructMapper = structMapper{
	structTagKey:        "db",
	columnSeparator:     ".",
	fieldMapperFn:       SnakeCaseFieldFunc,
	scannableTypes:      []reflect.Type{reflect.TypeOf((*sql.Scanner)(nil)).Elem()},
	allowUnknownColumns: false,
	maxDepth:            3,
}
