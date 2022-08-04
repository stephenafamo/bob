package scanto

import (
	"reflect"
)

func Get[T any](v *Values, name string) T {
	if v.recording {
		var x T
		v.record(name, reflect.TypeOf(x))
	}

	return v.get(name).(T)
}

// Non-generic version, can be used with reflection
func GetType(v *Values, name string, typ reflect.Type) any {
	if v.recording {
		v.record(name, typ)
	}

	return v.get(name)
}

func newValues(r Rows) (*Values, error) {
	cols, err := r.Columns()
	if err != nil {
		return nil, err
	}

	// convert columns to a map
	colMap := make(map[string]int, len(cols))
	for k, v := range cols {
		colMap[v] = k
	}

	return &Values{
		columns: colMap,
		types:   make(map[string]reflect.Type, len(cols)),
	}, nil
}

// Holds the values of a row
// use Get() to retrieve a value
// if multiple columns have the same name, only the last one remains
// so column names should be unique
type Values struct {
	columns   map[string]int
	recording bool
	types     map[string]reflect.Type
	scanned   []any
}

// IsRecording returns wether the values are currently in recording mode
// When recording, calls to Get() will record the expected type
func (v *Values) IsRecording() bool {
	return v.recording
}

// To get a copy of the columns to pass to mapper generators
// since modifing the map can have unintended side effects.
// Ideally, a generator should only call this once
func (v *Values) columnsCopy() map[string]int {
	m := make(map[string]int, len(v.columns))
	for k, v := range v.columns {
		m[k] = v
	}
	return m
}

func (v *Values) get(name string) any {
	index, ok := v.columns[name]
	if !ok || v.recording {
		x := reflect.New(v.types[name]).Elem().Interface()
		return x
	}

	return reflect.Indirect(
		reflect.ValueOf(v.scanned[index]),
	).Interface()
}

func (v *Values) record(name string, t reflect.Type) {
	v.types[name] = t
}

func (v *Values) scanRow(r Row) error {
	pointers := make([]any, len(v.columns))

	for name, i := range v.columns {
		t := v.types[name]
		if t == nil {
			var fallback interface{}
			pointers[i] = &fallback

			continue
		}

		pointers[i] = reflect.New(t).Interface()
	}

	err := r.Scan(pointers...)
	if err != nil {
		return err
	}

	v.scanned = pointers
	return nil
}
