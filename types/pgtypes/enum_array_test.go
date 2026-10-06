package pgtypes

import (
	"slices"
	"testing"

	"github.com/aarondl/opt/null"
	"github.com/jackc/pgx/v5/pgtype"
)

type testEnum string

func TestEnumArrayScan(t *testing.T) {
	want := EnumArray[testEnum]{"hello", "привет", `a"b`}

	for name, src := range map[string]any{
		"text":   `{hello,привет,"a\"b"}`,
		"bytes":  []byte(`{hello,привет,"a\"b"}`),
		"binary": encodeBinaryUnknownOID(t, []string{"hello", "привет", `a"b`}),
	} {
		var got EnumArray[testEnum]
		if err := got.Scan(src); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("%s: got %#v", name, got)
		}

		var wrapped null.Val[EnumArray[testEnum]]
		if err := wrapped.Scan(src); err != nil {
			t.Fatalf("%s wrapped: %v", name, err)
		}
		if got := wrapped.MustGet(); !slices.Equal(got, want) {
			t.Fatalf("%s wrapped: got %#v", name, got)
		}
	}

	var got EnumArray[testEnum]
	if err := got.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if got != nil {
		t.Fatalf("expected nil, got %#v", got)
	}

	if err := got.Scan("{}"); err != nil {
		t.Fatal(err)
	}
	if got == nil || len(got) != 0 {
		t.Fatalf("expected empty, got %#v", got)
	}
}

func TestEnumArrayValue(t *testing.T) {
	if v, err := EnumArray[testEnum](nil).Value(); err != nil || v != nil {
		t.Fatalf("nil: got %#v, %v", v, err)
	}
	if v, err := (EnumArray[testEnum]{}).Value(); err != nil || v != "{}" {
		t.Fatalf("empty: got %#v, %v", v, err)
	}
	if v, err := (EnumArray[testEnum]{"a", `b"c`}).Value(); err != nil || v != `{"a","b\"c"}` {
		t.Fatalf("values: got %#v, %v", v, err)
	}
}

func TestEnumArrayPgxNative(t *testing.T) {
	m := pgtype.NewMap()
	want := EnumArray[testEnum]{"a", "b"}

	for _, format := range []int16{pgtype.TextFormatCode, pgtype.BinaryFormatCode} {
		buf, err := m.Encode(pgtype.TextArrayOID, format, want, nil)
		if err != nil {
			t.Fatalf("encode format %d: %v", format, err)
		}

		var got EnumArray[testEnum]
		if err := m.Scan(pgtype.TextArrayOID, format, buf, &got); err != nil {
			t.Fatalf("scan format %d: %v", format, err)
		}
		if !slices.Equal(got, want) {
			t.Fatalf("format %d: got %#v", format, got)
		}
	}
}
