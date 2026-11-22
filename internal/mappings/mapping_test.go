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
		All:           []string{"row_id", "computed", "owner.row_id", "owner.computed", "label"},
		NonGenerated:  []string{"row_id", "owner.row_id", "label"},
		AutoIncrement: []string{"row_id", "owner.row_id"},
	}
	if diff := cmp.Diff(want, GetMappings[*model]()); diff != "" {
		t.Fatal(diff)
	}
}

func TestRootScannerHasNoNamedColumns(t *testing.T) {
	cols := GetMappings[sql.NullString]()
	if len(cols.All) != 0 || len(cols.NonGenerated) != 0 || len(cols.AutoIncrement) != 0 {
		t.Fatalf("root scanner mapped as table columns: %+v", cols)
	}
}
