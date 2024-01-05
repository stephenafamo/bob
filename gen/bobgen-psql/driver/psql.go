package driver

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/lib/pq"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
	"github.com/stephenafamo/scan"
	"github.com/stephenafamo/scan/stdscan"
	"github.com/volatiletech/strmangle"
)

type (
	Interface = drivers.Interface[any]
	DBInfo    = drivers.DBInfo[any]
)

type Enum struct {
	Schema string
	Name   string
	Type   string
	Values pq.StringArray
}

type Config struct {
	// The database connection string
	Dsn string
	// The database schemas to generate models for
	Schemas pq.StringArray
	// The name of this schema will not be included in the generated models
	// a context value can then be used to set the schema at runtime
	// useful for multi-tenant setups
	SharedSchema string `yaml:"shared_schema"`
	// List of tables that will be included. Others are ignored
	Only map[string][]string
	// List of tables that will be should be ignored. Others are included
	Except map[string][]string
	// How many tables to fetch in parallel
	Concurrency int
	// Which UUID package to use (gofrs or google)
	UUIDPkg string `yaml:"uuid_pkg"`

	//-------

	// The name of the folder to output the models package to
	Output string
	// The name you wish to assign to your generated models package
	Pkgname   string
	NoFactory bool `yaml:"no_factory"`
}

func New(config Config) Interface {
	// Set defaults
	if config.Schemas == nil {
		config.Schemas = pq.StringArray{"public"}
	}

	if config.SharedSchema == "" {
		config.SharedSchema = config.Schemas[0]
	}

	if config.UUIDPkg == "" {
		config.UUIDPkg = "gofrs"
	}

	if config.Concurrency < 1 {
		config.Concurrency = 10
	}

	types := helpers.Types()

	switch config.UUIDPkg {
	case "google":
		types["uuid.UUID"] = drivers.Type{
			Imports:    importers.List{`"github.com/google/uuid"`},
			RandomExpr: `return any(uuid.New()).(T)`,
		}
	default:
		types["uuid.UUID"] = drivers.Type{
			Imports:    importers.List{`"github.com/gofrs/uuid/v5"`},
			RandomExpr: `return any(uuid.Must(uuid.NewV4())).(T)`,
		}
	}

	return &driver{
		config: config,
		types:  types,
	}
}

// driver holds the database connection string and a handle
// to the database connection.
type driver struct {
	config Config
	conn   *sql.DB
	enums  []Enum
	types  drivers.Types
}

func (d *driver) Dialect() string {
	return "psql"
}

func (d *driver) Types() drivers.Types {
	return d.types
}

func (d *driver) Capabilities() drivers.Capabilities {
	return drivers.Capabilities{}
}

// Assemble all the information we need to provide back to the driver
func (d *driver) Assemble(ctx context.Context) (*DBInfo, error) {
	var dbinfo *DBInfo
	var err error

	if d.config.Dsn == "" {
		return nil, fmt.Errorf("database dsn is not set")
	}

	d.conn, err = sql.Open("postgres", d.config.Dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	defer d.conn.Close()

	dbinfo = &DBInfo{}

	// drivers.Tables call translateColumnType which uses Enums
	if err := d.loadEnums(ctx); err != nil {
		return nil, fmt.Errorf("unable to load enums: %w", err)
	}

	dbinfo.Tables, err = drivers.BuildDBInfo(ctx, d, d.config.Concurrency, d.config.Only, d.config.Except)
	if err != nil {
		return nil, err
	}

	dbinfo.Enums = make([]drivers.Enum, len(d.enums))
	for i, e := range d.enums {
		dbinfo.Enums[i] = drivers.Enum{
			Type:   e.Type,
			Values: e.Values,
		}
	}

	sort.Slice(dbinfo.Enums, func(i, j int) bool {
		return dbinfo.Enums[i].Type < dbinfo.Enums[j].Type
	})

	return dbinfo, err
}

const keyClause = "(CASE WHEN table_schema <> $1 THEN table_schema|| '.'  ELSE '' END || table_name)"

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (d *driver) TablesInfo(ctx context.Context, tableFilter drivers.Filter) (drivers.TablesInfo, error) {
	query := fmt.Sprintf(`SELECT
	  %s AS "key" ,
	  table_schema AS "schema",
	  table_name AS "name"
	FROM (
	  SELECT
		table_name,
		table_schema
	  FROM
		information_schema.tables
	  UNION
	  SELECT
		matviewname AS table_name,
		schemaname AS table_schema
	  FROM
		pg_matviews) AS v
	WHERE
	  v.table_schema = ANY ($2)`, keyClause)
	args := []any{d.config.SharedSchema, d.config.Schemas}

	include := tableFilter.Only
	exclude := tableFilter.Except

	if len(include) > 0 {
		query += fmt.Sprintf(" and %s in (%s)", keyClause, strmangle.Placeholders(true, len(include), 3, 1))
		for _, w := range include {
			args = append(args, w)
		}
	}

	if len(exclude) > 0 {
		query += fmt.Sprintf(" and %s not in (%s)", keyClause, strmangle.Placeholders(true, len(exclude), 3+len(include), 1))
		for _, w := range exclude {
			args = append(args, w)
		}
	}

	query += ` order by table_name;`

	return stdscan.All(ctx, d.conn, scan.StructMapper[drivers.TableInfo](), query, args...)
}

// Load details about a single table
func (d *driver) TableDetails(ctx context.Context, info drivers.TableInfo, colFilter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
	var columns []drivers.Column
	args := []any{info.Schema, info.Name}

	tableQuery := `
	SELECT
		c.ordinal_position,
		c.column_name,
		ct.column_type,
		c.udt_schema,
		c.udt_name,
		(
			SELECT
				data_type
			FROM
				information_schema.element_types e
			WHERE
				c.table_catalog = e.object_catalog
				AND c.table_schema = e.object_schema
				AND c.table_name = e.object_name
				AND 'TABLE' = e.object_type
				AND c.dtd_identifier = e.collection_type_identifier
		) AS array_type,
	c.domain_name,
	c.column_default,
	coalesce(col_description(('"' || c.table_schema || '"."' || c.table_name || '"')::regclass::oid, ordinal_position), '') AS column_comment,
	c.is_nullable = 'YES' AS is_nullable,
	(
		CASE WHEN c.is_generated = 'ALWAYS'
			OR c.identity_generation = 'ALWAYS' THEN
			TRUE
		ELSE
			FALSE
		END) AS is_generated,
	(
		CASE WHEN (
			SELECT
				CASE WHEN column_name = 'is_identity' THEN
				(
					SELECT
						c.is_identity = 'YES' AS is_identity)
				ELSE
					FALSE
				END AS is_identity
			FROM
				information_schema.columns
			WHERE
				table_schema = 'information_schema'
				AND table_name = 'columns'
				AND column_name = 'is_identity') IS NULL THEN
			'NO'
		ELSE
			is_identity
		END) = 'YES' AS is_identity
	FROM
		information_schema.columns AS c
		INNER JOIN pg_namespace AS pgn ON pgn.nspname = c.udt_schema
		LEFT JOIN pg_type pgt ON c.data_type = 'USER-DEFINED'
			AND pgn.oid = pgt.typnamespace
			AND c.udt_name = pgt.typname,
			LATERAL (
				SELECT
					(
						CASE WHEN pgt.typtype = 'e' THEN
							'ENUM'
						ELSE
							c.data_type
						END) AS column_type) ct
	WHERE c.table_name = $2 and c.table_schema = $1
	ORDER BY c.ordinal_position`

	//nolint:gosec
	query := fmt.Sprintf(`SELECT 
		column_name,
		column_type,
		udt_schema,
		udt_name,
		array_type,
		domain_name,
		column_default,
		column_comment,
		is_nullable,
		is_generated,
		is_identity
	FROM (
		%s
	) AS c`, tableQuery) // matviewQuery, tableQuery)

	filter := colFilter[info.Key]
	only := filter.Only
	except := filter.Except

	if len(only) > 0 || len(except) > 0 {
		query += " where "
	}

	if len(only) > 0 {
		query += fmt.Sprintf("c.column_name in (%s)", strmangle.Placeholders(true, len(only), 3, 1))
		for _, w := range only {
			args = append(args, w)
		}
	} else if len(except) > 0 {
		query += fmt.Sprintf("c.column_name not in (%s)", strmangle.Placeholders(true, len(except), 3, 1))
		for _, w := range except {
			args = append(args, w)
		}
	}

	query += ` order by c.ordinal_position;`

	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return "", "", nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var colName, colType, udtSchema, udtName, comment string
		var defaultValue, arrayType, domainName *string
		var nullable, generated, identity bool
		if err := rows.Scan(&colName, &colType, &udtSchema, &udtName, &arrayType, &domainName, &defaultValue, &comment, &nullable, &generated, &identity); err != nil {
			return "", "", nil, fmt.Errorf("unable to scan for table %s: %w", info.Key, err)
		}

		column := drivers.Column{
			Name:      colName,
			DBType:    colType,
			Comment:   comment,
			Nullable:  nullable,
			Generated: generated,
		}
		info := colInfo{
			UDTSchema: udtSchema,
			UDTName:   udtName,
		}

		if arrayType != nil {
			info.ArrType = *arrayType
		}

		if domainName != nil {
			column.DomainName = *domainName
		}

		if defaultValue != nil {
			column.Default = *defaultValue
		}

		if identity {
			column.Default = "IDENTITY"
		}

		// A generated column technically has a default value
		if generated && column.Default == "" {
			column.Default = "GENERATED"
		}

		// A nullable column can always default to NULL
		if nullable && column.Default == "" {
			column.Default = "NULL"
		}

		columns = append(columns, d.translateColumnType(column, info))
	}

	schema := info.Schema
	if schema == d.config.SharedSchema {
		schema = ""
	}

	return schema, info.Name, columns, nil
}

func (d *driver) loadEnums(ctx context.Context) error {
	if d.enums != nil {
		return nil
	}

	query := `SELECT pg_namespace.nspname AS schema, pg_type.typname AS name, array_agg(pg_enum.enumlabel order by pg_enum.enumsortorder) AS values
		FROM pg_type
		JOIN pg_enum ON pg_enum.enumtypid = pg_type.oid
		JOIN pg_namespace ON pg_namespace.oid = pg_type.typnamespace
		WHERE pg_namespace.nspname = ANY($1)
		GROUP BY schema, name`

	var err error
	d.enums, err = stdscan.All(
		ctx, d.conn,
		func(_ context.Context, _ []string) (scan.BeforeFunc, func(any) (Enum, error)) {
			return func(r *scan.Row) (any, error) {
					var e Enum
					r.ScheduleScan("schema", &e.Schema)
					r.ScheduleScan("name", &e.Name)
					r.ScheduleScan("values", &e.Values)
					return &e, nil
				}, func(a any) (Enum, error) {
					e := a.(*Enum)
					if e.Schema != "" && e.Schema != d.config.SharedSchema {
						e.Type = strmangle.TitleCase(e.Schema + "_" + e.Name)
					} else {
						e.Type = strmangle.TitleCase(e.Name)
					}

					return *e, nil
				}
		},
		query, d.config.Schemas,
	)
	if err != nil {
		return err
	}

	return nil
}
