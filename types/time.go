package types

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Time is a wrapper around time.Time for drivers that do not properly support time.Time
type Time struct {
	time.Time
}

func (ut *Time) Scan(src any) error {
	if src == nil {
		ut.Time = time.Time{}
		return nil
	}

	switch v := src.(type) {
	case int64:
		ut.Time = time.Unix(v, 0)
		return nil
	case time.Time:
		ut.Time = v
		return nil

	case []byte:
		return ut.parse(string(v))

	case string:
		return ut.parse(v)

	default:
		return fmt.Errorf("unsupported type for Time: %T", src)
	}
}

func (ut Time) Value() (driver.Value, error) {
	return ut.Time.Format(time.RFC3339), nil
}

func (ut *Time) parse(s string) error {
	for _, format := range []string{
		// SQLite formats
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02T15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02T15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02",
		// Go formats
		"2006-01-02 15:04:05.999999999 -0700 MST", // Default Go format
		time.RFC3339Nano,
		time.RFC3339,
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
	} {
		if t, err := time.ParseInLocation(format, s, time.UTC); err == nil {
			ut.Time = t
			return nil
		}
	}
	return fmt.Errorf("cannot parse Time from string: %s", s)
}
