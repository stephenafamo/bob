package gen

import (
	"errors"
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
)

func TestValidateAliases_NoConflicts(t *testing.T) {
	a := drivers.Aliases{
		"table1": drivers.TableAlias{
			UpSingular:    "Table1",
			UpPlural:      "Table1s",
			DownSingular:  "table1",
			DownPlural:    "table1s",
			Columns:       map[string]string{"id": "ID", "name": "Name"},
			Relationships: map[string]string{"rel1": "Rel1"},
		},
		"table2": drivers.TableAlias{
			UpSingular:    "Table2",
			UpPlural:      "Table2s",
			DownSingular:  "table2",
			DownPlural:    "table2s",
			Columns:       map[string]string{"id": "ID2", "desc": "Desc"},
			Relationships: map[string]string{"rel2": "Rel2"},
		},
	}
	if err := validateAliases(a); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestValidateAliases_GlobalAliasConflict(t *testing.T) {
	a := drivers.Aliases{
		"table1": drivers.TableAlias{
			UpSingular:    "Table", // Conflict here
			UpPlural:      "Tables",
			DownSingular:  "tablesss", // Conflict here
			DownPlural:    "tables",
			Columns:       map[string]string{"id": "ID"},
			Relationships: map[string]string{"rel1": "Rel1"},
		},
		"table2": drivers.TableAlias{
			UpSingular:    "Table", // Conflict here
			UpPlural:      "Tables2",
			DownSingular:  "table2",
			DownPlural:    "tablesss", // Conflict here
			Columns:       map[string]string{"id": "ID2"},
			Relationships: map[string]string{"rel2": "Rel2"},
		},
	}

	err := validateAliases(a)

	expectedError1 := globalAliasError{
		Value:  "Table",
		Type1:  "UpSingular",
		Type2:  "UpSingular",
		Table1: "table1",
		Table2: "table2",
	}
	if !errors.Is(err, expectedError1) {
		t.Errorf("expected %#v, got %v", expectedError1, err)
	}

	expectedError2 := globalAliasError{
		Value:  "tablesss",
		Type1:  "DownSingular",
		Type2:  "DownPlural",
		Table1: "table1",
		Table2: "table2",
	}
	if !errors.Is(err, expectedError2) {
		t.Errorf("expected %#v, got %v", expectedError2, err)
	}
}

func TestValidateAliases_ColumnAliasConflict(t *testing.T) {
	a := drivers.Aliases{
		"table1": drivers.TableAlias{
			UpSingular:    "Table1",
			UpPlural:      "Table1s",
			DownSingular:  "table1",
			DownPlural:    "table1s",
			Columns:       map[string]string{"id": "ID", "other": "ID"}, // Conflict
			Relationships: map[string]string{"rel1": "Rel1"},
		},
	}
	expectedError := tableAliasError{
		Type:      "column",
		Value:     "ID",
		Table:     "table1",
		Conflict1: "id",
		Conflict2: "other",
	}
	err := validateAliases(a)
	if !errors.Is(err, expectedError) {
		t.Errorf("expected %#v, got %v", expectedError, err)
	}
}

func TestValidateAliases_RelationshipAliasConflict(t *testing.T) {
	a := drivers.Aliases{
		"table1": drivers.TableAlias{
			UpSingular:    "Table1",
			UpPlural:      "Table1s",
			DownSingular:  "table1",
			DownPlural:    "table1s",
			Columns:       map[string]string{"id": "ID"},
			Relationships: map[string]string{"rel1": "Rel", "rel2": "Rel"}, // Conflict
		},
	}

	expectedError := tableAliasError{
		Type:      "relationship",
		Value:     "Rel",
		Table:     "table1",
		Conflict1: "rel1",
		Conflict2: "rel2",
	}
	err := validateAliases(a)
	if !errors.Is(err, expectedError) {
		t.Errorf("expected %#v, got %v", expectedError, err)
	}
}
