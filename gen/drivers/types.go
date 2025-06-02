package drivers

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/stephenafamo/bob/gen/language"
)

type Type struct {
	// If this type is an alias of another type
	// this is useful to have custom randomization for a type e.g. xml
	// NOTE: If an alias is set,
	// the only other relevant fields are RandomExpr and RandomExprImports
	AliasOf string `yaml:"alias_of"`
	// Imports needed for the type
	Imports []string `yaml:"imports"`
	// Any other types that this type depends on
	DependsOn []string `yaml:"depends_on"`
	// To be used in factory.random_type
	// Use TYPE to reference the name of the type
	// * A variable `f` of type `faker.Faker` is available
	// * Another variable `limits` which is a slice of strings with any limits
	//   for example, a VARCHAR(255) would have limits = ["255"]
	//   another example, a DECIMAL(10,2) would have limits = ["10", "2"]
	RandomExpr string `yaml:"random_expr"`
	// Additional imports for the randomize expression
	RandomExprImports []string `yaml:"random_expr_imports"`

	// CompareExpr is used to compare two values of this type
	// if not provided, == is used
	// Use AAA and BBB as placeholders for the two values
	CompareExpr string `yaml:"compare_expr"`
	// Imports needed for the compare expression
	CompareExprImports []string `yaml:"compare_expr_imports"`

	// NullType configures the type to use for null values of this type
	// if not provided, the type is wrapped in `sql.Null[T]`
	// the method of creating a null type can be customized
	NullType NullType `yaml:"null_type"`

	// Set this to true if the randomization should not be tested
	// this is useful for low-cardinality types like bool
	NoRandomizationTest bool `yaml:"no_randomization_test"`
	// Set this to true if the test to see if the type implements
	// the scanner and valuer interfaces should be skipped
	// this is useful for types that are based on a primitive type
	NoScannerValuerTest bool `yaml:"no_scanner_valuer_test"`
}

type NullType struct {
	// Name is the type to use for null values of this type
	Name string `yaml:"name"`
	// ValidExpr is used to check if a value of this type IS NOT NULL
	// e.g. `SRC.Valid` for sql.Null[T] or `SRC.IsSet()` for null.Val[T]
	ValidExpr string `yaml:"valid_expr"`
	// FromNullExpr is used to convert a null value of this type
	// to a non-null value, if not provided it is assigned directly
	FromNullExpr string `yaml:"from_null_expr"`
	// Imports needed for the from null expression
	FromNullExprImports []string `yaml:"from_null_expr_imports"`
	// ToNullExpr is used to convert a non-null value of this type
	// to a null value, if not provided it is assigned directly
	ToNullExpr string `yaml:"to_null_expr"`
	// Imports needed for the to null expression
	ToNullExprImports []string `yaml:"to_null_expr_imports"`
}

type NullTypeCreator interface {
	// NullType returns the type to use for null values of the given type
	NullType(string) NullType
	// NullTypeImport
	NullTypeImports(Type) []string
}

type Types struct {
	registered  map[string]Type
	nullCreator NullTypeCreator
}

func (t *Types) SetNullTypeCreator(creator NullTypeCreator) {
	t.nullCreator = creator
}

func (t Types) GetNullCreator() NullTypeCreator {
	return t.nullCreator
}

func (t Types) Contains(name string) bool {
	_, ok := t.registered[name]
	return ok
}

func (t Types) Index(name string) Type {
	return t.registered[name]
}

func (t *Types) Register(name string, typedef Type) {
	if t.registered == nil {
		t.registered = make(map[string]Type)
	}
	t.registered[name] = typedef
}

func (t *Types) RegisterAll(m map[string]Type) {
	if t.registered == nil {
		t.registered = make(map[string]Type)
	}
	maps.Copy(t.registered, m)
}

func (t Types) Get(curr string, i language.Importer, namedType string) string {
	name, def := t.GetNameAndDef(curr, namedType)
	i.ImportList(def.Imports)
	return name
}

func (t Types) GetNullable(curr string, i language.Importer, namedType string, null bool) string {
	name, def := t.GetNameAndDef(curr, namedType)
	if !null {
		i.ImportList(def.Imports)
		return name
	}

	nullTyp, imports := t.GetNullTypeWithImports(curr, namedType)
	i.ImportList(imports)
	return nullTyp.Name
}

func (t Types) GetWithoutImporting(curr string, namedType string) string {
	name, _ := t.GetNameAndDef(curr, namedType)
	return name
}

func (t Types) GetNameAndDef(curr string, namedType string) (string, Type) {
	var ok bool
	typedef := Type{AliasOf: namedType}

	for typedef.AliasOf != "" {
		namedType = typedef.AliasOf
		typedef, ok = t.registered[namedType]
		if !ok {
			return namedType, Type{}
		}
	}

	if len(typedef.Imports) > 0 && strings.HasSuffix(typedef.Imports[0], `"`+curr+`"`) {
		_, namedType, _ = strings.Cut(namedType, ".")
	}

	return namedType, typedef
}

func (t Types) GetNullType(currentPkg string, forType string) NullType {
	typ, _ := t.GetNullTypeWithImports(currentPkg, forType)
	return typ
}

func (t Types) GetNullTypeWithImports(currentPkg string, forType string) (NullType, []string) {
	name, def := t.GetNameAndDef(currentPkg, forType)

	if def.NullType.Name != "" {
		nullType := def.NullType
		if nullType.FromNullExpr == "" {
			nullType.FromNullExpr = "SRC"
		}
		if nullType.ToNullExpr == "" {
			nullType.ToNullExpr = "SRC"
		}

		name, _ := t.GetNameAndDef(currentPkg, nullType.Name)
		def.NullType.Name = name

		return def.NullType, t.registered[def.NullType.Name].Imports
	}

	if t.nullCreator == nil {
		t.nullCreator = DatabaseSqlNull{}
	}

	return t.nullCreator.NullType(name), t.nullCreator.NullTypeImports(def)
}

func (t Types) GetNullTypeValid(currentPkg string, forType string, varName string) string {
	colTyp, _ := t.GetNameAndDef(currentPkg, forType)
	nullTyp, _ := t.GetNullTypeWithImports(currentPkg, forType)
	return strings.NewReplacer(
		"SRC", varName,
		"TYPE", colTyp,
		"NULLTYPE", nullTyp.Name,
		"NULLVAL", "true",
	).Replace(nullTyp.ValidExpr)
}

var _ NullTypeCreator = DatabaseSqlNull{}

type DatabaseSqlNull struct{}

func (d DatabaseSqlNull) NullType(name string) NullType {
	return NullType{
		Name:                fmt.Sprintf("sql.Null[%s]", name),
		ValidExpr:           "SRC.Valid",
		FromNullExpr:        "SRC.V",
		FromNullExprImports: []string{},
		ToNullExpr:          "NULLTYPE{V: SRC, Valid: NULLVAL}",
		ToNullExprImports:   []string{`"database/sql"`},
	}
}

func (d DatabaseSqlNull) NullTypeImports(forType Type) []string {
	return append(slices.Clone(forType.Imports), `"database/sql"`)
}

var _ NullTypeCreator = AarondlNull{}

type AarondlNull struct{}

func (a AarondlNull) NullType(name string) NullType {
	return NullType{
		Name:                fmt.Sprintf("null.Val[%s]", name),
		ValidExpr:           "SRC.IsSet()",
		FromNullExpr:        "SRC.GetOrZero()",
		FromNullExprImports: []string{},
		ToNullExpr:          "null.From(SRC)",
		ToNullExprImports:   []string{`"github.com/aarondl/opt/null"`},
	}
}

func (a AarondlNull) NullTypeImports(forType Type) []string {
	return append(slices.Clone(forType.Imports), `"github.com/aarondl/opt/null"`)
}
