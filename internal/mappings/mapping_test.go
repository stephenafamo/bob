package mappings

import (
	"database/sql"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestMappedWriteColumns(t *testing.T) {
	type Fields struct {
		ID       int    `db:"row_id,pk,autoincr"`
		Computed int    `db:",generated"`
		Ignored  string `db:"-"`
	}
	type model struct {
		*Fields
		Author Fields `db:"owner"`
		Label  sql.NullString
	}
	want := Mapping{
		NonGenerated:  []string{"row_id", "owner.row_id", "label"},
		AutoIncrement: []string{"row_id", "owner.row_id"},
	}
	if diff := cmp.Diff(want, WriteColumns[*model](false, nil)); diff != "" {
		t.Fatal(diff)
	}
}

func TestCustomScannerWriteColumns(t *testing.T) {
	if diff := cmp.Diff([]string{"code"}, WriteColumns[string](true, []string{"code"}).NonGenerated); diff != "" {
		t.Fatal(diff)
	}
	type point struct{ X, Y int }
	type model struct {
		Pos      point `db:"pos"`
		Computed point `db:",generated"`
		ID       int   `db:",autoincr"`
	}
	want := Mapping{NonGenerated: []string{"pos", "id"}, AutoIncrement: []string{"id"}}
	if diff := cmp.Diff(want, WriteColumns[model](true, []string{"id", "pos"})); diff != "" {
		t.Fatal(diff)
	}
	want.NonGenerated = []string{"pos.x", "pos.y", "id"}
	if diff := cmp.Diff(want, WriteColumns[model](true, []string{"id", "pos.y"})); diff != "" {
		t.Fatal(diff)
	}
}
