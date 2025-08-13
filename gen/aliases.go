package gen

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
)

// initAliases takes the table information from the driver
// and fills in aliases where the user has provided none.
//
// This leaves us with a complete list of Go names for all tables,
// columns, and relationships.
func initAliases[C, I any](a drivers.Aliases, tables drivers.Tables[C, I], relMap Relationships) error {
	for _, t := range tables {
		tableAlias := a[t.Key]
		cleanKey := strings.ReplaceAll(t.Key, ".", "_")

		if len(tableAlias.UpPlural) == 0 {
			tableAlias.UpPlural = strmangle.TitleCase(strmangle.Plural(cleanKey))
		}
		if len(tableAlias.UpSingular) == 0 {
			tableAlias.UpSingular = strmangle.TitleCase(strmangle.Singular(cleanKey))
		}
		if len(tableAlias.DownPlural) == 0 {
			tableAlias.DownPlural = strmangle.CamelCase(strmangle.Plural(cleanKey))
		}
		if len(tableAlias.DownSingular) == 0 {
			tableAlias.DownSingular = strmangle.CamelCase(strmangle.Singular(cleanKey))
		}

		if tableAlias.Columns == nil {
			tableAlias.Columns = make(map[string]string)
		}
		if tableAlias.Relationships == nil {
			tableAlias.Relationships = make(map[string]string)
		}
		if tableAlias.Indexes == nil {
			tableAlias.Indexes = make(map[string]string)
		}
		if tableAlias.Constraints == nil {
			tableAlias.Constraints = make(map[string]string)
		}

		for _, c := range t.Columns {
			if _, ok := tableAlias.Columns[c.Name]; !ok {
				tableAlias.Columns[c.Name] = strmangle.TitleCase(c.Name)
			}

			r, _ := utf8.DecodeRuneInString(tableAlias.Columns[c.Name])
			if unicode.IsNumber(r) {
				tableAlias.Columns[c.Name] = "C" + tableAlias.Columns[c.Name]
			}
		}

		tableRels := relMap[t.Key]
		computed := relAlias(tableRels)
		for _, rel := range tableRels {
			if _, ok := tableAlias.Relationships[rel.Name]; !ok {
				tableAlias.Relationships[rel.Name] = computed[rel.Name]
			}
		}

		for _, idx := range t.Indexes {
			if _, ok := tableAlias.Indexes[idx.Name]; !ok {
				tableAlias.Indexes[idx.Name] = strmangle.TitleCase(idx.Name)
			}

			r, _ := utf8.DecodeRuneInString(tableAlias.Indexes[idx.Name])
			if unicode.IsNumber(r) {
				tableAlias.Indexes[idx.Name] = "I" + tableAlias.Indexes[idx.Name]
			}
		}

		for _, con := range t.Constraints.All() {
			if _, ok := tableAlias.Constraints[con.Name]; !ok {
				tableAlias.Constraints[con.Name] = strmangle.TitleCase(con.Name)
			}

			r, _ := utf8.DecodeRuneInString(tableAlias.Constraints[con.Name])
			if unicode.IsNumber(r) {
				tableAlias.Constraints[con.Name] = "C" + tableAlias.Constraints[con.Name]
			}
		}

		a[t.Key] = tableAlias
	}

	return validateAliases(a)
}

// For UpSingular, DownSingular, UpPlural, DownPlural
type globalAliasError struct {
	Type1  string
	Type2  string
	Value  string
	Table1 string
	Table2 string
}

func (e globalAliasError) Error() string {
	return fmt.Sprintf(
		"alias conflict for '%s': %s.%s conflicts with %s.%s",
		e.Value, e.Table1, e.Type1, e.Table2, e.Type2,
	)
}

func (e globalAliasError) Is(target error) bool {
	var t globalAliasError
	if !errors.As(target, &t) {
		return false
	}
	return t.Value == e.Value &&
		((t.Type1 == e.Type1 && t.Type2 == e.Type2 &&
			t.Table1 == e.Table1 && t.Table2 == e.Table2) ||
			(t.Type1 == e.Type2 && t.Type2 == e.Type1 &&
				t.Table1 == e.Table2 && t.Table2 == e.Table1))
}

// For Columns and Relationships
type tableAliasError struct {
	Type      string
	Value     string
	Table     string
	Conflict1 string
	Conflict2 string
}

func (e tableAliasError) Error() string {
	return fmt.Sprintf(
		"%s alias conflict for '%s' in table '%s': %s conflicts with %s",
		e.Type, e.Value, e.Table, e.Conflict1, e.Conflict2,
	)
}

func (e tableAliasError) Is(target error) bool {
	var t tableAliasError
	if !errors.As(target, &t) {
		return false
	}
	return t.Type == e.Type &&
		t.Value == e.Value &&
		t.Table == e.Table &&
		((t.Conflict1 == e.Conflict1 && t.Conflict2 == e.Conflict2) ||
			(t.Conflict1 == e.Conflict2 && t.Conflict2 == e.Conflict1))
}

/*
* Any duplicates in both UpSingular and UpPlural for all tables. To be clear, every entry should be unique across both lists.
* Any duplicates in both DownSingular and DownPlural for all tables. To be clear, every entry should be unique across both lists.
* Duplicates in the column aliases for each table.
* Duplicates in the relationship aliases for each table.
 */
func validateAliases(a drivers.Aliases) error {
	// Check for global alias uniqueness
	singularPlural := make(map[string]string)
	singularPluralType := make(map[string]string)

	globalErrors := []error{}
	// Table-level errors
	tableErrors := []error{}

	for tableKey, tableAlias := range a {
		for aliasType, alias := range func(yield func(string, string) bool) {
			if !yield("UpSingular", tableAlias.UpSingular) {
				return
			}

			if !yield("UpPlural", tableAlias.UpPlural) {
				return
			}

			if !yield("DownSingular", tableAlias.DownSingular) {
				return
			}

			if !yield("DownPlural", tableAlias.DownPlural) {
				return
			}
		} {
			if other, ok := singularPlural[alias]; ok {
				globalErrors = append(globalErrors, globalAliasError{
					Value: alias,

					Type1:  singularPluralType[alias],
					Table1: other,

					Type2:  aliasType,
					Table2: tableKey,
				})
			} else {
				singularPlural[alias] = tableKey
				singularPluralType[alias] = aliasType
			}
		}

		// Check column aliases for duplicates
		colReverse := make(map[string]string)
		for colName, colAlias := range tableAlias.Columns {
			if other, ok := colReverse[colAlias]; ok {
				tableErrors = append(tableErrors, tableAliasError{
					Type:      "column",
					Value:     colAlias,
					Table:     tableKey,
					Conflict1: other,
					Conflict2: colName,
				})
			} else {
				colReverse[colAlias] = colName
			}
		}

		// Check relationship aliases for duplicates
		relReverse := make(map[string]string)
		for relName, relAlias := range tableAlias.Relationships {
			if other, ok := relReverse[relAlias]; ok {
				tableErrors = append(tableErrors, tableAliasError{
					Type:      "relationship",
					Value:     relAlias,
					Table:     tableKey,
					Conflict1: other,
					Conflict2: relName,
				})
			} else {
				relReverse[relAlias] = relName
			}
		}
	}

	return errors.Join(errors.Join(globalErrors...), errors.Join(tableErrors...))
}
