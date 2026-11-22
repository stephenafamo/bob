package parser

import (
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func TestUpdateArgumentNullability(t *testing.T) {
	p := New(tables{{Key: "flags", Name: "flags", Columns: []drivers.Column{
		{Name: "enabled", DBType: "tinyint(1)", Nullable: true},
		{Name: "id", DBType: "int"},
	}}})
	for _, tc := range []struct {
		annotation string
		nullable   bool
	}{
		{"", true},
		{" /* ::notnull */", false},
	} {
		q, err := p.ParseQueries(t.Context(), "-- UpdateFlag\nUPDATE flags SET enabled = ?"+tc.annotation+" WHERE id = ?;")
		if err != nil {
			t.Fatal(err)
		}
		args := q[0].Args
		if len(args) != 2 {
			t.Fatalf("expected two arguments, got %v", args)
		}
		set, where := args[0].Col, args[1].Col
		if set.TypeName != "bool" || set.Nullable == nil || *set.Nullable != tc.nullable {
			t.Errorf("SET argument with annotation %q: got %+v, want bool with nullable=%t", tc.annotation, set, tc.nullable)
		}
		if where.TypeName != "int32" || where.Nullable == nil || *where.Nullable {
			t.Errorf("WHERE argument: got %+v, want non-nullable int32", where)
		}
	}
}
