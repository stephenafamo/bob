package drivers

import "encoding/json"

// Index represents an index in a table
type Index[Extra any] struct {
	Name        string   `yaml:"name" json:"name"`
	Columns     []string `yaml:"columns" json:"columns"`
	Expressions []string `yaml:"expressions" json:"expressions"`
	Extra       Extra    `yaml:"extra" json:"extra"`
}

// DBIndexes lists all indexes in the database schema keyed by table name
type DBIndexes[Extra any] map[string][]Index[Extra]

type Constraints[Extra any] struct {
	Primary *Constraint[Extra]  `yaml:"primary" json:"primary"`
	Foreign []ForeignKey[Extra] `yaml:"foreign" json:"foreign"`
	Uniques []Constraint[Extra] `yaml:"uniques" json:"uniques"`
}

type DBConstraints[Extra any] struct {
	PKs     map[string]*Constraint[Extra]
	FKs     map[string][]ForeignKey[Extra]
	Uniques map[string][]Constraint[Extra]
}

// Constraint represents a constraint in a database
type Constraint[Extra any] NamedColumnList[Extra]

type NamedColumnList[Extra any] struct {
	Name    string   `yaml:"name" json:"name"`
	Columns []string `yaml:"columns" json:"columns"`
	Extra   Extra    `yaml:"extra" json:"extra"`
}

// ForeignKey represents a foreign key constraint in a database
type ForeignKey[Extra any] struct {
	Constraint[Extra] `yaml:",inline" json:"-"`
	ForeignTable      string   `yaml:"foreign_table" json:"foreign_table"`
	ForeignColumns    []string `yaml:"foreign_columns" json:"foreign_columns"`
}

func (f *ForeignKey[E]) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name           string   `json:"name"`
		Columns        []string `json:"columns"`
		Extra          E        `json:"extra"`
		ForeignTable   string   `json:"foreign_table"`
		ForeignColumns []string `json:"foreign_columns"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	f.Name = tmp.Name
	f.Columns = tmp.Columns
	f.Extra = tmp.Extra
	f.ForeignTable = tmp.ForeignTable
	f.ForeignColumns = tmp.ForeignColumns

	return nil
}

func (f ForeignKey[E]) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Name           string   `json:"name"`
		Columns        []string `json:"columns"`
		Extra          E        `json:"extra"`
		ForeignTable   string   `json:"foreign_table"`
		ForeignColumns []string `json:"foreign_columns"`
	}{
		Name:           f.Name,
		Columns:        f.Columns,
		Extra:          f.Extra,
		ForeignTable:   f.ForeignTable,
		ForeignColumns: f.ForeignColumns,
	}

	return json.Marshal(tmp)
}
