package drivers

// DBIndexes lists all indexes in the database schema keyed by table name
type DBIndexes map[string][]Index

// Index represents an index in a table
type Index struct {
	Name        string   `yaml:"name" json:"name"`
	Columns     []string `yaml:"columns" json:"columns"`
	Expressions []string `yaml:"expressions" json:"expressions"`
}
