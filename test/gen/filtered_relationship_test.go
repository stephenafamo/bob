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
CREATE TABLE teams (id INTEGER PRIMARY KEY, location_id INTEGER NOT NULL, parent_id INTEGER, active BOOLEAN NOT NULL);
CREATE TABLE links (location_id TEXT, team_id INTEGER NOT NULL);
CREATE TABLE location_versions (id INTEGER PRIMARY KEY, location_id INTEGER NOT NULL REFERENCES locations(id),
  is_primary_version BOOLEAN NOT NULL DEFAULT false);
CREATE UNIQUE INDEX one_primary_per_location ON location_versions(location_id) WHERE is_primary_version;
CREATE TABLE collections (id INTEGER PRIMARY KEY, code BLOB UNIQUE);
CREATE TABLE entries (id INTEGER PRIMARY KEY, code BLOB REFERENCES collections(code));
CREATE TABLE revisions (seq INTEGER NOT NULL, code BLOB NOT NULL, label TEXT NOT NULL, PRIMARY KEY(code, seq));
CREATE VIEW revision_labels AS SELECT label FROM revisions;
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
			"collections": {Relationships: map[string]string{"fk_entries_0": "Entries"}},
			"locations": {Relationships: map[string]string{
				"locations_primary_version": "PrimaryVersion", "primary_versions": "PrimaryVersions",
				"child_teams":  "Teams",
				"linked_teams": "LinkedTeams",
			}},
			"location_versions": {Relationships: map[string]string{
				"locations_primary_version": "PrimaryVersionFor", "primary_versions": "PrimaryVersionsFor",
			}},
		},
		Relationships: gen.Relationships{
			"locations": {{Name: "locations_primary_version", NeverRequired: true, Sides: []orm.RelSide{{
				From: "locations", To: "location_versions", Columns: [][2]string{{"id", "location_id"}}, Modify: "from",
				ToWhere: []orm.RelWhere{{Column: "is_primary_version", SQLValue: "true", GoValue: "true"}},
			}}}, {Name: "primary_versions", NeverRequired: true, Sides: []orm.RelSide{{
				From: "locations", To: "location_versions", Columns: [][2]string{{"id", "location_id"}}, Modify: "to",
				ToWhere: []orm.RelWhere{{Column: "is_primary_version", SQLValue: "true", GoValue: "true"}},
			}}}, {Name: "child_teams", NoReverse: true, Sides: []orm.RelSide{
				{From: "locations", To: "teams", Columns: [][2]string{{"id", "location_id"}}, Modify: "to"},
				{From: "teams", To: "teams", Columns: [][2]string{{"id", "parent_id"}}, Modify: "to"},
			}}, {Name: "linked_teams", NoReverse: true, Sides: []orm.RelSide{
				{From: "locations", To: "links", Columns: [][2]string{{"id", "location_id"}}, Modify: "to"},
				{From: "links", To: "teams", Columns: [][2]string{{"team_id", "id"}}, Modify: "from"},
			}}},
		},
	}
	for _, tc := range []struct {
		name             string
		disableBacklinks bool
	}{
		{name: "backlinks"},
		{name: "no_backlinks", disableBacklinks: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.NoBackReferencing = tc.disableBacklinks
			config.RelationLoadedName = "Fetched"
			d := driver.New(driver.Config{Config: helpers.Config{Dsn: dsn}})
			testDriver(t, t.TempDir(), gen.SQLiteTemplates, config, d, modPath,
				[]string{"-run", "^Test(InsertPrimaryVersion|RelationshipAssignment|RepeatedTableCorrelation|CompositeRowRefresh|ViewHookDispatch)$"},
				templatePlugin[any]{dialect: "sqlite"})
		})
	}
}
