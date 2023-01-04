package driver

import (
	"embed"
	"fmt"
	"io/fs"
	"sort"

	"github.com/iancoleman/strcase"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
	"github.com/takuoki/gocase"
)

//go:embed templates
var templates embed.FS

//nolint:gochecknoglobals
var (
	ModelTemplates, _   = fs.Sub(templates, "templates/models")
	FactoryTemplates, _ = fs.Sub(templates, "templates/factory")
)

type (
	Interface = drivers.Interface[Extra]
	DBInfo    = drivers.DBInfo[Extra]
	Provider  struct {
		DriverName      string
		DriverPkg       string
		DriverSource    string
		DriverENVSource string
	}
	Extra struct {
		Provider Provider
		Enums    []Enum
	}
	Config struct {
		Provider    Provider
		Datamodel   Datamodel
		Schema      string
		Includes    []string
		Excludes    []string
		Concurrency int
	}
)

func New(config Config) Interface {
	return &Driver{config: config}
}

// Driver holds the database connection string and a handle
// to the database connection.
type Driver struct {
	config Config
	enums  map[string]Enum
}

// Assemble all the information we need to provide back to the driver
func (p *Driver) Assemble() (*DBInfo, error) {
	var dbinfo *DBInfo
	var err error

	defer func() {
		if r := recover(); r != nil && err == nil {
			dbinfo = nil
			err = r.(error)
		}
	}()

	dbinfo = &DBInfo{Schema: p.config.Schema}

	// drivers.Tables call translateColumnType which uses Enums
	p.loadEnums()

	dbinfo.Tables, err = drivers.Tables(p, p.config.Concurrency, p.config.Includes, p.config.Excludes)
	if err != nil {
		return nil, err
	}

	dbinfo.ExtraInfo.Provider = p.config.Provider
	dbinfo.ExtraInfo.Enums, err = p.Enums(p.config.Schema)
	if err != nil {
		return nil, err
	}

	return dbinfo, err
}

// TableNames connects to the postgres database and
// retrieves all table names from the information_schema where the
// table schema is schema. It uses a whitelist and blacklist.
func (d *Driver) TableNames(tableFilter drivers.Filter) ([]string, error) {
	models := d.config.Datamodel.Models
	names := make([]string, 0, len(models))

	for _, m := range models {
		if skip(m.Name, tableFilter.Include, tableFilter.Exclude) {
			continue
		}

		names = append(names, m.Name)
	}

	return names, nil
}

// ViewNames connects to the postgres database and
// retrieves all view names from the information_schema where the
// view schema is schema. It uses a whitelist and blacklist.
func (p *Driver) ViewNames(tableFilter drivers.Filter) ([]string, error) {
	return nil, nil
}

func (p *Driver) loadEnums() {
	if p.enums != nil {
		return
	}
	p.enums = map[string]Enum{}

	enums := p.config.Datamodel.Enums
	for _, enum := range enums {
		name := enum.Name
		values := make([]string, len(enum.Values))
		for i, val := range enum.Values {
			values[i] = val.Name
		}

		p.enums[name] = Enum{
			Name:   name,
			Type:   gocase.To(strcase.ToCamel(enum.Name)),
			Values: values,
		}
	}
}

type Enum struct {
	Name   string
	Type   string
	Values []string
}

func (p *Driver) Enums(schema string) ([]Enum, error) {
	enums := make([]Enum, 0, len(p.enums))
	for _, e := range p.enums {
		enums = append(enums, e)
	}

	sort.Slice(enums, func(i, j int) bool {
		return enums[i].Name < enums[j].Name
	})

	return enums, nil
}

func (p *Driver) ViewColumns(tableName string, filter drivers.ColumnFilter) ([]drivers.Column, error) {
	return p.TableColumns(tableName, filter)
}

// TableColumns takes a table name and attempts to retrieve the table information
// from the database information_schema.columns. It retrieves the column names
// and column types and returns those as a []Column after translateColumnType()
// converts the SQL types to Go types, for example: "varchar" to "string"
func (d *Driver) TableColumns(tableName string, colFilter drivers.ColumnFilter) ([]drivers.Column, error) {
	var model Model
	for _, m := range d.config.Datamodel.Models {
		if m.Name == tableName {
			model = m
			break
		}
	}

	allfilter := colFilter["*"]
	filter := colFilter[tableName]
	include := append(allfilter.Include, filter.Include...)
	exclude := append(allfilter.Exclude, filter.Exclude...)

	columns := make([]drivers.Column, 0, len(model.Fields))
	for _, field := range model.Fields {
		if skip(field.Name, include, exclude) {
			continue
		}

		if field.Kind == FieldKindObject {
			continue
		}

		column := drivers.Column{
			Name:      field.Name,
			DBType:    field.Type,
			Comment:   field.Documentation,
			Nullable:  !field.IsRequired,
			Generated: field.IsGenerated,
			Unique:    field.IsID || field.IsUnique,
		}

		if field.IsList {
			column.ArrType = column.DBType + "[]"
		}

		if field.HasDefaultValue {
			column.Default = "DEFAULT"
		}

		// A generated column technically has a default value
		if column.Generated && column.Default == "" {
			column.Default = "GENERATED"
		}

		// A nullable column can always default to NULL
		if !field.IsRequired && column.Default == "" {
			column.Default = "NULL"
		}

		columns = append(columns, d.translateColumnType(column))
	}

	return columns, nil
}

func (d *Driver) translateColumnType(c drivers.Column) drivers.Column {
	switch c.ArrType == "" {
	case true: // not an array
		switch c.DBType {
		case "String":
			c.Type = "string"
		case "Boolean":
			c.Type = "bool"
		case "Int":
			c.Type = "int"
		case "BigInt":
			c.Type = "int64"
		case "Float":
			c.Type = "float64"
		case "Bytes":
			c.Type = "[]byte"
		case "Decimal":
			c.Type = "decimal.Decimal"
		case "DateTime":
			c.Type = "time.Time"
		case "Json":
			c.Type = "types.JSON[json.RawMessage]"
		default:
			if enum, ok := d.enums[c.DBType]; ok {
				c.Type = enum.Type
			} else {
				c.Type = "string"
			}
		}

	case false: // Is an array
		switch c.DBType {
		case "String":
			c.Type = "pq.StringArray"
		case "Boolean":
			c.Type = "pq.BoolArray"
		case "Int", "BigInt":
			c.Type = "pq.Int64Array"
		case "Float":
			c.Type = "pq.Float64Array"
		case "Bytes":
			c.Type = "pq.ByteaArray"
		case "Decimal":
			c.Type = "parray.GenericArray[decimal.Decimal]"
			c.Imports = append(c.Imports, typMap["parray"]...)
			c.Imports = append(c.Imports, typMap["decimal.Decimal"]...)
		case "DateTime":
			c.Type = "parray.GenericArray[time.Time]"
			c.Imports = append(c.Imports, typMap["parray"]...)
			c.Imports = append(c.Imports, typMap["time.Time"]...)
		case "Json":
			c.Type = "parray.GenericArray[types.JSON[json.RawMessage]]"
			c.Imports = append(c.Imports, typMap["parray"]...)
			c.Imports = append(c.Imports, typMap["time.Time"]...)
		default:
			if enum, ok := d.enums[c.DBType]; ok {
				c.Imports = append(c.Imports, typMap["parray"]...)
				c.Type = fmt.Sprintf("parray.EnumArray[%s]", enum.Type)
			} else {
				c.Type = "pq.StringArray"
			}
		}
	}

	// fallback to other drivers?
	c.Imports = append(c.Imports, typMap[c.Type]...)
	return c
}

//nolint:gochecknoglobals
var typMap = map[string]importers.List{
	"time.Time":                   {`"time"`},
	"types.JSON[json.RawMessage]": {`"encoding/json"`, `"github.com/stephenafamo/bob/types"`},
	"decimal.Decimal":             {`"github.com/shopspring/decimal"`},
	"types.HStore":                {`"github.com/stephenafamo/bob/types"`},
	"pgeo.Point":                  {`"github.com/saulortega/pgeo"`},
	"pgeo.Line":                   {`"github.com/saulortega/pgeo"`},
	"pgeo.Lseg":                   {`"github.com/saulortega/pgeo"`},
	"pgeo.Box":                    {`"github.com/saulortega/pgeo"`},
	"pgeo.Path":                   {`"github.com/saulortega/pgeo"`},
	"pgeo.Polygon":                {`"github.com/saulortega/pgeo"`},
	"pq.ByteaArray":               {`"github.com/lib/pq"`},
	"pq.Int64Array":               {`"github.com/lib/pq"`},
	"pq.Float64Array":             {`"github.com/lib/pq"`},
	"pq.BoolArray":                {`"github.com/lib/pq"`},
	"pq.StringArray":              {`"github.com/lib/pq"`},
	"parray":                      {`"github.com/stephenafamo/bob/types/parray"`},
}

func skip(name string, include, exclude []string) bool {
	switch {
	case len(include) > 0:
		for _, i := range include {
			if i == name {
				return false
			}
		}
		return true

	case len(exclude) > 0:
		for _, i := range exclude {
			if i == name {
				return true
			}
		}
		return false

	default:
		return false
	}
}
