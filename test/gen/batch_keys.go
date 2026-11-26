package testgen

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
)

// TestBatchKeys exercises generated loaders with composite keys.
func TestBatchKeys[T, C, I any](t *testing.T, db *sql.DB, dialect string, templates gen.Templates, driver drivers.Interface[T, C, I]) {
	t.Helper()
	if testing.Short() {
		t.Skip("requires generated models")
	}
	keyType := "varchar(40)"
	tenantType := "bigint unsigned"
	fk := "batch_fk"
	switch dialect {
	case "psql":
		fk = "batch_children." + fk
		tenantType = "bigint"
	case "sqlite":
		fk = "fk_batch_children_0"
		tenantType = "integer"
		keyType = "text"
	}
	ddl := `CREATE TABLE batch_parents (tenant ` + tenantType + ` NOT NULL, code ` + keyType + ` NOT NULL, PRIMARY KEY (tenant, code));
		CREATE TABLE batch_children (id int PRIMARY KEY, tenant ` + tenantType + ` NOT NULL, code ` + keyType + `,
		CONSTRAINT batch_fk FOREIGN KEY (tenant, code) REFERENCES batch_parents(tenant, code));`
	switch dialect {
	case "psql":
		ddl += `CREATE TABLE batch_arrays (tenant int, id int[], PRIMARY KEY (tenant, id));
			CREATE TABLE batch_items (id int PRIMARY KEY, tenant int, key int[],
				CONSTRAINT batch_items_key_fkey FOREIGN KEY (tenant, key) REFERENCES batch_arrays(tenant, id));`
	}
	_, err := db.ExecContext(t.Context(), ddl)
	if err != nil {
		t.Fatal(err)
	}
	mod, err := filepath.Abs("../../../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	aliases := drivers.Aliases{
		"batch_parents":  {Relationships: map[string]string{fk: "Children"}},
		"batch_children": {Relationships: map[string]string{fk: "Parent"}},
	}
	if dialect == "psql" {
		aliases["batch_arrays"] = drivers.TableAlias{Relationships: map[string]string{"batch_items.batch_items_key_fkey": "Items"}}
	}
	testDriver(t, t.TempDir(), templates, gen.Config[C]{Aliases: aliases}, driver, mod,
		[]string{"-run", "^TestBatch(Key|Array)Ownership$"}, templatePlugin[C]{dialect: dialect})
}
