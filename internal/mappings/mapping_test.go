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
	if diff := cmp.Diff(want, WriteColumns[*model]([]string{"row_id", "computed", "owner.row_id", "owner.computed", "label"})); diff != "" {
		t.Fatal(diff)
	}
}

func TestCustomScannerWriteColumns(t *testing.T) {
	type point struct{ X, Y int }
	type model struct {
		Pos      point `db:"pos"`
		Computed point `db:",generated"`
		ID       int   `db:",autoincr"`
	}
	want := Mapping{NonGenerated: []string{"id", "pos"}, AutoIncrement: []string{"id"}}
	if diff := cmp.Diff(want, WriteColumns[model]([]string{"id", "pos", "computed"})); diff != "" {
		t.Fatal(diff)
	}
}
