package pgtypes

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"strings"
)

type Snapshot struct {
	Min    uint64
	Max    uint64
	Active []uint64
}

// String returns the string representation of the TSQuery.
// xmin:xmax:xip_list
func (t Snapshot) String() string {
	var s strings.Builder
	fmt.Fprintf(&s, "%d:%d:", t.Min, t.Max)

	if len(t.Active) == 0 {
		return s.String()
	}

	for i, id := range t.Active {
		if i > 0 {
			s.WriteString(",")
		}

		fmt.Fprintf(&s, "%d", id)
	}

	return s.String()
}

// Value implements the driver.Valuer interface.
func (t Snapshot) Value() (driver.Value, error) {
	return t.String(), nil
}

// Scan implements the sql.Scanner interface.
func (t *Snapshot) Scan(src any) error {
	if src == nil {
		return nil
	}

	var val string
	var err error

	switch src := src.(type) {
	case string:
		val = src
	case []byte:
		val = string(src)
	default:
		return fmt.Errorf("cannot scan %T into TSQuery", src)
	}

	parts := strings.Split(val, ":")
	if len(parts) < 2 {
		return fmt.Errorf("invalid snapshot format: %s", val)
	}

	t.Min, err = strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid xmin value: %s", parts[0])
	}

	t.Max, err = strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid xmax value: %s", parts[1])
	}

	if len(parts) < 3 || parts[2] == "" {
		return nil
	}

	xipList := strings.Split(parts[2], ",")
	t.Active = make([]uint64, len(xipList))
	for i, id := range xipList {
		t.Active[i], err = strconv.ParseUint(id, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid xip_list value: %s", id)
		}
	}

	return nil
}
