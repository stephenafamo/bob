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
	// ValidExpr is used to check if a value of this type is valid
	// e.g. `SRC.Valid` for sql.Null[T] or `SRC.IsSet()` for null.Val[T]
	ValidExpr string `yaml:"valid_expr"`
	// UseExpr is used to convert a null value of this type
	// to a non-null value, if not provided it is assigned directly
	UseExpr string `yaml:"use_expr"`
	// Imports needed for the user expression
	UseExprImports []string `yaml:"use_expr_imports"`
	// CreateExpr is used to create a null value of this type
	// if not provided it is assigned directly
	CreateExpr string `yaml:"create_expr"`
	// Imports needed for the to null expression
	CreateExprImports []string `yaml:"create_expr_imports"`
}

type TypeModifier interface {
	// NullType returns the type to use for null values of the given type
	NullType(string) NullType
	// NullTypeImport
	NullTypeImports(Type) []string
	// OptionalType returns the type to use for optional values of the given type
	// fromOrToNull may be used to modify the behaviour of the `UseExpr` and `CreateExpr`
	// When `fromOrToNull` is false
	// * the `UseExpr` should convert the optional value to a non-optional value
	// * the `CreateExpr` should convert a non-optional value to an optional value
	// When `fromOrToNull` is true
	// * the `UseExpr` should convert the optional value to a non-optional but nullable value
	// * the `CreateExpr` should convert a non-optional but nullable value to an optional value
	OptionalType(typName string, def Type, isNull bool, fromOrToNull bool) (NullType, []string)
}

type Types struct {
	registered   map[string]Type
	typeModifier TypeModifier
}

func (t *Types) SetTypeModifier(creator TypeModifier) {
	t.typeModifier = creator
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
	if !null {
		name, def := t.GetNameAndDef(curr, namedType)
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
		if nullType.UseExpr == "" {
			nullType.UseExpr = "SRC"
		}
		if nullType.CreateExpr == "" {
			nullType.CreateExpr = "SRC"
		}

		name, _ := t.GetNameAndDef(currentPkg, nullType.Name)
		def.NullType.Name = name

		return def.NullType, t.registered[def.NullType.Name].Imports
	}

	return t.typeModifier.NullType(name), t.typeModifier.NullTypeImports(def)
}

func (t Types) GetNullTypeValid(currentPkg string, forType string, varName string) string {
	colTyp, _ := t.GetNameAndDef(currentPkg, forType)
	nullTyp, _ := t.GetNullTypeWithImports(currentPkg, forType)
	return strings.NewReplacer(
		"SRC", varName,
		"BASETYPE", colTyp,
		"NULLTYPE", nullTyp.Name,
		"NULLVAL", "true",
	).Replace(nullTyp.ValidExpr)
}

func (t Types) GetOptional(curr string, i language.Importer, namedType string, null bool) NullType {
	opt, imports := t.getOptional(curr, namedType, null, null)
	i.ImportList(imports)
	return opt
}

func (t Types) GetOptionalWithoutImporting(curr string, namedType string, null bool) NullType {
	opt, _ := t.getOptional(curr, namedType, null, null)
	return opt
}

func (t Types) getOptional(curr string, namedType string, isNull, fromOrToNull bool) (NullType, []string) {
	name, def := t.GetNameAndDef(curr, namedType)
	return t.typeModifier.OptionalType(name, def, isNull, fromOrToNull)
}

func (t Types) IsOptionalValid(currentPkg string, forType string, null bool, varName string) string {
	colTyp, _ := t.GetNameAndDef(currentPkg, forType)
	optTyp, _ := t.getOptional(currentPkg, forType, null, null)
	nullTyp := t.GetNullType(currentPkg, forType)
	return strings.NewReplacer(
		"SRC", varName,
		"NULLTYPE", nullTyp.Name,
		"BASETYPE", colTyp,
		"OPTIONALTYPE", optTyp.Name,
	).Replace(optTyp.ValidExpr)
}

func (t Types) FromOptional(currentPkg string, i language.Importer, forType string, varName string, isNull, fromOrToNull bool) string {
	colTyp, _ := t.GetNameAndDef(currentPkg, forType)
	optTyp, _ := t.getOptional(currentPkg, forType, isNull, fromOrToNull)
	nullTyp := t.GetNullType(currentPkg, forType)
	i.ImportList(optTyp.UseExprImports)
	return strings.NewReplacer(
		"SRC", varName,
		"NULLTYPE", nullTyp.Name,
		"BASETYPE", colTyp,
		"OPTIONALTYPE", optTyp.Name,
	).Replace(optTyp.UseExpr)
}

func (t Types) ToOptional(currentPkg string, i language.Importer, forType string, varName string, isNull, fromOrToNull bool) string {
	colTyp, _ := t.GetNameAndDef(currentPkg, forType)
	optTyp, _ := t.getOptional(currentPkg, forType, isNull, fromOrToNull)
	nullTyp := t.GetNullType(currentPkg, forType)
	i.ImportList(optTyp.CreateExprImports)
	return strings.NewReplacer(
		"SRC", varName,
		"NULLTYPE", nullTyp.Name,
		"BASETYPE", colTyp,
		"OPTIONALTYPE", optTyp.Name,
	).Replace(optTyp.CreateExpr)
}

var _ TypeModifier = AarondlNull{}

type AarondlNull struct{}

func (a AarondlNull) NullType(name string) NullType {
	return NullType{
		Name:              fmt.Sprintf("null.Val[%s]", name),
		ValidExpr:         "SRC.IsValue()",
		UseExpr:           "SRC.MustGet()",
		UseExprImports:    []string{},
		CreateExpr:        "null.From(SRC)",
		CreateExprImports: []string{`"github.com/aarondl/opt/null"`},
	}
}

func (a AarondlNull) OptionalType(name string, def Type, isNull, fromOrToNull bool) (NullType, []string) {
	imports := slices.Clone(def.Imports)

	if def.NullType.Name != "" {
		ot := NullType{
			Name:              fmt.Sprintf("omit.Val[%s]", name),
			ValidExpr:         "SRC.IsValue()",
			UseExpr:           "SRC.MustGet()",
			UseExprImports:    []string{},
			CreateExpr:        "omit.From(SRC)",
			CreateExprImports: []string{`"github.com/aarondl/opt/omit"`},
		}

		switch {
		case isNull && fromOrToNull:
			ot.Name = fmt.Sprintf("omit.Val[%s]", def.NullType.Name)

		case isNull && !fromOrToNull:
			ot.Name = fmt.Sprintf("omit.Val[%s]", def.NullType.Name)
			// int32 = omit.Val[sql.NullInt32]
			// a := b.MustGet().Int32
			ot.UseExpr = strings.ReplaceAll(
				def.NullType.UseExpr,
				"SRC", ot.UseExpr,
			)
			ot.UseExprImports = append(
				ot.UseExprImports, def.NullType.UseExprImports...,
			)

			// omit.Val[sql.NullInt32] = int32
			// b := omit.From(sql.NullInt32{Int32: 1, Valid: true})
			ot.CreateExpr = strings.ReplaceAll(
				ot.CreateExpr,
				"SRC", def.NullType.CreateExpr,
			)
			ot.CreateExpr = strings.ReplaceAll(ot.CreateExpr, "NULLVAL", "true")
			ot.CreateExprImports = append(
				ot.CreateExprImports, def.NullType.CreateExprImports...,
			)

		case !isNull && fromOrToNull:
			// sql.NullInt32 = omit.Val[int32]
			// a := sql.NullInt32{Int32: b.MustGet(), Valid: true}
			ot.UseExpr = strings.ReplaceAll(
				def.NullType.CreateExpr,
				"SRC", ot.UseExpr,
			)
			ot.UseExprImports = append(
				ot.UseExprImports, def.NullType.CreateExprImports...,
			)

			// omit.Val[int32] = sql.NullInt32
			// b := omit.From(a.Int32)
			ot.CreateExpr = strings.ReplaceAll(
				ot.CreateExpr,
				"SRC", def.NullType.UseExpr,
			)
			ot.CreateExprImports = append(
				ot.CreateExprImports, def.NullType.UseExprImports...,
			)

		}

		return ot, append(imports, `"github.com/aarondl/opt/omit"`)
	}

	if !isNull {
		ot := NullType{
			Name:              fmt.Sprintf("omit.Val[%s]", name),
			ValidExpr:         "SRC.IsValue()",
			UseExpr:           "SRC.MustGet()",
			UseExprImports:    []string{},
			CreateExpr:        "omit.From(SRC)",
			CreateExprImports: []string{`"github.com/aarondl/opt/omit"`},
		}

		if fromOrToNull {
			ot.UseExpr = "null.From(SRC.MustGet())"
			ot.UseExprImports = []string{`"github.com/aarondl/opt/null"`}

			ot.CreateExpr = "omit.From(SRC.MustGet())"
		}

		return ot, append(imports, `"github.com/aarondl/opt/omit"`)
	}

	nt := NullType{
		Name:              fmt.Sprintf("omitnull.Val[%s]", name),
		ValidExpr:         "!SRC.IsUnset()",
		UseExpr:           "SRC.MustGet()",
		UseExprImports:    []string{},
		CreateExpr:        "omitnull.From(SRC)",
		CreateExprImports: []string{`"github.com/aarondl/opt/omitnull"`},
	}

	if fromOrToNull {
		nt.UseExpr = "SRC.MustGetNull()"
		nt.CreateExpr = "omitnull.FromNull(SRC)"
	}

	return nt, append(imports, `"github.com/aarondl/opt/omitnull"`)
}

func (a AarondlNull) NullTypeImports(forType Type) []string {
	return append(slices.Clone(forType.Imports), `"github.com/aarondl/opt/null"`)
}

var _ TypeModifier = AarondlNullPointers{}

type AarondlNullPointers struct{}

func (a AarondlNullPointers) NullType(name string) NullType {
	return NullType{
		Name:              fmt.Sprintf("null.Val[%s]", name),
		ValidExpr:         "SRC.IsValue()",
		UseExpr:           "SRC.MustGet()",
		UseExprImports:    []string{},
		CreateExpr:        "null.From(SRC)",
		CreateExprImports: []string{`"github.com/aarondl/opt/null"`},
	}
}

func (a AarondlNullPointers) OptionalType(name string, def Type, isNull, fromOrToNull bool) (NullType, []string) {
	return optionalTypePointers(a, name, def, isNull, fromOrToNull)
}

func (a AarondlNullPointers) NullTypeImports(forType Type) []string {
	return append(slices.Clone(forType.Imports), `"github.com/aarondl/opt/null"`)
}

type DatabaseSqlNull struct{}

func (d DatabaseSqlNull) NullType(name string) NullType {
	return NullType{
		Name:              fmt.Sprintf("sql.Null[%s]", name),
		ValidExpr:         "SRC.Valid",
		UseExpr:           "SRC.V",
		UseExprImports:    nil,
		CreateExpr:        "NULLTYPE{V: SRC, Valid: NULLVAL}",
		CreateExprImports: []string{`"database/sql"`},
	}
}

func (d DatabaseSqlNull) OptionalType(name string, def Type, isNull, fromOrToNull bool) (NullType, []string) {
	return optionalTypePointers(d, name, def, isNull, fromOrToNull)
}

func (d DatabaseSqlNull) NullTypeImports(forType Type) []string {
	return append(slices.Clone(forType.Imports), `"database/sql"`)
}

func optionalTypePointers(tm TypeModifier, name string, def Type, isNull, fromOrToNull bool) (NullType, []string) {
	ot := NullType{
		Name:              fmt.Sprintf("*%s", name),
		ValidExpr:         "SRC != nil",
		UseExpr:           "func () BASETYPE { if SRC == nil { return *new(BASETYPE) }; return *SRC }()",
		UseExprImports:    nil,
		CreateExpr:        "func () *BASETYPE { return &SRC }()",
		CreateExprImports: nil,
	}

	nullType := def.NullType
	if nullType.Name == "" {
		nullType = tm.NullType(name)
	}

	if isNull {
		ot = NullType{
			Name:              fmt.Sprintf("*%s", nullType.Name),
			ValidExpr:         "SRC != nil",
			UseExpr:           "func () NULLTYPE { if SRC == nil { return *new(NULLTYPE) }; v := SRC; return *v }()",
			UseExprImports:    nil,
			CreateExpr:        "func () *NULLTYPE { v := SRC; return &v }()",
			CreateExprImports: nil,
		}
	}

	switch {
	case isNull && !fromOrToNull:
		ot.Name = fmt.Sprintf("omit.Val[%s]", def.NullType.Name)
		ot.UseExpr = strings.ReplaceAll(
			nullType.UseExpr,
			"SRC", ot.UseExpr,
		)
		ot.UseExprImports = append(
			ot.UseExprImports, nullType.UseExprImports...,
		)

		ot.CreateExpr = strings.ReplaceAll(
			ot.CreateExpr,
			"SRC", nullType.CreateExpr,
		)

		ot.CreateExpr = strings.ReplaceAll(ot.CreateExpr, "NULLVAL", "true")
		ot.CreateExprImports = append(
			ot.CreateExprImports, nullType.CreateExprImports...,
		)

	case !isNull && fromOrToNull:
		ot.UseExpr = strings.ReplaceAll(
			def.NullType.CreateExpr,
			"SRC", ot.UseExpr,
		)
		ot.UseExprImports = append(
			ot.UseExprImports, def.NullType.CreateExprImports...,
		)

		ot.CreateExpr = strings.ReplaceAll(
			ot.CreateExpr,
			"SRC", def.NullType.UseExpr,
		)
		ot.CreateExprImports = append(
			ot.CreateExprImports, def.NullType.UseExprImports...,
		)

	}

	return ot, def.Imports
}
