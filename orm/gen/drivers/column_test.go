package drivers

import (
	"reflect"
	"strings"
	"testing"
)

func TestColumnNames(t *testing.T) {
	t.Parallel()

	cols := []Column{
		{Name: "one"},
		{Name: "two"},
		{Name: "three"},
	}

	out := strings.Join(ColumnNames(cols), " ")
	if out != "one two three" {
		t.Error("output was wrong:", out)
	}
}

func TestColumnDBTypes(t *testing.T) {
	cols := []Column{
		{Name: "test_one", DBType: "integer"},
		{Name: "test_two", DBType: "interval"},
	}

	res := ColumnDBTypes(cols)
	if res["TestOne"] != "integer" {
		t.Errorf(`Expected res["TestOne"]="integer", got: %s`, res["TestOne"])
	}
	if res["TestTwo"] != "interval" {
		t.Errorf(`Expected res["TestOne"]="interval", got: %s`, res["TestOne"])
	}
}

func TestColumnsFromList(t *testing.T) {
	t.Parallel()

	if ColumnsFromList(nil, "table") != nil {
		t.Error("expected a shortcut to getting nil back")
	}

	if got := ColumnsFromList([]string{"a.b", "b", "c.d", "c.a"}, "c"); !reflect.DeepEqual(got, []string{"d", "a"}) {
		t.Error("list was wrong:", got)
	}
	if got := ColumnsFromList([]string{"a.b", "b", "c.d", "c.a"}, "b"); len(got) != 0 {
		t.Error("list was wrong:", got)
	}
	if got := ColumnsFromList([]string{"*.b", "b", "c.d"}, "c"); !reflect.DeepEqual(got, []string{"b", "d"}) {
		t.Error("list was wrong:", got)
	}
}
