package drivers

import "encoding/json"

// DBIndexes lists all indexes in the database schema keyed by table name
type DBIndexes[Extra any] map[string][]Index[Extra]

// Index represents an index in a table
type Index[Extra any] struct {
	Type    string        `yaml:"type" json:"type"`
	Name    string        `yaml:"name" json:"name"`
	Columns []IndexColumn `yaml:"columns" json:"columns"`
	Unique  bool          `yaml:"unique" json:"unique"`
	Comment string        `json:"comment" yaml:"comment"`
	Extra   Extra         `yaml:"extra" json:"extra"`
}

type IndexColumn struct {
	Name         string `yaml:"name" json:"name"`
	Desc         bool   `yaml:"desc" json:"desc"`
	IsExpression bool   `yaml:"is_expression" json:"is_expression"`
}

func (i Index[E]) HasExpressionColumn() bool {
	for _, c := range i.Columns {
		if c.IsExpression {
			return true
		}
	}
	return false
}

func (i Index[E]) NonExpressionColumns() []string {
	cols := make([]string, 0, len(i.Columns))
	for _, c := range i.Columns {
		if !c.IsExpression {
			cols = append(cols, c.Name)
		}
	}

	return cols
}

type DBConstraints[Extra any] struct {
	PKs     map[string]*Constraint[Extra]
	FKs     map[string][]ForeignKey[Extra]
	Uniques map[string][]Constraint[Extra]
	Checks  map[string][]Check[Extra]
}

type Constraints[Extra any] struct {
	Primary *Constraint[Extra]  `yaml:"primary" json:"primary"`
	Foreign []ForeignKey[Extra] `yaml:"foreign" json:"foreign"`
	Uniques []Constraint[Extra] `yaml:"uniques" json:"uniques"`
	Checks  []Check[Extra]      `yaml:"check" json:"check"`
}

// Constraint represents a constraint in a database
type Constraint[Extra any] struct {
	Name    string   `yaml:"name" json:"name"`
	Columns []string `yaml:"columns" json:"columns"`
	Comment string   `json:"comment" yaml:"comment"`
	Extra   Extra    `yaml:"extra" json:"extra"`
}

// ForeignKey represents a foreign key constraint in a database
type ForeignKey[Extra any] struct {
	Constraint[Extra] `yaml:",squash" json:"-"`
	ForeignTable      string   `yaml:"foreign_table" json:"foreign_table"`
	ForeignColumns    []string `yaml:"foreign_columns" json:"foreign_columns"`
}

func (f *ForeignKey[E]) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name           string   `json:"name"`
		Columns        []string `json:"columns"`
		ForeignTable   string   `json:"foreign_table"`
		ForeignColumns []string `json:"foreign_columns"`
		Comment        string   `json:"comment"`
		Extra          E        `json:"extra"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	f.Name = tmp.Name
	f.Columns = tmp.Columns
	f.ForeignTable = tmp.ForeignTable
	f.ForeignColumns = tmp.ForeignColumns
	f.Comment = tmp.Comment
	f.Extra = tmp.Extra

	return nil
}

func (f ForeignKey[E]) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Name           string   `json:"name"`
		Columns        []string `json:"columns"`
		ForeignTable   string   `json:"foreign_table"`
		ForeignColumns []string `json:"foreign_columns"`
		Comment        string   `json:"comment"`
		Extra          E        `json:"extra"`
	}{
		Name:           f.Name,
		Columns:        f.Columns,
		ForeignTable:   f.ForeignTable,
		ForeignColumns: f.ForeignColumns,
		Comment:        f.Comment,
		Extra:          f.Extra,
	}

	return json.Marshal(tmp)
}

// Check represents a check constraint in a database
type Check[Extra any] struct {
	Constraint[Extra] `yaml:",squash" json:"-"`
	Expression        string `yaml:"expression" json:"expression"`
}

func (c *Check[E]) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name       string   `json:"name"`
		Columns    []string `json:"columns"`
		Expression string   `json:"expression"`
		Comment    string   `json:"comment"`
		Extra      E        `json:"extra"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	c.Name = tmp.Name
	c.Columns = tmp.Columns
	c.Expression = tmp.Expression
	c.Comment = tmp.Comment
	c.Extra = tmp.Extra

	return nil
}

func (c Check[E]) MarshalJSON() ([]byte, error) {
	tmp := struct {
		Name       string   `json:"name"`
		Columns    []string `json:"columns"`
		Expression string   `json:"expression"`
		Comment    string   `json:"comment"`
		Extra      E        `json:"extra"`
	}{
		Name:       c.Name,
		Columns:    c.Columns,
		Expression: c.Expression,
		Comment:    c.Comment,
		Extra:      c.Extra,
	}

	return json.Marshal(tmp)
}
