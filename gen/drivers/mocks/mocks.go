package mocks

import (
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
)

// MockDriver is a mock implementation of the bdb driver Interface
type MockDriver struct{}

// Assemble the DBInfo
func (m *MockDriver) Assemble() (*drivers.DBInfo[any], error) {
	var err error
	dbinfo := &drivers.DBInfo[any]{}

	defer func() {
		if r := recover(); r != nil && err == nil {
			dbinfo = nil
			err = r.(error)
		}
	}()

	dbinfo.Tables, err = drivers.Tables(m, 1, nil, []string{"hangars"})
	if err != nil {
		return nil, err
	}

	return dbinfo, err
}

// TableNames returns a list of mock table names
func (m *MockDriver) TableNames(filter drivers.Filter) ([]string, error) {
	if len(filter.Include) > 0 {
		return filter.Include, nil
	}
	tables := []string{"pilots", "jets", "airports", "licenses", "hangars", "languages", "pilot_languages"}
	return strmangle.SetComplement(tables, filter.Exclude), nil
}

func (m *MockDriver) ViewNames(filter drivers.Filter) ([]string, error) {
	if len(filter.Include) > 0 {
		return filter.Include, nil
	}
	tables := []string{"pilots_with_jets"}
	return strmangle.SetComplement(tables, filter.Exclude), nil
}

// Columns returns a list of mock columns
func (m *MockDriver) TableColumns(tableName string, filter drivers.ColumnFilter) ([]drivers.Column, error) {
	var cols []drivers.Column //nolint:prealloc

	for _, col := range map[string][]drivers.Column{
		"pilots": {
			{Name: "id", Type: "int", DBType: "integer"},
			{Name: "name", Type: "string", DBType: "character"},
		},
		"airports": {
			{Name: "id", Type: "int", DBType: "integer"},
			{Name: "size", Type: "int", DBType: "integer", Nullable: true},
		},
		"jets": {
			{Name: "id", Type: "int", DBType: "integer"},
			{Name: "pilot_id", Type: "int", DBType: "integer", Nullable: true, Unique: true},
			{Name: "airport_id", Type: "int", DBType: "integer"},
			{Name: "name", Type: "string", DBType: "character", Nullable: false},
			{Name: "color", Type: "string", DBType: "character", Nullable: true},
			{Name: "uuid", Type: "string", DBType: "uuid", Nullable: true},
			{Name: "identifier", Type: "string", DBType: "uuid", Nullable: false},
			{Name: "cargo", Type: "[]byte", DBType: "bytea", Nullable: false},
			{Name: "manifest", Type: "[]byte", DBType: "bytea", Nullable: true, Unique: true},
		},
		"licenses": {
			{Name: "id", Type: "int", DBType: "integer"},
			{Name: "pilot_id", Type: "int", DBType: "integer"},
		},
		"hangars": {
			{Name: "id", Type: "int", DBType: "integer"},
			{Name: "name", Type: "string", DBType: "character", Nullable: true, Unique: true},
		},
		"languages": {
			{Name: "id", Type: "int", DBType: "integer"},
			{Name: "language", Type: "string", DBType: "character", Nullable: false, Unique: true},
		},
		"pilot_languages": {
			{Name: "pilot_id", Type: "int", DBType: "integer"},
			{Name: "language_id", Type: "int", DBType: "integer"},
		},
	}[tableName] {
		cols = append(cols, m.translateColumnType(col))
	}

	return cols, nil
}

// ViewColumns returns a list of mock columns
func (m *MockDriver) ViewColumns(viewName string, filter drivers.ColumnFilter) ([]drivers.Column, error) {
	return map[string][]drivers.Column{
		"pilots_with_jets": {
			{Name: "pilot_name", Type: "string", DBType: "character"},
			{Name: "jet_name", Type: "string", DBType: "character", Nullable: false},
			{Name: "jet_color", Type: "string", DBType: "character", Nullable: true},
		},
	}[viewName], nil
}

// ForeignKeyInfo returns a list of mock foreignkeys
func (m *MockDriver) Constraints(drivers.ColumnFilter) (drivers.DBConstraints, error) {
	return drivers.DBConstraints{
		PKs: map[string]*drivers.PrimaryKey{
			"jets":            {Name: "jets_pkey", Columns: []string{"id"}},
			"airports":        {Name: "airports_pkey", Columns: []string{"id"}},
			"pilots":          {Name: "pilots_pkey", Columns: []string{"id"}},
			"languages":       {Name: "languages_pkey", Columns: []string{"id"}},
			"pilot_languages": {Name: "pilot_languages_pkey", Columns: []string{"pilot_id", "language_id"}},
			"licenses":        {Name: "licenses_pkey", Columns: []string{"id"}},
			"hangars":         {Name: "hangars_pkey", Columns: []string{"id"}},
		},
		FKs: map[string][]drivers.ForeignKey{
			"jets": {
				{
					Constraint: drivers.Constraint{
						Name:    "jets_pilot_id_fk",
						Columns: []string{"pilot_id"},
					},
					ForeignTable:   "pilots",
					ForeignColumns: []string{"id"},
				},
				{
					Constraint: drivers.Constraint{
						Name:    "jets_airport_id_fk",
						Columns: []string{"airport_id"},
					},
					ForeignTable:   "airports",
					ForeignColumns: []string{"id"},
				},
			},
			"licenses": {
				{
					Constraint: drivers.Constraint{
						Name:    "licenses_pilot_id_fk",
						Columns: []string{"pilot_id"},
					},
					ForeignTable:   "pilots",
					ForeignColumns: []string{"id"},
				},
			},
			"pilot_languages": {
				{
					Constraint: drivers.Constraint{
						Name:    "pilot_id_fk",
						Columns: []string{"pilot_id"},
					},
					ForeignTable:   "pilots",
					ForeignColumns: []string{"id"},
				},
				{
					Constraint: drivers.Constraint{
						Name:    "language_id_fk",
						Columns: []string{"language_id"},
					},
					ForeignTable:   "languages",
					ForeignColumns: []string{"id"},
				},
			},
		},
	}, nil
}

// translateColumnType converts a column to its type
func (m *MockDriver) translateColumnType(c drivers.Column) drivers.Column {
	switch c.DBType {
	case "bigint", "bigserial":
		c.Type = "int64"
	case "integer", "serial":
		c.Type = "int"
	case "smallint", "smallserial":
		c.Type = "int16"
	case "decimal", "numeric", "double precision":
		c.Type = "float64"
	case `"char"`:
		c.Type = "types.Byte"
	case "bytea":
		c.Type = "[]byte"
	case "boolean":
		c.Type = "bool"
	case "date", "time", "timestamp without time zone", "timestamp with time zone":
		c.Type = "time.Time"
	default:
		c.Type = "string"
	}

	return c
}

// PrimaryKeyInfo returns mock primary key info for the passed in table name
func (m *MockDriver) PrimaryKeyInfo(schema, tableName string) (*drivers.PrimaryKey, error) {
	return map[string]*drivers.PrimaryKey{
		"pilots": {
			Name:    "pilot_id_pkey",
			Columns: []string{"id"},
		},
		"airports": {
			Name:    "airport_id_pkey",
			Columns: []string{"id"},
		},
		"jets": {
			Name:    "jet_id_pkey",
			Columns: []string{"id"},
		},
		"licenses": {
			Name:    "license_id_pkey",
			Columns: []string{"id"},
		},
		"hangars": {
			Name:    "hangar_id_pkey",
			Columns: []string{"id"},
		},
		"languages": {
			Name:    "language_id_pkey",
			Columns: []string{"id"},
		},
		"pilot_languages": {
			Name:    "pilot_languages_pkey",
			Columns: []string{"pilot_id", "language_id"},
		},
	}[tableName], nil
}

// Open mimics a database open call and returns nil for no error
func (m *MockDriver) Open() error { return nil }

// Close mimics a database close call
func (m *MockDriver) Close() {}
