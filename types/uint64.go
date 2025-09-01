package types

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

// Uint64 is a custom type for uint64 that scans and saves to the database as a string.
type Uint64 uint64

// Value implements the driver.Valuer interface, converting the Uint64 to a string.
func (u Uint64) Value() (driver.Value, error) {
	return strconv.FormatUint(uint64(u), 10), nil
}

// Scan implements the sql.Scanner interface, converting a string from the database to Uint64.
func (u *Uint64) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	case int64:
		*u = Uint64(v)
		return nil
	case uint64:
		*u = Uint64(v)
		return nil
	case nil:
		return fmt.Errorf("cannot scan nil value into Uint64")
	default:
		return fmt.Errorf("unsupported scan type for Uint64: %T", src)
	}

	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return fmt.Errorf("failed to scan Uint64: %w", err)
	}

	*u = Uint64(val)
	return nil
}
