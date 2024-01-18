package driver

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"ariga.io/atlas/schemahcl"
	"ariga.io/atlas/sql/mysql"
	"ariga.io/atlas/sql/postgres"
	"ariga.io/atlas/sql/schema"
	"ariga.io/atlas/sql/sqlite"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
	"github.com/volatiletech/strmangle"
)

type (
	Interface = drivers.Interface[any]
	DBInfo    = drivers.DBInfo[any]
	Config    struct {
		// What dialect to generate with
		// psql | mysql | sqlite
		Dialect string
		// Where the hcl files are
		Dir string
		// The name of this schema will not be included in the generated models
		// a context value can then be used to set the schema at runtime
		// useful for multi-tenant setups
		SharedSchema string `yaml:"shared_schema"`
		// List of tables that will be included. Others are ignored
		Only map[string][]string
		// List of tables that will be should be ignored. Others are included
		Except map[string][]string
		// Which UUID package to use (gofrs or google)
		UUIDPkg string `yaml:"uuid_pkg"`

		Output    string
		Pkgname   string
		NoFactory bool `yaml:"no_factory"`
	}
)

func New(config Config, fs fs.FS) Interface {
	if config.Dir == "" {
		config.Dir = "."
	}

	types := helpers.Types()

	switch config.UUIDPkg {
	case "google":
		types["uuid.UUID"] = drivers.Type{
			Imports: importers.List{`"github.com/google/uuid"`},
		}
	default:
		types["uuid.UUID"] = drivers.Type{
			Imports: importers.List{`"github.com/gofrs/uuid/v5"`},
		}
	}

	return &driver{
		config: config,
		fs:     fs,
		types:  types,
	}
}

// driver holds the database connection string and a handle
// to the database connection.
type driver struct {
	config Config
	fs     fs.FS
	enums  map[string]drivers.Enum
	types  drivers.Types
}

func (d *driver) Dialect() string {
	return d.config.Dialect
}

func (d *driver) Capabilities() drivers.Capabilities {
	return drivers.Capabilities{}
}

func (d *driver) Types() drivers.Types {
	return d.types
}

// Assemble all the information we need to provide back to the driver
func (d *driver) Assemble(ctx context.Context) (*DBInfo, error) {
	var err error
	var dbinfo *DBInfo
	var evalFunc schemahcl.EvalFunc
	switch d.config.Dialect {
	case "psql":
		evalFunc = postgres.EvalHCL
	case "mysql":
		evalFunc = mysql.EvalHCL
	case "sqlite":
		evalFunc = sqlite.EvalHCL
	case "":
		return nil, fmt.Errorf("dialect must be specified")
	default:
		return nil, fmt.Errorf("Unsupported dialect %q", d.config.Dialect)
	}

	parser, err := parseHCLPaths(d.fs)
	if err != nil {
		return nil, err
	}

	realm := &schema.Realm{}
	if err := evalFunc(parser, realm, nil); err != nil {
		return nil, err
	}

	if d.config.SharedSchema == "" {
		d.config.SharedSchema = realm.Schemas[0].Name
	}

	d.loadEnums(realm)
	dbinfo = &DBInfo{
		Enums:  d.getEnums(),
		Tables: d.tables(realm),
	}

	return dbinfo, err
}

func (d *driver) key(schema string, table string) string {
	key := table
	if schema != "" && schema != d.config.SharedSchema {
		key = schema + "." + table
	}

	return key
}

func (d *driver) schema(schema string) string {
	if schema == d.config.SharedSchema {
		return ""
	}

	return schema
}

func (d *driver) tables(realm *schema.Realm) []drivers.Table {
	tables := make([]drivers.Table, 0, len(realm.Schemas))

	tblFilter := drivers.ParseTableFilter(d.config.Only, d.config.Except)

	colFilter := drivers.ParseColumnFilter(d.tableNames(realm, drivers.Filter{
		Only:   tblFilter.Only,
		Except: tblFilter.Except,
	}), d.config.Only, d.config.Except)

	for _, schema := range realm.Schemas {
		for _, atlasTable := range schema.Tables {
			if drivers.Skip(atlasTable.Name, tblFilter.Only, tblFilter.Except) {
				continue
			}

			pk, uniques, fks := d.getKeys(atlasTable, colFilter)
			table := drivers.Table{
				Key:    d.key(schema.Name, atlasTable.Name),
				Schema: d.schema(schema.Name),
				Name:   atlasTable.Name,
				Constraints: drivers.Constraints{
					Primary: pk,
					Uniques: uniques,
					Foreign: fks,
				},
				Columns: d.tableColumns(atlasTable, colFilter),
			}
			tables = append(tables, table)
		}
	}

	return tables
}

func (d *driver) tableNames(realm *schema.Realm, tableFilter drivers.Filter) []string {
	names := make([]string, 0, len(realm.Schemas))

	for _, s := range realm.Schemas {
		for _, m := range s.Tables {
			key := d.key(s.Name, m.Name)
			if drivers.Skip(key, tableFilter.Only, tableFilter.Except) {
				continue
			}

			names = append(names, key)
		}
	}

	return names
}

func (d *driver) tableColumns(table *schema.Table, colFilter drivers.ColumnFilter) []drivers.Column {
	key := d.key(table.Schema.Name, table.Name)
	allfilter := colFilter["*"]
	filter := colFilter[key]
	include := append(allfilter.Only, filter.Only...)
	exclude := append(allfilter.Except, filter.Except...)

	columns := make([]drivers.Column, 0, len(table.Columns))
	for _, atlasCol := range table.Columns {
		if drivers.Skip(atlasCol.Name, include, exclude) {
			continue
		}

		var dbType bytes.Buffer
		// err := json.NewEncoder(&dbType).Encode(atlasCol.Type)
		// if err != nil {
		// return nil, err
		// }

		column := drivers.Column{
			Name:     atlasCol.Name,
			DBType:   strings.TrimSpace(dbType.String()),
			Nullable: atlasCol.Type.Null,
		}

		for _, a := range atlasCol.Attrs {
			// Get the column comment
			if attr, ok := a.(*schema.Comment); ok && attr != nil {
				column.Comment = attr.Text
			}

			// If the column has a generation expression
			if attr, ok := a.(*schema.GeneratedExpr); ok && attr != nil {
				column.Generated = true
			}
			// Postgres identity columns are generated
			if attr, ok := a.(*postgres.Identity); ok && attr != nil {
				column.Generated = true
			}

			// check for mysql autoincr columns
			if attr, ok := a.(*mysql.AutoIncrement); ok && attr != nil {
				column.AutoIncr = true
			}
			// check for sqlite autoincr columns
			if attr, ok := a.(*sqlite.AutoIncrement); ok && attr != nil {
				column.AutoIncr = true
			}
		}

		if atlasCol.Default != nil {
			column.Default = "DEFAULT"
		}

		// A generated column technically has a default value
		if column.Generated && column.Default == "" {
			column.Default = "GENERATED"
		}

		// A nullable column can always default to NULL
		if atlasCol.Type.Null && column.Default == "" {
			column.Default = "NULL"
		}

		column = d.translateColumnType(column, key, atlasCol.Type.Type)
		columns = append(columns, column)
	}

	return columns
}

//nolint:gocyclo
func (d *driver) translateColumnType(c drivers.Column, tableKey string, typ schema.Type) drivers.Column {
	switch t := typ.(type) {
	case *schema.BoolType:
		c.Type = "bool"

	case *schema.StringType:
		c.Type = "string"

	case *schema.BinaryType:
		c.Type = "[]byte"

	case *schema.IntegerType:
		switch t.T {
		case "tinyint":
			c.Type = "int8"
		case "smallint":
			c.Type = "int16"
		case "mediumint":
			c.Type = "int32"
		case "int", "integer":
			c.Type = "int32"
		case "bigint":
			c.Type = "int64"
		default:
			c.Type = "int"
		}
		if t.Unsigned {
			c.Type = "u" + c.Type
		}

	case *postgres.SerialType:
		switch t.T {
		case "smallserial", "serial2":
			c.Type = "int16"
		case "serial", "serial4":
			c.Type = "int32"
		case "bigserial", "serial8":
			c.Type = "int64"
		default:
			c.Type = "int"
		}

	case *schema.FloatType:
		switch t.T {
		case "float":
			c.Type = "float32"
			if t.Precision > 24 {
				c.Type = "float64"
			}
		case "real":
			c.Type = "float32"
		case "double", "double precision":
			c.Type = "float64"
		default:
			c.Type = "float64"
		}

	case *schema.TimeType:
		c.Type = "time.Time"

	case *schema.DecimalType:
		c.Type = "decimal.Decimal"

	case *schema.JSONType:
		c.Type = "types.JSON[json.RawMessage]"

	case *schema.UUIDType:
		c.Type = "uuid.UUID"

	case *schema.EnumType:
		enumName := t.T
		if d.config.Dialect == "mysql" {
			enumName = tableKey + "_" + c.Name
		}

		if enum, ok := d.enums[enumName]; ok {
			c.Type = enum.Type
		} else {
			c.Type = "string"
		}

	case *schema.SpatialType:
		if d.config.Dialect != "psql" {
			c.Type = "string"
			break
		}

		switch t.T {
		case "box":
			c.Type = "pgeo.Box"
		case "circle":
			c.Type = "pgeo.Circle"
		case "line":
			c.Type = "pgeo.Line"
		case "lseg":
			c.Type = "pgeo.Lseg"
		case "path":
			c.Type = "pgeo.Path"
		case "point":
			c.Type = "pgeo.Point"
		case "polygon":
			c.Type = "pgeo.Polygon"
		default:
			c.Type = "string"
		}

	case *postgres.ArrayType:
		switch t.Type.(type) {
		case *schema.BoolType:
			c.Type = "pq.BoolArray"
		case *schema.BinaryType:
			c.Type = "pq.ByteaArray"
		case *schema.StringType:
			c.Type = "pq.StringArray"
		case *schema.FloatType:
			c.Type = "pq.Float64Array"
		case *schema.IntegerType, *postgres.SerialType:
			c.Type = "pq.Int64Array"
		case *schema.EnumType:
			c2 := d.translateColumnType(c, tableKey, t.Type)
			c.Type = helpers.AddPgEnumArrayType(d.types, c2.Type)
		default:
			c2 := d.translateColumnType(c, tableKey, t.Type)
			c.Type = helpers.AddPgGenericArrayType(d.types, c2.Type)
		}

	default:
		c.Type = "string"
	}

	return c
}

func (d *driver) getKeys(table *schema.Table, colFilter drivers.ColumnFilter) (*drivers.PrimaryKey, []drivers.Constraint, []drivers.ForeignKey) {
	var pk *drivers.PrimaryKey
	var uniques []drivers.Constraint
	var fks []drivers.ForeignKey

	filter := colFilter[d.key(table.Schema.Name, table.Name)]
	only := filter.Only
	except := filter.Except

	// If it is a composite primary key defined on the model
	if table.PrimaryKey != nil && len(table.PrimaryKey.Parts) > 0 {
		shouldSkip := false
		cols := make([]string, len(table.PrimaryKey.Parts))

		for i, p := range table.PrimaryKey.Parts {
			if p.C == nil || drivers.Skip(p.C.Name, only, except) {
				shouldSkip = true
			}
			cols[i] = p.C.Name
		}

		if !shouldSkip {
			pkName := table.PrimaryKey.Name
			if pkName == "" {
				pkName = "pk_" + table.Name
			}
			pk = &drivers.Constraint{
				Name:    pkName,
				Columns: cols,
			}
		}
	}

	for _, unique := range table.Indexes {
		if !unique.Unique {
			continue
		}

		shouldSkip := false
		cols := make([]string, len(unique.Parts))

		for i, f := range unique.Parts {
			if f.X != nil || drivers.Skip(f.C.Name, only, except) {
				shouldSkip = true
			}

			cols[i] = f.C.Name
		}

		if !shouldSkip {
			keyName := unique.Name
			if keyName == "" {
				keyName = fmt.Sprintf("unique_%s_%s", table.Name, strings.Join(cols, "_"))
			}

			uniques = append(uniques, drivers.Constraint{
				Name:    keyName,
				Columns: cols,
			})
		}
	}

	for i, fk := range table.ForeignKeys {
		shouldSkip := false

		ftableKey := d.key(fk.RefTable.Schema.Name, fk.RefTable.Name)
		fFilter := colFilter[ftableKey]

		cols := make([]string, len(fk.Columns))
		fcols := make([]string, len(fk.RefColumns))

		for i := range fk.Columns {
			cols[i] = fk.Columns[i].Name
			fcols[i] = fk.RefColumns[i].Name

			if drivers.Skip(cols[i], only, except) ||
				drivers.Skip(fcols[i], fFilter.Only, fFilter.Except) {
				shouldSkip = true
			}
		}

		if !shouldSkip {
			keyName := fmt.Sprintf("fk_%s_%d", table.Name, i)

			fks = append(fks, drivers.ForeignKey{
				Name:           keyName,
				Columns:        cols,
				ForeignTable:   ftableKey,
				ForeignColumns: fcols,
			})
		}
	}

	return pk, uniques, fks
}

func (d *driver) loadEnums(realm *schema.Realm) {
	if d.enums != nil {
		return
	}
	d.enums = map[string]drivers.Enum{}

	for _, s := range realm.Schemas {
		for _, t := range s.Tables {
			tableKey := d.key(t.Schema.Name, t.Name)
			for _, c := range t.Columns {
				enum, ok := c.Type.Type.(*schema.EnumType)
				if !ok {
					continue
				}
				enumName := enum.T
				if d.config.Dialect == "mysql" {
					enumName = tableKey + "_" + c.Name
				}

				d.enums[enumName] = drivers.Enum{
					Type:   strmangle.TitleCase(enumName),
					Values: enum.Values,
				}
			}
		}
	}
}

func (p *driver) getEnums() []drivers.Enum {
	enums := make([]drivers.Enum, 0, len(p.enums))
	for _, e := range p.enums {
		enums = append(enums, e)
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Type < enums[j].Type
	})

	return enums
}
