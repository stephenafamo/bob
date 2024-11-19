package driver

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
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
		fs: testfiles.PostgresSchema,
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
		OverwriteGolden: false,
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
