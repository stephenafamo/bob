package mocks

import (
	"context"

	"github.com/stephenafamo/bob/gen/drivers"
)

// MockDriver is a mock implementation of the bdb driver Interface
type MockDriver struct{}

// Assemble the DBInfo
func (m *MockDriver) Assemble(ctx context.Context) (*drivers.DBInfo[any], error) {
	var err error
	dbinfo := &drivers.DBInfo[any]{}

	defer func() {
		if r := recover(); r != nil && err == nil {
			dbinfo = nil
			err = r.(error)
		}
	}()

	dbinfo.Tables, err = drivers.Tables(ctx, m, 1, nil, []string{"hangars"})
	if err != nil {
		return nil, err
	}

	return dbinfo, err
}

// TableNames returns a list of mock table names
func (m *MockDriver) TablesInfo(_ context.Context, filter drivers.Filter) (drivers.TablesInfo, error) {
	return []drivers.TableInfo{
		{Key: "pilots", Name: "pilots"},
		{Key: "schema.jets", Schema: "schema", Name: "jets"},
		{Key: "airports", Name: "airports"},
		{Key: "licenses", Name: "licenses"},
		{Key: "hangars", Name: "hangars"},
		{Key: "languages", Name: "languages"},
		{Key: "pilot_languages", Name: "pilot_languages"},
	}, nil
}

func (m *MockDriver) ViewsInfo(_ context.Context, filter drivers.Filter) (drivers.TablesInfo, error) {
	return []drivers.TableInfo{
		{Name: "pilots_with_jets"},
	}, nil
}

// Columns returns a list of mock columns
func (m *MockDriver) TableColumns(_ context.Context, info drivers.TableInfo, filter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
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
		"schema.jets": {
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
	}[info.Key] {
		cols = append(cols, m.translateColumnType(col))
	}

	return info.Schema, info.Name, cols, nil
}

// ViewColumns returns a list of mock columns
func (m *MockDriver) ViewColumns(_ context.Context, info drivers.TableInfo, filter drivers.ColumnFilter) (string, string, []drivers.Column, error) {
	var cols []drivers.Column //nolint:prealloc

	for _, col := range map[string][]drivers.Column{
		"pilots_with_jets": {
			{Name: "pilot_name", Type: "string", DBType: "character"},
			{Name: "jet_name", Type: "string", DBType: "character", Nullable: false},
			{Name: "jet_color", Type: "string", DBType: "character", Nullable: true},
		},
	}[info.Key] {
		cols = append(cols, m.translateColumnType(col))
	}

	return info.Schema, info.Name, cols, nil
}

// ForeignKeyInfo returns a list of mock foreignkeys
func (m *MockDriver) Constraints(context.Context, drivers.ColumnFilter) (drivers.DBConstraints, error) {
	return drivers.DBConstraints{
		PKs: map[string]*drivers.PrimaryKey{
			"schema.jets":     {Name: "jets_pkey", Columns: []string{"id"}},
			"airports":        {Name: "airports_pkey", Columns: []string{"id"}},
			"pilots":          {Name: "pilots_pkey", Columns: []string{"id"}},
			"languages":       {Name: "languages_pkey", Columns: []string{"id"}},
			"pilot_languages": {Name: "pilot_languages_pkey", Columns: []string{"pilot_id", "language_id"}},
			"licenses":        {Name: "licenses_pkey", Columns: []string{"id"}},
			"hangars":         {Name: "hangars_pkey", Columns: []string{"id"}},
		},
		FKs: map[string][]drivers.ForeignKey{
			"schema.jets": {
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
