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
	Name    string   `json:"name"`
	Columns []string `json:"columns"`
}

// ForeignKey represents a foreign key constraint in a database
type ForeignKey struct {
	Constraint
	ForeignTable   string   `json:"foreign_table"`
	ForeignColumns []string `json:"foreign_columns"`
}
