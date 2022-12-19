package types

import (
	"bytes"
	"database/sql/driver"
	"encoding"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/aarondl/opt"
)

//nolint:gochecknoglobals
var (
	encodingTextMarshalerIntf   = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
	encodingTextUnmarshalerIntf = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

type JSON[T any] struct {
	value T
}

func (j JSON[T]) Get() T {
	return j.value
}

// Value implements the driver Valuer interface.
func (j JSON[T]) Value() (driver.Value, error) {
	return json.Marshal(j)
}

// Scan implements the Scanner interface.
func (j *JSON[T]) Scan(value interface{}) error {
	var err error

	switch x := value.(type) {
	case string:
		err = json.NewDecoder(bytes.NewBuffer([]byte(x))).Decode(j)
	case []byte:
		err = json.NewDecoder(bytes.NewBuffer(x)).Decode(j)
	case nil:
		return nil

	default:
		err = fmt.Errorf("cannot scan type %T: %v", value, value)
	}

	return err
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSON[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.value)
}

// MarshalJSON implements json.Marshaler.
func (j JSON[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.value)
}

// MarshalText implements encoding.TextMarshaler.
func (j JSON[T]) MarshalText() ([]byte, error) {
	refVal := reflect.ValueOf(j.value)
	if refVal.Type().Implements(encodingTextMarshalerIntf) {
		valuer := refVal.Interface().(encoding.TextMarshaler)
		return valuer.MarshalText()
	}

	var text string
	if err := opt.ConvertAssign(&text, j.value); err != nil {
		return nil, err
	}
	return []byte(text), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (j *JSON[T]) UnmarshalText(text []byte) error {
	refVal := reflect.ValueOf(&j.value)
	if refVal.Type().Implements(encodingTextUnmarshalerIntf) {
		valuer := refVal.Interface().(encoding.TextUnmarshaler)
		if err := valuer.UnmarshalText(text); err != nil {
			return err
		}
		return nil
	}

	if err := opt.ConvertAssign(&j.value, string(text)); err != nil {
		return err
	}

	return nil
}
