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

type fkTarget struct {
	tableIdx int
	colIdx   int
}

// processTypeReplacements checks the config for type replacements
// and performs them, then infers foreign key column types from referenced columns.
func processTypeReplacements[C, I any](types drivers.Types, replacements []Replace, tables []drivers.Table[C, I]) {
	replacedRule := make([][]int, len(tables))
	for i, t := range tables {
		replacedRule[i] = make([]int, len(t.Columns))
		for j := range replacedRule[i] {
			replacedRule[i][j] = -1
		}
	}

	for ruleIdx, r := range replacements {
		didMatch := false
		for i := range tables {
			t := tables[i]

			if !shouldReplaceInTable(t, r) {
				continue
			}

			for j := range t.Columns {
				c := t.Columns[j]
				if r.Match.Matches(c) {
					didMatch = true

					if ok := types.Contains(r.Replace); !ok && !isPrimitiveType(r.Replace) {
						fmt.Printf("WARNING: No definition found for replacement: %q\n", r.Replace)
					}

					t.Columns[j].Type = r.Replace
					replacedRule[i][j] = ruleIdx
				}
			}
		}

		// Print a warning if we didn't match anything
		if !didMatch {
			c := r.Match
			fmt.Printf("WARNING: No match found for replacement: %+v\n", c)
		}
	}

	inferForeignKeyTypes(tables, replacedRule)
}

func buildForeignKeyEdges[C, I any](tables []drivers.Table[C, I]) [][][]fkTarget {
	tableIndex := make(map[string]int, len(tables))
	colIndex := make([]map[string]int, len(tables))
	edges := make([][][]fkTarget, len(tables))

	for i, t := range tables {
		tableIndex[t.Key] = i
		cols := make(map[string]int, len(t.Columns))
		for j, c := range t.Columns {
			cols[c.Name] = j
		}
		colIndex[i] = cols
		edges[i] = make([][]fkTarget, len(t.Columns))
	}

	for i, t := range tables {
		for _, fk := range t.Constraints.Foreign {
			if len(fk.Columns) != len(fk.ForeignColumns) {
				continue
			}

			ftIdx, ok := tableIndex[fk.ForeignTable]
			if !ok {
				continue
			}

			for k, localCol := range fk.Columns {
				cIdx, ok := colIndex[i][localCol]
				if !ok {
					continue
				}

				fcIdx, ok := colIndex[ftIdx][fk.ForeignColumns[k]]
				if !ok {
					continue
				}

				edges[i][cIdx] = append(edges[i][cIdx], fkTarget{tableIdx: ftIdx, colIdx: fcIdx})
			}
		}
	}

	return edges
}

func inferForeignKeyTypes[C, I any](tables []drivers.Table[C, I], replacedRule [][]int) {
	edges := buildForeignKeyEdges(tables)

	// Propagate explicit replacements along foreign keys.
	// replacedRule strictly increases on each update, guaranteeing termination even with cycles.
	for {
		changed := false
		for i := range tables {
			for j := range tables[i].Columns {
				for _, target := range edges[i][j] {
					if replacedRule[target.tableIdx][target.colIdx] > replacedRule[i][j] {
						tables[i].Columns[j].Type = tables[target.tableIdx].Columns[target.colIdx].Type
						replacedRule[i][j] = replacedRule[target.tableIdx][target.colIdx]
						changed = true
					}
				}
			}
		}
		if !changed {
			break
		}
	}

	// Align unreplaced foreign key columns whose base type differs from their referenced column,
	// propagating outward from settled/root columns to avoid oscillating on cycles.
	settled := make([][]bool, len(tables))
	for i := range tables {
		settled[i] = make([]bool, len(tables[i].Columns))
		for j := range tables[i].Columns {
			settled[i][j] = replacedRule[i][j] >= 0 || len(edges[i][j]) == 0
		}
	}

	for {
		changed := false
		for i := range tables {
			for j := range tables[i].Columns {
				if settled[i][j] {
					continue
				}
				for _, target := range edges[i][j] {
					if settled[target.tableIdx][target.colIdx] {
						tables[i].Columns[j].Type = tables[target.tableIdx].Columns[target.colIdx].Type
						settled[i][j] = true
						changed = true
						break
					}
				}
			}
		}
		if !changed {
			break
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

// shouldReplaceInTable checks if tables were specified in types.match in the config.
// If tables were set, it checks if the given table is among the specified tables.
func shouldReplaceInTable[C, I any](t drivers.Table[C, I], r Replace) bool {
	if len(r.Tables) == 0 {
		return true
	}

	return slices.Contains(r.Tables, t.Key)
}
