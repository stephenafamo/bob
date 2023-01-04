package driver

// FieldKind describes a scalar, object or enum.
type FieldKind string

// FieldKind values
const (
	FieldKindScalar FieldKind = "scalar"
	FieldKindObject FieldKind = "object"
	FieldKindEnum   FieldKind = "enum"
)

// Document describes the root of the AST.
type Document struct {
	Datamodel Datamodel `json:"datamodel"`
}

type Datamodel struct {
	Models []Model `json:"models"`
	Enums  []struct {
		Name   string      `json:"name"`
		Values []EnumValue `json:"values"`
		// DBName (optional)
		DBName string `json:"dBName"`
	} `json:"enums"`
}

type PrimaryKey struct {
	Name   string   `json:"name"`
	Fields []string `json:"fields"`
}

type UniqueIndex struct {
	InternalName string   `json:"name"`
	Fields       []string `json:"fields"`
}

// Model describes a Prisma type model, which usually maps to a database table or collection.
type Model struct {
	// Name describes the singular name of the model.
	Name       string `json:"name"`
	IsEmbedded bool   `json:"isEmbedded"`
	// DBName (optional)
	DBName        string        `json:"dbName"`
	Fields        []Field       `json:"fields"`
	UniqueIndexes []UniqueIndex `json:"uniqueIndexes"`
	PrimaryKey    PrimaryKey    `json:"primaryKey"`
}

// Field describes properties of a single model field.
type Field struct {
	Kind       FieldKind `json:"kind"`
	Name       string    `json:"name"`
	IsRequired bool      `json:"isRequired"`
	IsList     bool      `json:"isList"`
	IsUnique   bool      `json:"isUnique"`
	IsReadOnly bool      `json:"isReadOnly"`
	IsID       bool      `json:"isId"`
	Type       string    `json:"type"`
	// DBName (optional)
	DBName        string `json:"dBName"`
	IsGenerated   bool   `json:"isGenerated"`
	IsUpdatedAt   bool   `json:"isUpdatedAt"`
	Documentation string `json:"documentation"`
	// RelationFromFields (optional)
	RelationFromFields []string `json:"relationFromFields"`
	// RelationToFields (optional)
	RelationToFields []string `json:"relationToFields"`
	// RelationOnDelete (optional)
	RelationOnDelete string `json:"relationOnDelete"`
	// RelationName (optional)
	RelationName string `json:"relationName"`
	// HasDefaultValue
	HasDefaultValue bool `json:"hasDefaultValue"`
}

// EnumValue contains detailed information about an enum type.
type EnumValue struct {
	Name string `json:"name"`
	// DBName (optional)
	DBName string `json:"dBName"`
}
