package pgtypes

import (
	"database/sql/driver"
	"fmt"
)

type LSN uint64

// Scan implements the sql.Scanner interface.
func (lsn *LSN) Scan(src interface{}) error {
	if src == nil {
		return fmt.Errorf("cannot scan nil")
	}

	var s string

	switch src := src.(type) {
	case string:
		s = src
	case []byte:
		s = string(src)

	default:
		return fmt.Errorf("cannot scan %T", src)
	}

	var hi, lo uint32
	n, err := fmt.Sscanf(s, "%X/%X", &hi, &lo)
	if err != nil {
		return fmt.Errorf(`scanning hi/lo: %v: %w`, src, err)
	}

	if n != 2 {
		return fmt.Errorf("invalid pg_lsn value")
	}

	*lsn = LSN(uint64(hi)<<32 | uint64(lo))

	return nil
}

// Value implements the driver.Valuer interface.
func (lsn LSN) Value() (driver.Value, error) {
	return fmt.Sprintf("%X/%X", lsn>>32, uint32(lsn)), nil
}
