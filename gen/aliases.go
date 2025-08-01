package gen

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
)

// initAliases takes the table information from the driver
// and fills in default aliases where the user has provided none.
// This function ensures that every table, column, and relationship
// has an alias defined in the `a drivers.Aliases` map.
func initAliases[C, I any](a drivers.Aliases, tables drivers.Tables[C, I], relMap Relationships) {
	for _, t := range tables {
		tableAlias := a[t.Key] // Get existing or new alias struct
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

		for _, c := range t.Columns {
			if _, ok := tableAlias.Columns[c.Name]; !ok {
				generatedName := strmangle.TitleCase(c.Name)
				r, _ := utf8.DecodeRuneInString(generatedName)
				if unicode.IsNumber(r) {
					generatedName = "C" + generatedName
				}
				tableAlias.Columns[c.Name] = generatedName
			}
		}

		tableRels := relMap[t.Key]
		// relAlias computes default relationship names based on relationship properties
		// This function's internal logic is assumed to exist and work as before.
		computedRelAliases := relAlias(tableRels)
		for _, rel := range tableRels {
			if _, ok := tableAlias.Relationships[rel.Name]; !ok {
				if computedName, computedOK := computedRelAliases[rel.Name]; computedOK {
					tableAlias.Relationships[rel.Name] = computedName
				} else {
					// Fallback if relAlias somehow doesn't provide a name, though it should.
					// This could be a point of error or assert if rel.Name must be in computedRelAliases.
					// For now, generate a simple default.
					tableAlias.Relationships[rel.Name] = strmangle.TitleCase(rel.Name)
				}
			}
		}
		a[t.Key] = tableAlias // Ensure the modified/initialized alias struct is put back in the map
	}
}

// validateAliases checks for alias clashes after all aliases have been populated.
// It takes the fully populated `drivers.Aliases` map and checks for several types of clashes.
func validateAliases[C, I any](a drivers.Aliases, tables drivers.Tables[C, I], relMap Relationships) []error {
	var errors []error

	// Global tracking for table aliases
	upSingularGlobal := make(map[string]string)
	upPluralGlobal := make(map[string]string)
	downSingularGlobal := make(map[string]string)
	downPluralGlobal := make(map[string]string)

	for _, t := range tables {
		tableAlias := a[t.Key] // Aliases are expected to be populated by initAliases

		// Table Alias Clashes
		// UpSingular
		currentUpSingular := tableAlias.UpSingular // Directly use the populated alias
		if existingTableKey, ok := upSingularGlobal[currentUpSingular]; ok && existingTableKey != t.Key {
			errors = append(errors, fmt.Errorf("alias clash: UpSingular '%s' used by table '%s' and table '%s'", currentUpSingular, existingTableKey, t.Key))
		}
		if existingTableKey, ok := upPluralGlobal[currentUpSingular]; ok { // Check against UpPlural of any table
			errors = append(errors, fmt.Errorf("alias clash: UpSingular '%s' (table '%s') conflicts with UpPlural '%s' (table '%s')", currentUpSingular, t.Key, currentUpSingular, existingTableKey))
		}
		if _, ok := upSingularGlobal[currentUpSingular]; !ok { // Add to map only if it's not causing a direct same-type clash from another table (already handled)
			upSingularGlobal[currentUpSingular] = t.Key
		}


		// UpPlural
		currentUpPlural := tableAlias.UpPlural // Directly use the populated alias
		if existingTableKey, ok := upPluralGlobal[currentUpPlural]; ok && existingTableKey != t.Key {
			errors = append(errors, fmt.Errorf("alias clash: UpPlural '%s' used by table '%s' and table '%s'", currentUpPlural, existingTableKey, t.Key))
		}
		if existingTableKey, ok := upSingularGlobal[currentUpPlural]; ok { // Check against UpSingular of any table
			errors = append(errors, fmt.Errorf("alias clash: UpPlural '%s' (table '%s') conflicts with UpSingular '%s' (table '%s')", currentUpPlural, t.Key, currentUpPlural, existingTableKey))
		}
		if _, ok := upPluralGlobal[currentUpPlural]; !ok {
			upPluralGlobal[currentUpPlural] = t.Key
		}

		// DownSingular
		currentDownSingular := tableAlias.DownSingular // Directly use the populated alias
		if existingTableKey, ok := downSingularGlobal[currentDownSingular]; ok && existingTableKey != t.Key {
			errors = append(errors, fmt.Errorf("alias clash: DownSingular '%s' used by table '%s' and table '%s'", currentDownSingular, existingTableKey, t.Key))
		}
		if existingTableKey, ok := downPluralGlobal[currentDownSingular]; ok { // Check against DownPlural of any table
			errors = append(errors, fmt.Errorf("alias clash: DownSingular '%s' (table '%s') conflicts with DownPlural '%s' (table '%s')", currentDownSingular, t.Key, currentDownSingular, existingTableKey))
		}
		if _, ok := downSingularGlobal[currentDownSingular]; !ok {
			downSingularGlobal[currentDownSingular] = t.Key
		}

		// DownPlural
		currentDownPlural := tableAlias.DownPlural // Directly use the populated alias
		if existingTableKey, ok := downPluralGlobal[currentDownPlural]; ok && existingTableKey != t.Key {
			errors = append(errors, fmt.Errorf("alias clash: DownPlural '%s' used by table '%s' and table '%s'", currentDownPlural, existingTableKey, t.Key))
		}
		if existingTableKey, ok := downSingularGlobal[currentDownPlural]; ok { // Check against DownSingular of any table
			errors = append(errors, fmt.Errorf("alias clash: DownPlural '%s' (table '%s') conflicts with DownSingular '%s' (table '%s')", currentDownPlural, t.Key, currentDownPlural, existingTableKey))
		}
		if _, ok := downPluralGlobal[currentDownPlural]; !ok {
			downPluralGlobal[currentDownPlural] = t.Key
		}

		// Column Alias Clashes (within the current table)
		// tableAlias.Columns is expected to be populated by initAliases
		if tableAlias.Columns != nil { // Check if Columns map exists
			tableColumnAliases := make(map[string]string)
			for _, c := range t.Columns { // Iterate over actual columns to get their names
				alias, aliasExists := tableAlias.Columns[c.Name]
				if !aliasExists {
					// This case should ideally not happen if initAliases did its job.
					// Or, it means a column exists in the table definition but not in the alias map.
					// Depending on strictness, this could be an error itself.
					// For now, we assume initAliases ensures all columns have an alias entry.
					errors = append(errors, fmt.Errorf("consistency error: column '%s' in table '%s' has no alias entry", c.Name, t.Key))
					continue
				}

				if existingColName, ok := tableColumnAliases[alias]; ok {
					errors = append(errors, fmt.Errorf("alias clash in table '%s': column alias '%s' is used by both column '%s' and column '%s'", t.Key, alias, existingColName, c.Name))
				} else {
					tableColumnAliases[alias] = c.Name
				}
			}
		}

		// Relationship Alias Clashes (within the current table)
		// tableAlias.Relationships is expected to be populated by initAliases
		if tableAlias.Relationships != nil { // Check if Relationships map exists
			tableRelationshipAliases := make(map[string]string)
			// Iterate over tableAlias.Relationships which contains original_name -> alias_name
			for relOriginalName, relAliasName := range tableAlias.Relationships {
				if existingRelOriginalName, ok := tableRelationshipAliases[relAliasName]; ok {
					errors = append(errors, fmt.Errorf("alias clash in table '%s': relationship alias '%s' is used by both relationship '%s' and relationship '%s'", t.Key, relAliasName, existingRelOriginalName, relOriginalName))
				} else {
					tableRelationshipAliases[relAliasName] = relOriginalName
				}
			}
		}
	}

	return errors
}
