package drivers

import (
	"reflect"
	"slices"
	"testing"
)

func TestQueryResultTagNames(t *testing.T) {
	// The duplicate must not take ID2 from the explicitly named column.
	// URLValue and UrlValue are distinct in Go but collide in snake and camel case.
	cols := QueryCols{{Name: "id"}, {Name: "id"}, {Name: "id2"}, {Name: "URLValue"}, {Name: "UrlValue"}}.WithNames()
	wantFields := []string{"ID", "ID3", "ID2", "URLValue", "UrlValue"}
	for i, col := range cols {
		if col.Name != wantFields[i] {
			t.Fatalf("field %d: got %q, want %q", i, col.Name, wantFields[i])
		}
	}
	for _, tc := range []struct {
		casing string
		want   []string
	}{
		{"snake", []string{"id", "id3", "id2", "url_value", "url_value_2"}},
		{"camel", []string{"id", "id3", "id2", "urlValue", "urlValue2"}},
		{"title", wantFields},
		{"alias", wantFields},
	} {
		t.Run(tc.casing, func(t *testing.T) {
			tags := QueryResultTags{Casing: tc.casing, Keys: []string{"json"}}
			var got []string
			for _, tag := range tags.FieldTags(cols) {
				got = append(got, reflect.StructTag(tag).Get("json"))
			}
			if !slices.Equal(got, tc.want) {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
