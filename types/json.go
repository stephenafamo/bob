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
	encodingTextMarshalerIntf   = reflect.TypeFor[encoding.TextMarshaler]()
	encodingTextUnmarshalerIntf = reflect.TypeFor[encoding.TextUnmarshaler]()
)

func NewJSON[T any](val T) JSON[T] {
	return JSON[T]{Val: val}
}

type JSON[T any] struct {
	Val T
}

// Value implements the driver Valuer interface.
func (j JSON[T]) Value() (driver.Value, error) {
	data, err := json.Marshal(j)
	if err != nil {
		return nil, err
	}
	// Return string instead of []byte to avoid pgx treating it as bytea
	// when using the simple protocol with database/sql.
	// This matches the behavior of pgtype.JSON for compatibility.
	// See: https://github.com/jackc/pgtype/issues/45
	return string(data), nil
}

// Scan implements the Scanner interface.
func (j *JSON[T]) Scan(value any) error {
	switch x := value.(type) {
	case string:
		return json.NewDecoder(bytes.NewBuffer([]byte(x))).Decode(j)
	case []byte:
		return json.NewDecoder(bytes.NewBuffer(x)).Decode(j)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan type %T: %v", value, value)
	}
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *JSON[T]) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &j.Val)
}

// MarshalJSON implements json.Marshaler.
func (j JSON[T]) MarshalJSON() ([]byte, error) {
	return json.Marshal(j.Val)
}

// MarshalText implements encoding.TextMarshaler.
func (j JSON[T]) MarshalText() ([]byte, error) {
	refVal := reflect.ValueOf(j.Val)
	if refVal.Type().Implements(encodingTextMarshalerIntf) {
		valuer := refVal.Interface().(encoding.TextMarshaler)
		return valuer.MarshalText()
	}

	var text string
	if err := opt.ConvertAssign(&text, j.Val); err != nil {
		return nil, err
	}
	return []byte(text), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (j *JSON[T]) UnmarshalText(text []byte) error {
	refVal := reflect.ValueOf(&j.Val)
	if refVal.Type().Implements(encodingTextUnmarshalerIntf) {
		valuer := refVal.Interface().(encoding.TextUnmarshaler)
		if err := valuer.UnmarshalText(text); err != nil {
			return err
		}
		return nil
	}

	if err := opt.ConvertAssign(&j.Val, string(text)); err != nil {
		return err
	}

	return nil
}
