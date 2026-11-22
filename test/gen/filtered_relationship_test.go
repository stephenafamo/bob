package testgen

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-sqlite/driver"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/orm"
)

func TestFilteredRelationshipInsert(t *testing.T) {
	root := t.TempDir()
	dsn := filepath.Join(root, "versions.sqlite")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.ExecContext(t.Context(), `
CREATE TABLE locations (id INTEGER PRIMARY KEY);
CREATE TABLE location_versions (id INTEGER PRIMARY KEY, location_id INTEGER NOT NULL REFERENCES locations(id),
  is_primary_version BOOLEAN NOT NULL DEFAULT false);
CREATE UNIQUE INDEX one_primary_per_location ON location_versions(location_id) WHERE is_primary_version;
`); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SQLITE_TEST_DSN", dsn)
	modPath, err := filepath.Abs("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	config := gen.Config[any]{
		TypeSystem: "database/sql",
		Aliases: drivers.Aliases{
			"locations":         {Relationships: map[string]string{"locations_primary_version": "PrimaryVersion"}},
			"location_versions": {Relationships: map[string]string{"locations_primary_version": "PrimaryVersionFor"}},
		},
		Relationships: gen.Relationships{
			"locations": {{Name: "locations_primary_version", NeverRequired: true, Sides: []orm.RelSide{{
				From: "locations", To: "location_versions", Columns: [][2]string{{"id", "location_id"}}, Modify: "from",
				ToWhere: []orm.RelWhere{{Column: "is_primary_version", SQLValue: "true", GoValue: "true"}},
			}}}},
		},
	}
	d := driver.New(driver.Config{Config: helpers.Config{Dsn: dsn}})
	testDriver(
		t,
		root,
		gen.SQLiteTemplates,
		config,
		d,
		modPath,
		[]string{"-run", "^TestInsertPrimaryVersion$"},
		templatePlugin[any]{dialect: "sqlite"},
	)
}
