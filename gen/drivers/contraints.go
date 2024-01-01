package drivers

type DBConstraints struct {
	PKs     map[string]*PrimaryKey
	FKs     map[string][]ForeignKey
	Uniques map[string][]Constraint
}

// PrimaryKey represents a primary key constraint in a database
type PrimaryKey = Constraint

// Constraint represents a primary key constraint in a database
type Constraint struct {
	Name    string   `yaml:"name" json:"name"`
	Columns []string `yaml:"columns" json:"columns"`
}

// ForeignKey represents a foreign key constraint in a database
type ForeignKey struct {
	Name           string   `yaml:"name" json:"name"`
	Columns        []string `yaml:"columns" json:"columns"`
	ForeignTable   string   `yaml:"foreign_table" json:"foreign_table"`
	ForeignColumns []string `yaml:"foreign_columns" json:"foreign_columns"`
}
