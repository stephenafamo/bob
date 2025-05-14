package driver

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	psqlDriver "github.com/stephenafamo/bob/gen/bobgen-psql/driver"
	sqliteDriver "github.com/stephenafamo/bob/gen/bobgen-sqlite/driver"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
)

func TestPostgres(t *testing.T) {
	t.Parallel()
	out, cleanup := prep(t, "psql")
	defer cleanup()

	config := Config{
		Pattern: "psql/*.sql",
		fs:      testfiles.PostgresSchema,
	}

	testgen.TestDriver(t, testgen.DriverTestConfig[any, any, psqlDriver.IndexExtra]{
		Root: out,
		GetDriver: func() drivers.Interface[any, any, psqlDriver.IndexExtra] {
			d, err := getPsqlDriver(context.Background(), config)
			if err != nil {
				t.Fatalf("getting psql driver: %s", err)
			}
			return d
		},
		GoldenFile:      "../../bobgen-psql/driver/psql.golden.json",
		Templates:       &helpers.Templates{Models: []fs.FS{gen.PSQLModelTemplates}},
		OverwriteGolden: false,
	})
}

func TestMySQL(t *testing.T) {
	t.Parallel()
	out, cleanup := prep(t, "mysql")
	defer cleanup()

	config := Config{
		Pattern: "mysql/*.sql",
		fs:      testfiles.MySQLSchema,
	}

	testgen.TestDriver(t, testgen.DriverTestConfig[any, any, any]{
		Root: out,
		GetDriver: func() drivers.Interface[any, any, any] {
			d, err := getMySQLDriver(context.Background(), config)
			if err != nil {
				t.Fatalf("getting mysql driver: %s", err)
			}
			return d
		},
		GoldenFile:      "../../bobgen-mysql/driver/mysql.golden.json",
		Templates:       &helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
		OverwriteGolden: false,
	})
}

func TestSQLite(t *testing.T) {
	t.Parallel()
	out, cleanup := prep(t, "sqlite")
	defer cleanup()

	config := Config{
		Schemas: []string{"one"},
		Pattern: "sqlite/*.sql",
		fs:      testfiles.SQLiteSchema,
	}

	testgen.TestDriver(t, testgen.DriverTestConfig[any, any, sqliteDriver.IndexExtra]{
		Root: out,
		GetDriver: func() drivers.Interface[any, any, sqliteDriver.IndexExtra] {
			d, err := getSQLiteDriver(context.Background(), config)
			if err != nil {
				t.Fatalf("getting sqlite driver: %s", err)
			}
			return d
		},
		GoldenFile:      "../../bobgen-sqlite/driver/sqlite.golden.json",
		Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
		OverwriteGolden: false,
	})
}

func prep(t *testing.T, name string) (string, func()) {
	t.Helper()
	out, err := os.MkdirTemp("", fmt.Sprintf("bobgen_sql_%s_", name))
	if err != nil {
		t.Fatalf("unable to create tempdir: %s", err)
	}

	return out, func() {
		if t.Failed() {
			t.Log("template test output:", out)
			return
		}
		os.RemoveAll(out)
	}
}
