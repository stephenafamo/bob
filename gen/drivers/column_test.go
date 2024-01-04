package drivers

import (
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
