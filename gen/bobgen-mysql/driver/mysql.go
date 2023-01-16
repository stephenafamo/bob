package driver

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
	"sync"
	"text/template"

	"github.com/go-sql-driver/mysql"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
	"github.com/volatiletech/strmangle"
)

//go:embed templates
var templates embed.FS

//nolint:gochecknoglobals
var (
	ModelTemplates, _ = fs.Sub(templates, "templates/models")
	rgxEnum           = regexp.MustCompile(`^enum\([^\)]+\)$`)
)

type (
	Interface = drivers.Interface[any]
	DBInfo    = drivers.DBInfo[any]
)

type Config struct {
	// The database connection string
	Dsn string
	// List of tables that will be included. Others are ignored
	Only map[string][]string
	// List of tables that will be should be ignored. Others are included
	Except map[string][]string
	// How many tables to fetch in parallel
	Concurrency int

	//-------

	// The name of the folder to output the models package to
	Output string
	// The name you wish to assign to your generated models package
	Pkgname   string
	NoFactory bool `yaml:"no_factory"`
}

func New(config Config) Interface {
	return &Driver{config: config}
}

// Driver holds the database connection string and a handle
// to the database connection.
type Driver struct {
	config Config

	conn   *sql.DB
	dbName string

	enums  []drivers.Enum
	enumMu sync.Mutex
}

func (d *Driver) Capabilities() drivers.Capabilities {
	return drivers.Capabilities{
		BulkInsert: false,
	}
}

// Assemble all the information we need to provide back to the driver
func (d *Driver) Assemble(ctx context.Context) (*DBInfo, error) {
	var dbinfo *DBInfo
	var err error

	config, err := mysql.ParseDSN(d.config.Dsn)
	if err != nil {
		return nil, err
	}

	if config.DBName == "" {
		return nil, fmt.Errorf("no database name given: %w", err)
	}
	d.dbName = config.DBName

	d.conn, err = sql.Open("mysql", d.config.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer d.conn.Close()

	dbinfo = &DBInfo{}

	dbinfo.Tables, err = drivers.BuildDBInfo(ctx, d, d.config.Concurrency, d.config.Only, d.config.Except)
	if err != nil {
		return nil, err
	}

	dbinfo.Enums = d.enums
	sort.Slice(dbinfo.Enums, func(i, j int) bool {
		return dbinfo.Enums[i].Type < dbinfo.Enums[j].Type
	})

	return dbinfo, err
}

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (d *Driver) TablesInfo(ctx context.Context, tableFilter drivers.Filter) (drivers.TablesInfo, error) {
	query := "SELECT table_name as `key`, table_name as name FROM information_schema.tables WHERE table_schema = ?"
	args := []any{d.dbName}

	include := tableFilter.Only
	exclude := tableFilter.Except

	if len(include) > 0 {
		query += fmt.Sprintf(" and table_name in (%s)", strmangle.Placeholders(true, len(include), 3, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf(" and table_name not in (%s)", strmangle.Placeholders(true, len(exclude), 3+len(include), 1))
		for _, w := range exclude {
			args = append(args, w)
		}
	}

	query += ` order by table_name;`

	return stdscan.All(ctx, d.conn, scan.StructMapper[drivers.TableInfo](), query, args...)
}

// Load details about a single table
func (d *Driver) TableDetails(ctx context.Context, info drivers.TableInfo, colFilter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
	filter := colFilter[info.Key]
	var columns []drivers.Column
	schema := d.dbName
	tableName := info.Name
	args := []interface{}{tableName, tableName, schema, schema, schema, schema, tableName, tableName, schema}

	query := `
	select
	c.column_name,
	c.column_type,
	c.column_comment,
	c.data_type,
	if (extra = 'auto_increment','auto_increment', c.column_default) AS default_value,
	c.is_nullable = 'YES' AS nullable,
	(c.extra = 'STORED GENERATED' OR c.extra = 'VIRTUAL GENERATED') is_generated,
	exists (
		select c.column_name
		from information_schema.table_constraints tc
		inner join information_schema.key_column_usage kcu
			on tc.constraint_name = kcu.constraint_name
		where tc.table_name = ? and kcu.table_name = ? and tc.table_schema = ? and kcu.table_schema = ? and
			c.column_name = kcu.column_name and
			(tc.constraint_type = 'PRIMARY KEY' or tc.constraint_type = 'UNIQUE') and
			(select count(*) from information_schema.key_column_usage where table_schema = ? and
			constraint_schema = ? and table_name = ? and constraint_name = tc.constraint_name) = 1
	) as is_unique
	from information_schema.columns as c
	where table_name = ? and table_schema = ?`

	if len(filter.Only) > 0 {
		query += fmt.Sprintf(" and c.column_name in (%s)", strings.Repeat(",?", len(filter.Only))[1:])
		for _, w := range filter.Only {
			args = append(args, w)
		}
	} else if len(filter.Except) > 0 {
		if len(filter.Except) > 0 {
			query += fmt.Sprintf(" and c.column_name not in (%s)", strings.Repeat(",?", len(filter.Except))[1:])
			for _, w := range filter.Except {
				args = append(args, w)
			}
		}
	}

	query += ` order by c.ordinal_position;`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName, colFullType, colComment, colType string
		var nullable, generated, unique bool
		var defaultValue *string
		if err := rows.Scan(&colName, &colFullType, &colComment, &colType, &defaultValue, &nullable, &generated, &unique); err != nil {
			return "", "", nil, fmt.Errorf("unable to scan for table %s: %w", tableName, err)
		}

		if colFullType == "tinyint(1)" {
			colType = "bool"
		}

		column := drivers.Column{
			Name:      colName,
			Comment:   colComment,
			DBType:    colType,
			Nullable:  nullable,
			Unique:    unique,
			Generated: generated,
		}

		if defaultValue != nil {
			column.Default = *defaultValue
		}

		// A generated column technically has a default value
		if column.Default == "" && column.Generated {
			column.Default = "AUTO_GENERATED"
		}

		if !rgxEnum.MatchString(colFullType) {
			column = d.translateColumnType(column, colFullType)
		} else {
			enumTyp := strmangle.TitleCase(tableName + "_" + colName)
			column.Type = enumTyp
			d.enumMu.Lock()
			d.enums = append(d.enums, drivers.Enum{
				Type:   enumTyp,
				Values: parseEnumVals(colFullType),
			})
			d.enumMu.Unlock()
		}

		columns = append(columns, column)
	}

	return "", tableName, columns, nil
}

// parseEnumVals returns the values from an enum string
// mysql: enum('values'...)
func parseEnumVals(s string) []string {
	s = s[6 : len(s)-2]
	return strings.Split(s, "','")
}

// translateTableColumnType converts mysql database types to Go types, for example
// "varchar" to "string" and "bigint" to "int64". It returns this parsed data
// as a Column object.
func (*Driver) translateColumnType(c drivers.Column, fullType string) drivers.Column {
	unsigned := strings.HasSuffix(fullType, " unsigned")
	switch c.DBType {
	case "tinyint":
		if unsigned {
			c.Type = "uint8"
		} else {
			c.Type = "int8"
		}
	case "smallint":
		if unsigned {
			c.Type = "uint16"
		} else {
			c.Type = "int16"
		}
	case "mediumint":
		if unsigned {
			c.Type = "uint32"
		} else {
			c.Type = "int32"
		}
	case "int", "integer":
		if unsigned {
			c.Type = "uint"
		} else {
			c.Type = "int"
		}
	case "bigint":
		if unsigned {
			c.Type = "uint64"
		} else {
			c.Type = "int64"
		}
	case "float":
		c.Type = "float32"
	case "double", "double precision", "real":
		c.Type = "float64"
	case "boolean", "bool":
		c.Type = "bool"
	case "date", "datetime", "timestamp":
		c.Type = "time.Time"
		c.Imports = append(c.Imports, typMap[c.Type]...)
	case "binary", "varbinary", "tinyblob", "blob", "mediumblob", "longblob":
		c.Type = "[]byte"
	case "numeric", "decimal", "dec", "fixed":
		c.Type = "decimal.Decimal"
		c.Imports = append(c.Imports, typMap[c.Type]...)
	case "json":
		c.Type = "types.JSON[json.RawMessage]"
		c.Imports = append(c.Imports, typMap[c.Type]...)
	default:
		c.Type = "string"
	}

	return c
}

//nolint:gochecknoglobals
var typMap = map[string]importers.List{
	"time.Time":                   {`"time"`},
	"decimal.Decimal":             {`"github.com/shopspring/decimal"`},
	"types.JSON[json.RawMessage]": {`"encoding/json"`, `"github.com/stephenafamo/bob/types"`},
}

func (d *Driver) Constraints(ctx context.Context, _ drivers.ColumnFilter) (drivers.DBConstraints, error) {
	ret := drivers.DBConstraints{
		PKs:     map[string]*drivers.Constraint{},
		FKs:     map[string][]drivers.ForeignKey{},
		Uniques: map[string][]drivers.Constraint{},
	}

	query := `SELECT
	tc.table_name AS table_name,
	tc.constraint_name AS name,
	tc.constraint_type AS type,
	kcu.column_name AS column_name,
	referenced_table_name AS foreign_table,
	referenced_column_name AS foreign_column
	FROM information_schema.table_constraints AS tc
	LEFT JOIN information_schema.key_column_usage AS kcu 
		ON kcu.table_name = tc.table_name 
		AND kcu.table_schema = tc.table_schema 
		AND kcu.constraint_name = tc.constraint_name
	WHERE tc.constraint_type IN ('PRIMARY KEY', 'UNIQUE', 'FOREIGN KEY') AND tc.table_schema = ?
	ORDER BY tc.table_name, tc.constraint_name, tc.constraint_type, kcu.ordinal_position`

	type constraint struct {
		TableName     string
		Name          string
		Type          string
		ColumnName    string
		ForeignTable  sql.NullString
		ForeignColumn sql.NullString
	}
	constraints, err := stdscan.All(ctx, d.conn, scan.StructMapper[constraint](), query, d.dbName)
	if err != nil {
		return ret, err
	}

	// Extra for the loop
	constraints = append(constraints, constraint{})

	var current drivers.Constraint
	var table, foreignTable, currentTyp string
	var foreignCols []string
	for i, c := range constraints {
		if i != 0 && (c.TableName != table || c.Name != current.Name || c.Type != currentTyp) {
			switch currentTyp {
			case "PRIMARY KEY":
				// Create a new constraint because it is a pointer
				ret.PKs[table] = &drivers.Constraint{
					Name:    current.Name,
					Columns: current.Columns,
				}
			case "UNIQUE":
				ret.Uniques[table] = append(ret.Uniques[table], current)
			case "FOREIGN KEY":
				ret.FKs[table] = append(ret.FKs[table], drivers.ForeignKey{
					Constraint:     current,
					ForeignTable:   foreignTable,
					ForeignColumns: foreignCols,
				})
			}

			// reset things
			current = drivers.Constraint{}
			table, foreignTable, currentTyp, foreignCols = "", "", "", nil //nolint:ineffassign
		}

		table = c.TableName
		currentTyp = c.Type

		current.Name = c.Name
		current.Columns = append(current.Columns, c.ColumnName)
		if c.ForeignTable.Valid {
			foreignTable = c.ForeignTable.String
			foreignCols = append(foreignCols, c.ForeignColumn.String)
		}
	}

	return ret, nil
}

func CustomFuncs() template.FuncMap {
	fm := map[string]any{
		"uniqueColPairs": uniqueColPairs,
	}
	return fm
}

func uniqueColPairs(t drivers.Table) string {
	ret := make([]string, 0, len(t.Uniques)+1)
	if t.PKey != nil {
		ret = append(ret, fmt.Sprintf("%#v", t.PKey.Columns))
	}

	for _, unique := range t.Uniques {
		ret = append(ret, fmt.Sprintf("%#v", unique.Columns))
	}

	return strings.Join(ret, ", ")
}
