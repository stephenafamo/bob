package pgtypes

import (
	"database/sql"
	"database/sql/driver"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"sync"
	"testing"

	"github.com/aarondl/opt/null"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stephenafamo/bob/types"
)

// scannerString is an element type that only supports sql.Scanner,
// like decimal.Decimal or uuid.UUID.
type scannerString string

func (s *scannerString) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*s = scannerString(v)
	case []byte:
		*s = scannerString(v)
	default:
		return fmt.Errorf("cannot scan %T", src)
	}
	return nil
}

func (s scannerString) Value() (driver.Value, error) {
	return string(s), nil
}

// encodeBinary encodes v as a Postgres array in the binary wire format,
// which is what pgx hands to sql.Scanner implementations.
func encodeBinary(t *testing.T, oid uint32, v any) []byte {
	t.Helper()

	buf, err := pgtype.NewMap().Encode(oid, pgtype.BinaryFormatCode, v, nil)
	if err != nil {
		t.Fatalf("encoding %v: %v", v, err)
	}

	return buf
}

// encodeBinaryUnknownOID encodes a binary text[] whose element OID pgx does not
// know about, which is how arrays of enums arrive from a native pgx connection.
func encodeBinaryUnknownOID(t *testing.T, v []string) []byte {
	t.Helper()

	const unknownOID = 1_000_000
	buf := encodeBinary(t, pgtype.TextArrayOID, v)
	binary.BigEndian.PutUint32(buf[8:12], unknownOID)

	return buf
}

func testScan[T any](t *testing.T, name string, src any, want Array[T]) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		var got Array[T]
		if err := got.Scan(src); err != nil {
			t.Fatalf("scan: %v", err)
		}

		if (got == nil) != (want == nil) {
			t.Fatalf("nil mismatch: got %#v, want %#v", got, want)
		}

		if !reflect.DeepEqual(got, want) {
			t.Fatalf("got %#v, want %#v", got, want)
		}
	})
}

func TestArrayScanText(t *testing.T) {
	testScan(t, "nil", nil, Array[int32](nil))
	testScan(t, "empty", "{}", Array[int32]{})
	testScan(t, "empty bytes", []byte("{}"), Array[int32]{})
	testScan(t, "int32", "{1,2,3}", Array[int32]{1, 2, 3})
	testScan(t, "int32 bytes", []byte("{1,2,3}"), Array[int32]{1, 2, 3})
	testScan(t, "int64", "{1,-2}", Array[int64]{1, -2})
	testScan(t, "int16", "{1,2}", Array[int16]{1, 2})
	testScan(t, "float32", "{1.5,2}", Array[float32]{1.5, 2})
	testScan(t, "float64", "{1.5,2.25}", Array[float64]{1.5, 2.25})
	testScan(t, "bool", "{t,f}", Array[bool]{true, false})
	testScan(t, "string", `{"a\"b","c,d","",NULL}`, Array[*string]{ptr(`a"b`), ptr("c,d"), ptr(""), nil})
	testScan(t, "string unquoted", "{a,b}", Array[string]{"a", "b"})
	testScan(t, "explicit bounds", "[1:2]={1,2}", Array[int32]{1, 2})
	testScan(t, "multi dimensional flattened", "{{1,2},{3,4}}", Array[int32]{1, 2, 3, 4})
	testScan(t, "bytea", `{"\\x0102","\\x"}`, Array[[]byte]{{1, 2}, {}})
	testScan(t, "scanner", `{"1.5","2"}`, Array[scannerString]{"1.5", "2"})
	testScan(t, "json", `{"{\"a\":1}","[1]"}`, Array[types.JSON[json.RawMessage]]{
		{Val: json.RawMessage(`{"a":1}`)},
		{Val: json.RawMessage(`[1]`)},
	})
	testScan(t, "null val", "{1,NULL}", Array[null.Val[int32]]{null.From[int32](1), null.FromPtr[int32](nil)})
	testScan(t, "sql null", "{1,NULL}", Array[sql.Null[int32]]{{V: 1, Valid: true}, {}})
}

func TestArrayScanBinary(t *testing.T) {
	testScan(t, "int32", encodeBinary(t, pgtype.Int4ArrayOID, []int32{1, 2, 3}), Array[int32]{1, 2, 3})
	testScan(t, "int64", encodeBinary(t, pgtype.Int8ArrayOID, []int64{1, -2}), Array[int64]{1, -2})
	testScan(t, "float64", encodeBinary(t, pgtype.Float8ArrayOID, []float64{1.5, 2.25}), Array[float64]{1.5, 2.25})
	testScan(t, "bool", encodeBinary(t, pgtype.BoolArrayOID, []bool{true, false}), Array[bool]{true, false})
	testScan(t, "string", encodeBinary(t, pgtype.TextArrayOID, []string{`a"b`, "c,d", ""}), Array[string]{`a"b`, "c,d", ""})
	testScan(t, "string with null", encodeBinary(t, pgtype.TextArrayOID, []*string{ptr("a"), nil}), Array[*string]{ptr("a"), nil})
	testScan(t, "empty", encodeBinary(t, pgtype.Int4ArrayOID, []int32{}), Array[int32]{})
	testScan(t, "bytea", encodeBinary(t, pgtype.ByteaArrayOID, [][]byte{{1, 2}, {}}), Array[[]byte]{{1, 2}, {}})
	testScan(t, "scanner", encodeBinary(t, pgtype.TextArrayOID, []string{"1.5", "2"}), Array[scannerString]{"1.5", "2"})
	testScan(t, "json", encodeBinary(t, pgtype.JSONBArrayOID, []json.RawMessage{[]byte(`{"a":1}`)}), Array[types.JSON[json.RawMessage]]{
		{Val: json.RawMessage(`{"a":1}`)},
	})
	testScan(t, "null val", encodeBinary(t, pgtype.Int4ArrayOID, []*int32{ptr[int32](1), nil}), Array[null.Val[int32]]{null.From[int32](1), null.FromPtr[int32](nil)})
}

func TestArrayScanUnknownBinaryOID(t *testing.T) {
	buf := encodeBinaryUnknownOID(t, []string{"hello", "привет"})

	var got Array[string]
	if err := got.Scan(buf); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, Array[string]{"hello", "привет"}) {
		t.Fatalf("got %#v", got)
	}
}

func TestArrayScanErrors(t *testing.T) {
	var ints Array[int32]
	if err := ints.Scan("{1,NULL}"); err == nil {
		t.Error("expected error scanning NULL element into int32")
	}
	if err := ints.Scan(encodeBinary(t, pgtype.Int4ArrayOID, []*int32{nil})); err == nil {
		t.Error("expected error scanning binary NULL element into int32")
	}
	if err := ints.Scan(42); err == nil {
		t.Error("expected error scanning int")
	}
	if err := ints.Scan("not an array"); err == nil {
		t.Error("expected error scanning invalid text")
	}
}

// The shapes from https://github.com/stephenafamo/bob/issues/739
func TestArrayScanWrapped(t *testing.T) {
	binary := encodeBinary(t, pgtype.Int4ArrayOID, []int32{1, 2})

	var nullVal null.Val[Array[int32]]
	if err := nullVal.Scan(binary); err != nil {
		t.Fatalf("null.Val: %v", err)
	}
	if got := nullVal.MustGet(); !slices.Equal(got, Array[int32]{1, 2}) {
		t.Fatalf("null.Val: got %#v", got)
	}

	var sqlNull sql.Null[Array[int32]]
	if err := sqlNull.Scan(binary); err != nil {
		t.Fatalf("sql.Null: %v", err)
	}
	if !sqlNull.Valid || !slices.Equal(sqlNull.V, Array[int32]{1, 2}) {
		t.Fatalf("sql.Null: got %#v", sqlNull)
	}

	if err := nullVal.Scan(nil); err != nil {
		t.Fatalf("null.Val nil: %v", err)
	}
	if nullVal.IsValue() {
		t.Fatal("null.Val: expected null")
	}
}

func TestArrayValue(t *testing.T) {
	cases := []struct {
		name string
		val  driver.Valuer
		want driver.Value
	}{
		{"nil", Array[int32](nil), nil},
		{"empty", Array[int32]{}, "{}"},
		{"int32", Array[int32]{1, 2}, "{1,2}"},
		{"bool", Array[bool]{true, false}, "{t,f}"},
		{"string", Array[string]{`a"b`, "c,d", ""}, `{"a\"b","c,d",""}`},
		{"bytea", Array[[]byte]{{1, 2}, {}}, `{"\\x0102","\\x"}`},
		{"valuer", Array[scannerString]{"1.5"}, `{"1.5"}`},
		{"null val", Array[null.Val[int32]]{null.From[int32](1), null.FromPtr[int32](nil)}, "{1,NULL}"},
		{"wrapped", null.From(Array[int32]{1}), "{1}"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.val.Value()
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %#v, want %#v", got, tc.want)
			}
		})
	}
}

// pgx uses ArrayGetter/ArraySetter directly, without going through Scan/Value
func TestArrayPgxNative(t *testing.T) {
	m := pgtype.NewMap()

	for _, format := range []int16{pgtype.TextFormatCode, pgtype.BinaryFormatCode} {
		buf, err := m.Encode(pgtype.TextArrayOID, format, Array[string]{`a"b`, ""}, nil)
		if err != nil {
			t.Fatalf("encode format %d: %v", format, err)
		}

		var got Array[string]
		if err := m.Scan(pgtype.TextArrayOID, format, buf, &got); err != nil {
			t.Fatalf("scan format %d: %v", format, err)
		}
		if !slices.Equal(got, Array[string]{`a"b`, ""}) {
			t.Fatalf("format %d: got %#v", format, got)
		}

		var wrapped null.Val[Array[string]]
		if err := m.Scan(pgtype.TextArrayOID, format, buf, &wrapped); err != nil {
			t.Fatalf("scan wrapped format %d: %v", format, err)
		}
		if got := wrapped.MustGet(); !slices.Equal(got, Array[string]{`a"b`, ""}) {
			t.Fatalf("wrapped format %d: got %#v", format, got)
		}

		var nilArr Array[string]
		buf, err = m.Encode(pgtype.TextArrayOID, format, nilArr, nil)
		if err != nil {
			t.Fatalf("encode nil format %d: %v", format, err)
		}
		if buf != nil {
			t.Fatalf("nil array should encode as NULL, got %v", buf)
		}
	}
}

// A pgtype.Map is not safe for concurrent use; scanning must not share one
func TestArrayScanConcurrent(t *testing.T) {
	bin := encodeBinary(t, pgtype.Int4ArrayOID, []int32{1, 2, 3})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				var a Array[int32]
				if err := a.Scan(bin); err != nil {
					t.Error(err)
				}
				var s Array[string]
				if err := s.Scan("{a,b}"); err != nil {
					t.Error(err)
				}
				var f Array[float64]
				if err := f.Scan(`{1.5}`); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	wg.Wait()
}

func ptr[T any](v T) *T {
	return &v
}
