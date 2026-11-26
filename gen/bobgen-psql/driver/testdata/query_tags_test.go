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

func TestNestedGrouping(t *testing.T) {
	db, err := bob.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx := t.Context()
	_, err = db.ExecContext(ctx, `CREATE TABLE sample (seq INTEGER PRIMARY KEY, id INTEGER, label TEXT, child INTEGER);
		INSERT INTO sample (id, label, child) VALUES
		(3, 'b', 7), (1, 'a', 8), (3, 'a', 9), (3, 'b', 4), (3, 'b', 7),
		(3, NULL, 1), (3, NULL, 2), (1, 'a', NULL), (0, '', NULL), (0, '', 0);`)
	if err != nil {
		t.Fatal(err)
	}

	// Same ID with different labels is two parents. NULL labels never compare
	// equal, and a NULL child from an outer join must not manufacture a child.
	parents, err := GroupedWidgets().All(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(parents)
	const want = `[{"id":3,"label":"b","children":[{"id":7},{"id":4}]},{"id":1,"label":"a","children":[{"id":8}]},{"id":3,"label":"a","children":[{"id":9}]},{"id":3,"label":null,"children":[{"id":1}]},{"id":3,"label":null,"children":[{"id":2}]},{"id":0,"label":"","children":[{"id":0}]}]`
	if err != nil || string(data) != want {
		t.Fatalf("grouped parents: %s, %v; want %s", data, err, want)
	}

	// BLOB parents need byte equality. Equal bytes from different scanned
	// rows must still collect children into the first matching parent.
	byCode, err := GroupedBytes().All(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(byCode)
	const wantBytes = `[{"code":"Yg==","children":[{"id":7},{"id":4}]},{"code":"YQ==","children":[{"id":8},{"id":9}]}]`
	if err != nil || string(data) != wantBytes {
		t.Fatalf("grouped BLOBs: %s, %v; want %s", data, err, wantBytes)
	}
}
