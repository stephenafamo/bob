package testgen

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-sqlite/driver"
	"github.com/stephenafamo/bob/orm"
)

func TestFactoryBridgeParents(t *testing.T) {
	root := t.TempDir()
	dsn := filepath.Join(root, "bridge.sqlite")
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	_, err = db.ExecContext(t.Context(), `
CREATE TABLE tasks (id INTEGER PRIMARY KEY);
CREATE TABLE exporters (id INTEGER PRIMARY KEY);
CREATE TABLE links (id INTEGER PRIMARY KEY, note TEXT NOT NULL,
  task_id INTEGER NOT NULL REFERENCES tasks, exporter_id INTEGER NOT NULL REFERENCES exporters,
  UNIQUE(task_id, exporter_id));
CREATE TRIGGER no_relink BEFORE UPDATE ON links BEGIN SELECT RAISE(ABORT, 'unexpected bridge update'); END;
`)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("SQLITE_TEST_DSN", dsn)
	modPath, err := filepath.Abs("../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	config := gen.Config[any]{Relationships: gen.Relationships{
		"tasks": {{Name: "task_exporters", Sides: []orm.RelSide{
			{From: "tasks", To: "links", Columns: [][2]string{{"id", "task_id"}}},
			{From: "links", To: "exporters", Columns: [][2]string{{"exporter_id", "id"}}},
		}}},
	}}
	d := driver.New(driver.Config{Config: helpers.Config{Dsn: dsn}})
	testDriver(
		t,
		root,
		gen.SQLiteTemplates,
		config,
		d,
		modPath,
		[]string{"-run", "^TestCreateExportersThroughLinks$"},
		templatePlugin[any]{dialect: "sqlite"},
	)
}
