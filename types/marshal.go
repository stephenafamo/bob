package types

import (
	"database/sql/driver"
	"encoding"
	"fmt"
)

type Text[T interface {
	encoding.TextMarshaler
}, Tp interface {
	*T
	encoding.TextUnmarshaler
}] struct {
	Val T
}

func (t Text[T, Tp]) Value() (driver.Value, error) {
	return t.Val.MarshalText()
}

func (t *Text[T, Tp]) Scan(value any) error {
	switch x := value.(type) {
	case string:
		v := Tp(&t.Val)
		return v.UnmarshalText([]byte(x))
	case []byte:
		v := Tp(&t.Val)
		return v.UnmarshalText(x)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan type %T: %v", value, value)
	}
}

type Binary[T interface {
	encoding.BinaryMarshaler
}, Tp interface {
	*T
	encoding.BinaryUnmarshaler
}] struct {
	Val T
}

func (b Binary[T, Tp]) Value() (driver.Value, error) {
	return b.Val.MarshalBinary()
}

func (b *Binary[T, Tp]) Scan(value any) error {
	switch x := value.(type) {
	case string:
		v := Tp(&b.Val)
		return v.UnmarshalBinary([]byte(x))
	case []byte:
		v := Tp(&b.Val)
		return v.UnmarshalBinary(x)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan type %T: %v", value, value)
	}
}
