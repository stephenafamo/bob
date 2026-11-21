package drivers

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"github.com/stephenafamo/bob/gen/language"
)

// A relationship's to_where.go_value must be assignable to its setter field
// even when the literal's default type differs from the column type.
func TestToOptionalPointerLiteralValid(t *testing.T) {
	t.Parallel()

	typeCases := []struct {
		modifier TypeModifier
		name     string
		forType  string
		lit      string
	}{
		{DatabaseSqlNull{}, "database/sql bool literal true", "bool", "true"},
		{DatabaseSqlNull{}, "database/sql int literal 1", "int", "1"},
		{DatabaseSqlNull{}, "database/sql int32 literal 1", "int32", "1"},
		{AarondlNullPointers{}, "opt/null bool literal true", "bool", "true"},
		{AarondlNullPointers{}, "opt/null int literal 1", "int", "1"},
		{AarondlNullPointers{}, "opt/null int32 literal 1", "int32", "1"},
	}

	for _, tc := range typeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var registry Types
			registry.Register(tc.forType, Type{})
			registry.SetTypeModifier(tc.modifier)

			imp := language.Languages{}.GetOutputLanguage(".go").Importer()
			expr := registry.ToOptional("", imp, tc.forType, tc.lit, false, false)
			src := "package models\nvar _ *" + tc.forType + " = " + expr
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "setter.go", src, 0)
			if err != nil {
				t.Fatal(err)
			}

			var config types.Config
			if _, err := config.Check("models", fset, []*ast.File{file}, nil); err != nil {
				t.Fatalf("%s\n%v", src, err)
			}
		})
	}
}
