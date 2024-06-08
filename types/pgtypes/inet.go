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
	switch v := src.(type) {
	case []byte:
		return i.Prefix.UnmarshalBinary(v)
	case string:
		if strings.Contains(v, "/") {
			return i.Prefix.UnmarshalText([]byte(v))
		}
		addr, err := netip.ParseAddr(v)
		if err != nil {
			return err
		}
		i.Prefix = netip.PrefixFrom(addr, addr.BitLen())
		return nil
	default:
		return fmt.Errorf("cannot scan type %T: %v", src, src)
	}
}

// Value implements the driver.Valuer interface.
func (i Inet) Value() (driver.Value, error) {
	return i.Prefix.MarshalText()
}
