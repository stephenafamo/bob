package pgtypes

import (
	"database/sql/driver"
	"fmt"
	"net"
)

type Macaddr struct {
	Addr net.HardwareAddr
}

// Scan implements the database/sql Scanner interface.
func (dst *Macaddr) Scan(src any) error {
	if src == nil {
		return fmt.Errorf("cannot scan nil")
	}

	var addr net.HardwareAddr
	var err error

	switch src := src.(type) {
	case string:
		addr, err = net.ParseMAC(src)
	case []byte:
		addr, err = net.ParseMAC(string(src))

	default:
		return fmt.Errorf("cannot scan %T", src)
	}

	if err != nil {
		return err
	}

	*dst = Macaddr{Addr: addr}
	return nil
}

// Value implements the database/sql/driver Valuer interface.
func (src Macaddr) Value() (driver.Value, error) {
	return src.Addr.String(), nil
}
