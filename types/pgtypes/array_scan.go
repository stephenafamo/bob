package pgtypes

import (
	"encoding/binary"
	"fmt"
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
)

// arrayMaps holds pgtype.Map instances for scanning arrays.
// A Map is not safe for concurrent use, so each scan takes one from the pool.
// Creating a Map is cheap since it defers to pgx's default type registry.
//
//nolint:gochecknoglobals
var arrayMaps = sync.Pool{New: func() any { return pgtype.NewMap() }}

// textElemType is used for element types that pgx does not know about.
// The text codec hands the element to sql.Scanner implementations as a string.
//
//nolint:gochecknoglobals
var textElemType = &pgtype.Type{Name: "text", OID: pgtype.TextOID, Codec: pgtype.TextCodec{}}

// cardinality returns the number of elements described by the dimensions.
func cardinality(dims []pgtype.ArrayDimension) int {
	if len(dims) == 0 {
		return 0
	}

	n := 1
	for _, d := range dims {
		n *= int(d.Length)
	}

	return n
}

// isBinaryArray reports whether b is a Postgres array in the binary wire format.
// The binary format starts with a big-endian int32 number of dimensions (at most 6),
// so the first byte is always 0. The text format starts with '{' or '['.
func isBinaryArray(b []byte) bool {
	return len(b) >= 12 && b[0] == 0
}

// scanArray decodes src into target using pgx's array codec.
// src may be nil (SQL NULL), a text array literal (string or []byte),
// or the pgx binary wire format ([]byte), which is what pgx hands to
// sql.Scanner implementations when the row is in binary format.
// elemPtr is a pointer to the element type, used to pick the element codec
// when decoding the text format.
func scanArray(src any, target pgtype.ArraySetter, elemPtr any) error {
	var (
		buf    []byte
		format int16
	)

	switch v := src.(type) {
	case nil:
		return target.SetDimensions(nil)
	case string:
		buf, format = []byte(v), pgtype.TextFormatCode
	case []byte:
		buf, format = v, pgtype.TextFormatCode
		if isBinaryArray(v) {
			format = pgtype.BinaryFormatCode
		}
	default:
		return fmt.Errorf("pgtypes: cannot scan %T into a Postgres array", src)
	}

	m := arrayMaps.Get().(*pgtype.Map)
	defer arrayMaps.Put(m)

	var dt *pgtype.Type
	if format == pgtype.BinaryFormatCode {
		elemOID := binary.BigEndian.Uint32(buf[8:12])
		var ok bool
		if dt, ok = m.TypeForOID(elemOID); !ok {
			dt = &pgtype.Type{Name: "unknown", OID: elemOID, Codec: pgtype.TextCodec{}}
		}
	} else {
		var ok bool
		if dt, ok = m.TypeForValue(elemPtr); !ok {
			dt = textElemType
		}
	}

	codec := &pgtype.ArrayCodec{ElementType: dt}
	plan := codec.PlanScan(m, 0, format, target)
	if plan == nil {
		return fmt.Errorf("pgtypes: cannot scan Postgres array with elements of type %s into %T", dt.Name, elemPtr)
	}

	return plan.Scan(buf, target)
}
