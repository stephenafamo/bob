package driver

import (
	"context"
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	testfiles "github.com/stephenafamo/bob/test/files"
	testgen "github.com/stephenafamo/bob/test/gen"
)

func TestPostgres(t *testing.T) {
	t.Parallel()
	out, cleanup := prep(t, "psql")
	defer cleanup()

	config := Config{
		fs: testfiles.PostgresSchema,
	}

	testgen.TestDriver(t, testgen.DriverTestConfig[any]{
		Root: out,
		GetDriver: func() drivers.Interface[any] {
			d, err := getPsqlDriver(context.Background(), config)
			if err != nil {
				t.Fatalf("getting psql driver: %s", err)
			}
			return d
		},
		GoldenFile:      "../../bobgen-psql/driver/psql.golden.json",
		OverwriteGolden: false,
		Templates:       &helpers.Templates{Models: []fs.FS{gen.PSQLModelTemplates}},
	})
}

func TestMySQL(t *testing.T) {
	t.Parallel()
	out, cleanup := prep(t, "mysql")
	defer cleanup()

	config := Config{
		fs: testfiles.MySQLSchema,
	}

	testgen.TestDriver(t, testgen.DriverTestConfig[any]{
		Root: out,
		GetDriver: func() drivers.Interface[any] {
			d, err := getMySQLDriver(context.Background(), config)
			if err != nil {
				t.Fatalf("getting mysql driver: %s", err)
			}
			return d
		},
		GoldenFile:      "../../bobgen-mysql/driver/mysql.golden.json",
		OverwriteGolden: false,
		Templates:       &helpers.Templates{Models: []fs.FS{gen.MySQLModelTemplates}},
	})
}

func TestSQLite(t *testing.T) {
	t.Parallel()
	out, cleanup := prep(t, "sqlite")
	defer cleanup()

	config := Config{
		fs:      testfiles.SQLiteSchema,
		Schemas: []string{"one"},
	}

	testgen.TestDriver(t, testgen.DriverTestConfig[any]{
		Root: out,
		GetDriver: func() drivers.Interface[any] {
			d, err := getSQLiteDriver(context.Background(), config)
			if err != nil {
				t.Fatalf("getting sqlite driver: %s", err)
			}
			return d
		},
		GoldenFile:      "../../bobgen-sqlite/driver/sqlite.golden.json",
		OverwriteGolden: false,
		Templates:       &helpers.Templates{Models: []fs.FS{gen.SQLiteModelTemplates}},
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
