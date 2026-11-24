package testgen

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/gen"
	"github.com/stephenafamo/bob/gen/drivers"
)

// TestBatchKeys exercises generated loaders against a case-insensitive
// composite foreign key in the caller's disposable database.
func TestBatchKeys[T, C, I any](t *testing.T, db *sql.DB, dialect string, templates gen.Templates, driver drivers.Interface[T, C, I]) {
	t.Helper()
	if testing.Short() {
		t.Skip("requires generated models")
	}
	keyType := "varchar(40) CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci"
	tenantType := "bigint unsigned"
	fk := "batch_fk"
	if dialect == "psql" {
		fk = "batch_children." + fk
		tenantType = "bigint"
		if _, err := db.ExecContext(
			t.Context(),
			`CREATE COLLATION batch_ci (provider = icu, locale = 'und-u-ks-level2', deterministic = false)`,
		); err != nil {
			t.Fatal(err)
		}
		keyType = "character(40) COLLATE batch_ci"
	}
	ddl := `CREATE TABLE batch_parents (tenant ` + tenantType + ` NOT NULL, code ` + keyType + ` NOT NULL, PRIMARY KEY (tenant, code));
		CREATE TABLE batch_children (id int PRIMARY KEY, tenant ` + tenantType + ` NOT NULL, code ` + keyType + `,
		CONSTRAINT batch_fk FOREIGN KEY (tenant, code) REFERENCES batch_parents(tenant, code));`
	switch dialect {
	case "psql":
		ddl += `CREATE TABLE batch_arrays (tenant int, id int[], PRIMARY KEY (tenant, id));
			CREATE TABLE batch_items (id int PRIMARY KEY, tenant int, key int[],
				CONSTRAINT batch_items_key_fkey FOREIGN KEY (tenant, key) REFERENCES batch_arrays(tenant, id));`
	case "mysql":
		ddl += `CREATE TABLE batch_exact_parents (id INT PRIMARY KEY,
			amount DECIMAL(65,0) NOT NULL, ratio DECIMAL(35,30) NOT NULL, token VARBINARY(16) NOT NULL, stamp DATETIME(6) NOT NULL,
			UNIQUE (amount, ratio, token, stamp));
			CREATE TABLE batch_exact_children (id INT PRIMARY KEY,
			amount DECIMAL(65,0) NOT NULL, ratio DECIMAL(35,30) NOT NULL, token VARBINARY(20) NOT NULL, stamp DATETIME(6) NOT NULL,
			CONSTRAINT batch_exact_fk FOREIGN KEY (amount, ratio, token, stamp) REFERENCES batch_exact_parents(amount, ratio, token, stamp));`
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
	switch dialect {
	case "psql":
		aliases["batch_arrays"] = drivers.TableAlias{Relationships: map[string]string{"batch_items.batch_items_key_fkey": "Items"}}
	case "mysql":
		aliases["batch_exact_parents"] = drivers.TableAlias{Relationships: map[string]string{"batch_exact_fk": "Children"}}
	}
	testDriver(t, t.TempDir(), templates, gen.Config[C]{Aliases: aliases}, driver, mod,
		[]string{"-run", "^TestBatch(Key|Array|Exact)Ownership$"}, templatePlugin[C]{dialect: dialect})
}
