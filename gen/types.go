package gen

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

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

// matchString reports whether string a matches a pattern.
// Pattern a can be either a literal string (case-insensitive comparison)
// or a regular expression enclosed with / slashes.
// Regex patterns are automatically made case-insensitive.
func matchString(pattern, candidate string) bool {
	stringPatterns, regexPatterns := drivers.ClassifyPatterns([]string{pattern})
	for _, pattern := range stringPatterns {
		if strings.EqualFold(pattern, candidate) {
			return true
		}
	}

	for _, pattern := range regexPatterns {
		caseInsensitivePattern := "(?i)" + pattern
		if matched, _ := regexp.MatchString(caseInsensitivePattern, candidate); matched {
			return true
		}
	}

	return false
}

// matchColumn determines if col matches all specified criteria in spec (logical AND).
//
// Empty spec fields and the `Unique` property are ignored (as those can vary independent of type).
// String fields are matched case-insensitively and by regex.
// Boolean fields are only evaluated when string fields have matched.
func matchColumn(col, spec drivers.Column) bool {
	matchedStringRule := false

	matches := func(pattern, value string) bool {
		if pattern == "" {
			return true // empty pattern matches anything
		}
		if matchString(pattern, value) {
			matchedStringRule = true
			return true
		}
		return false
	}

	if !matches(spec.Name, col.Name) {
		return false
	}

	if !matches(spec.Type, col.Type) {
		return false
	}

	if !matches(spec.DBType, col.DBType) {
		return false
	}

	if !matches(spec.DomainName, col.DomainName) {
		return false
	}

	if !matches(spec.Comment, col.Comment) {
		return false
	}

	// Boolean fields are only checked if at least one string field matched
	if !matchedStringRule {
		return false
	}

	if spec.Generated != col.Generated {
		return false
	}

	if spec.AutoIncr != col.AutoIncr {
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
