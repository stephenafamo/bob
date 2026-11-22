package queries

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/stephenafamo/bob"
	_ "modernc.org/sqlite"
)

func TestSerializedRows(t *testing.T) {
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	flat, err := GetWidget().One(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	nested, err := GetNestedWidgets().All(t.Context(), db)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		value any
		want  string
	}{
		{flat, `{"widget_id":11,"widget_id2":22,"display_name":"sample"}`},
		{nested, `[{"related_widgets":7,"related_widgets2":[{"widget_id":42}]}]`},
	} {
		data, err := json.Marshal(tc.value)
		if err != nil || string(data) != tc.want {
			t.Errorf("Marshal(%T) = %s, %v; want %s", tc.value, data, err, tc.want)
		}
	}

	field, ok := reflect.TypeOf(flat).FieldByName("WidgetID2")
	if !ok || field.Tag.Get("db") != "widget_id" || field.Tag.Get("yaml") != "widget_id2" {
		t.Errorf("unexpected tags on second widget ID: %q", field.Tag)
	}
}
