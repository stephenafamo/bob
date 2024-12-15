package drivers

type QueryFolder struct {
	Path  string
	Files []QueryFile
}

type QueryFile struct {
	Path    string
	Queries []Query
}

type Query struct {
	Name        string `yaml:"name"`
	SQL         string `yaml:"raw"`
	RowName     string `yaml:"row_name"`
	GenerateRow bool   `yaml:"generate_row"`

	Columns []QueryArg `yaml:"columns"`
	Args    []QueryArg `yaml:"args"`
}

type QueryArg struct {
	Name     string `yaml:"name"`
	Nullable bool   `yaml:"nullable"`
	TypeName string `yaml:"type"`
	Refs     []Ref  `yaml:"refs"`
}

type Ref struct {
	Key    string `yaml:"key"`
	Column string `yaml:"column"`
}

type db interface {
	GetColumn(key string, col string) Column
}

func (c QueryArg) Type(db db) string {
	if len(c.Refs) == 0 {
		return c.TypeName
	}

	ref := db.GetColumn(c.Refs[0].Key, c.Refs[0].Column).Type

	for _, r := range c.Refs[1:] {
		if ref != db.GetColumn(r.Key, r.Column).Type {
			return c.TypeName
		}
	}

	return ref
}
