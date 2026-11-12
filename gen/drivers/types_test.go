package drivers

import (
	"strings"
	"testing"
)

// fakeImporter records imports; only ImportList is exercised by ToOptional.
type fakeImporter []string

func (f *fakeImporter) Import(pkgs ...string) string {
	*f = append(*f, pkgs...)
	return ""
}

func (f *fakeImporter) ImportList(list []string) string {
	*f = append(*f, list...)
	return ""
}

func (f *fakeImporter) ToList() []string {
	return *f
}

// TestToOptionalPointerLiteralValid checks that the generated code for an
// optional/pointer value built from a literal (e.g. a relationship to_where
// go_value like "true" or "1") is valid Go. The pointer type systems
// ("github.com/aarondl/opt/null" -> AarondlNullPointers and "database/sql" ->
// DatabaseSqlNull) wrap the source value with &SRC, which becomes the invalid
// "&true"/"&1" (cannot take the address of a literal). See bob issue #553.
func TestToOptionalPointerLiteralValid(t *testing.T) {
	t.Parallel()

	typeCases := []struct {
		modifier TypeModifier
		name     string
		forType  string
		lit      string // the literal thrown at the generator (e.g. go_value "true")
	}{
		{DatabaseSqlNull{}, "database/sql bool literal true", "bool", "true"},
		{DatabaseSqlNull{}, "database/sql int literal 1", "int", "1"},
		{AarondlNullPointers{}, "opt/null bool literal true", "bool", "true"},
		{AarondlNullPointers{}, "opt/null int literal 1", "int", "1"},
	}

	for _, tc := range typeCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var types Types
			types.Register("bool", Type{})
			types.Register("int", Type{})
			types.SetTypeModifier(tc.modifier)

			var imp fakeImporter
			// request a non-null bool/int handled as an optional pointer
			expr := types.ToOptional("", &imp, tc.forType, tc.lit, false, false)

			// The generated expression must be addressable Go: it may reference
			// SRC through a variable, never the literal directly (&true or &1).
			if bad := "&" + tc.lit; strings.Contains(expr, bad) {
				t.Errorf("ToOptional(%q, %q) produced non-addressable literal %q in %q; want valid Go", tc.forType, tc.lit, bad, expr)
			}
		})
	}
}
