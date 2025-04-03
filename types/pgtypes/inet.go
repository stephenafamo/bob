package pgtypes

import (
	"database/sql/driver"
	"fmt"
	"net/netip"
	"strings"
)

type Inet struct {
	netip.Prefix
}

// Scan implements the sql.Scanner interface.
func (i *Inet) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case []byte:
		s = string(v)
	case string:
		s = v
	default:
		return fmt.Errorf("cannot scan type %T: %v", src, src)
	}

	if strings.Contains(s, "/") {
		return i.Prefix.UnmarshalText([]byte(s))
	}

	addr, err := netip.ParseAddr(s)
	if err != nil {
		return err
	}

	i.Prefix = netip.PrefixFrom(addr, addr.BitLen())

	return nil
}

// Value implements the driver.Valuer interface.
func (i Inet) Value() (driver.Value, error) {
	return i.Prefix.MarshalText()
}
