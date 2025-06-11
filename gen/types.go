package gen

import (
	"fmt"
	"slices"

	"github.com/stephenafamo/bob/gen/drivers"
)

func isPrimitiveType(name string) bool {
	switch name {
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64",
		"byte", "rune", "string", "bool":
		return true
	default:
		return false
	}
}

// processTypeReplacements checks the config for type replacements
// and performs them.
func processTypeReplacements[C, I any](types drivers.Types, replacements []Replace, tables []drivers.Table[C, I]) {
	for _, r := range replacements {
		didMatch := false
		for i := range tables {
			t := tables[i]

			if !shouldReplaceInTable(t, r) {
				continue
			}

			for j := range t.Columns {
				c := t.Columns[j]
				if matchColumn(c, r.Match) {
					didMatch = true

					if ok := types.Contains(r.Replace); !ok && !isPrimitiveType(r.Replace) {
						fmt.Printf("WARNING: No definition found for replacement: %q\n", r.Replace)
					}

					t.Columns[j].Type = r.Replace
				}
			}
		}

		// Print a warning if we didn't match anything
		if !didMatch {
			c := r.Match
			fmt.Printf(
				"WARNING: No match found for replacement:\nname: %s\ndb_type: %s\ndefault: %s\ncomment: %s\nnullable: %t\ngenerated: %t\nautoincr: %t\ndomain_name: %s\n",
				c.Name, c.DBType, c.Default, c.Comment, c.Nullable, c.Generated, c.AutoIncr, c.DomainName)
		}
	}
}

// matchColumn checks if a column 'c' matches specifiers in 'm'.
// Anything defined in m is checked against a's values, the
// match is a done using logical and (all specifiers must match).
// Bool fields are only checked if a string type field matched first
// and if a string field matched they are always checked (must be defined).
//
// Doesn't care about Unique columns since those can vary independent of type.
func matchColumn(c, m drivers.Column) bool {
	matchedSomething := false

	// return true if we matched, or we don't have to match
	// if we actually matched against something, then additionally set
	// matchedSomething so we can check boolean values too.
	matches := func(matcher, value string) bool {
		if len(matcher) != 0 && matcher != value {
			return false
		}
		matchedSomething = true
		return true
	}

	if !matches(m.Name, c.Name) {
		return false
	}
	if !matches(m.Type, c.Type) {
		return false
	}
	if !matches(m.DBType, c.DBType) {
		return false
	}

	if !matches(m.DomainName, c.DomainName) {
		return false
	}
	if !matches(m.Comment, c.Comment) {
		return false
	}

	if !matchedSomething {
		return false
	}

	if m.Generated != c.Generated {
		return false
	}

	return true
}

// shouldReplaceInTable checks if tables were specified in types.match in the config.
// If tables were set, it checks if the given table is among the specified tables.
func shouldReplaceInTable[C, I any](t drivers.Table[C, I], r Replace) bool {
	if len(r.Tables) == 0 {
		return true
	}

	return slices.Contains(r.Tables, t.Key)
}
