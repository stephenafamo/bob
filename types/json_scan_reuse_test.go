package types

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestJSONScanReusedDestination(t *testing.T) {
	type nested struct {
		Name string   `json:"name"`
		Tags []string `json:"tags"`
	}
	type record struct {
		Nested *nested `json:"nested"`
	}
	var destination JSON[record]
	if err := destination.Scan([]byte(`{"nested":{"name":"First","tags":["a","b"]}}`)); err != nil {
		t.Fatal(err)
	}
	previous := destination.Val
	if err := destination.Scan(`{"nested":{"name":"Second","tags":["c"]}}`); err != nil {
		t.Fatal(err)
	}
	want := record{Nested: &nested{Name: "First", Tags: []string{"a", "b"}}}
	if diff := cmp.Diff(want, previous); diff != "" {
		t.Fatalf("previous row changed (-want +got):\n%s", diff)
	}
	if err := destination.Scan([]byte(`{}`)); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(record{}, destination.Val); diff != "" {
		t.Fatalf("absent fields retained (-want +got):\n%s", diff)
	}
	if err := destination.Scan([]byte(`{"nested":{"name":"First"}}`)); err != nil {
		t.Fatal(err)
	}
	previous = destination.Val
	if err := destination.Scan([]byte(`{"nested":{"name":"Wrong","tags":123}}`)); err == nil {
		t.Fatal("expected invalid JSON field error")
	}
	if diff := cmp.Diff(previous, destination.Val); diff != "" {
		t.Fatalf("failed scan changed destination (-want +got):\n%s", diff)
	}
	if destination.Val.Nested.Name != "First" {
		t.Fatal("failed scan mutated previous row")
	}
	if err := destination.Scan(nil); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(record{}, destination.Val); diff != "" {
		t.Fatalf("nil scan retained fields (-want +got):\n%s", diff)
	}
}
