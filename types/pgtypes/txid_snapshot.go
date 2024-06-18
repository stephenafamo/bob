package pgtypes

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

type TxIDSnapshot struct {
	Min    string
	Max    string
	Active []string
}

// String returns the string representation of the TxIDSnapshot.
func (s TxIDSnapshot) String() string {
	return fmt.Sprintf("%s:%s:%s", s.Min, s.Max, strings.Join(s.Active, ","))
}

// Value implements the driver.Valuer interface.
func (s TxIDSnapshot) Value() (driver.Value, error) {
	return s.String(), nil
}

// Scan implements the sql.Scanner interface.
func (s *TxIDSnapshot) Scan(src any) error {
	if src == nil {
		return fmt.Errorf("cannot scan nil")
	}

	var val string

	switch src := src.(type) {
	case string:
		val = src
	case []byte:
		val = string(src)

	default:
		return fmt.Errorf("cannot scan %T into TxIDSnapshot", src)
	}

	parts := strings.Split(val, ",")
	if len(parts) != 3 {
		return fmt.Errorf("invalid txid_snapshot value")
	}

	*s = TxIDSnapshot{
		Min:    parts[0],
		Max:    parts[1],
		Active: strings.Split(parts[2], ","),
	}

	return nil
}
