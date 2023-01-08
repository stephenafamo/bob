package drivers

import (
	"github.com/stephenafamo/bob/orm"
)

func BuildRelationships(tables []Table) map[string][]orm.Relationship {
	relationships := map[string][]orm.Relationship{}

	tableNameMap := make(map[string]Table, len(tables))
	for _, t := range tables {
		tableNameMap[t.Key] = t
	}

	for _, t1 := range tables {
		fkUniqueMap := make(map[string][2]bool, len(t1.FKeys))
		fkNullableMap := make(map[string]bool, len(t1.FKeys))

		// Build BelongsTo, ToOne and ToMany
		for _, fk := range t1.FKeys {
			t2, ok := tableNameMap[fk.ForeignTable]
			if !ok {
				continue // no matching target table
			}

			localUnique := hasExactUnique(t1, fk.Columns...)
			foreignUnique := hasExactUnique(t2, fk.ForeignColumns...)
			fkUniqueMap[fk.Name] = [2]bool{localUnique, foreignUnique}

			localNullable := allNullable(t1, fk.Columns...)
			fkNullableMap[fk.Name] = localNullable

			pair1 := make(map[string]string, len(fk.Columns))
			pair2 := make(map[string]string, len(fk.Columns))
			for index, localCol := range fk.Columns {
				foreignCol := fk.ForeignColumns[index]
				pair1[localCol] = foreignCol
				pair2[foreignCol] = localCol
			}

			relationships[t1.Key] = append(relationships[t1.Key], orm.Relationship{
				Name: fk.Name,
				Sides: []orm.RelSide{{
					From:        t1.Key,
					FromColumns: fk.Columns,
					To:          t2.Key,
					ToColumns:   fk.ForeignColumns,
					ToKey:       false,
					ToUnique:    foreignUnique,
					KeyNullable: localNullable,
				}},
			})

			if !t1.IsJoinTable && t1.Key != t2.Key {
				relationships[t2.Key] = append(relationships[t2.Key], orm.Relationship{
					Name: fk.Name,
					Sides: []orm.RelSide{{
						From:        t2.Key,
						FromColumns: fk.ForeignColumns,
						To:          t1.Key,
						ToColumns:   fk.Columns,
						ToKey:       true,
						ToUnique:    localUnique,
						KeyNullable: localNullable,
					}},
				})
			}
		}

		if !t1.IsJoinTable {
			continue
		}

		// Build ManyToMany
		rels := relationships[t1.Key]
		if len(rels) != 2 {
			panic("join table does not have 2 relationships")
		}
		r1, r2 := rels[0], rels[1]

		relationships[r1.Sides[0].To] = append(relationships[r1.Sides[0].To], orm.Relationship{
			Name:        r1.Name + r2.Name,
			ByJoinTable: true,
			Sides: []orm.RelSide{
				{
					From:        r1.Sides[0].To,
					FromColumns: r1.Sides[0].ToColumns,
					To:          t1.Key,
					ToColumns:   r1.Sides[0].FromColumns,
					ToKey:       true,
					ToUnique:    fkUniqueMap[r1.Name][0],
					KeyNullable: fkNullableMap[r1.Name],
				},
				{
					From:        t1.Key,
					FromColumns: r2.Sides[0].FromColumns,
					To:          r2.Sides[0].To,
					ToColumns:   r2.Sides[0].ToColumns,
					ToKey:       false,
					ToUnique:    fkUniqueMap[r1.Name][1],
					KeyNullable: fkNullableMap[r2.Name],
				},
			},
		})
		relationships[r2.Sides[0].To] = append(relationships[r2.Sides[0].To], orm.Relationship{
			Name:        r1.Name + r2.Name,
			ByJoinTable: true,
			Sides: []orm.RelSide{
				{
					From:        r2.Sides[0].To,
					FromColumns: r2.Sides[0].ToColumns,
					To:          t1.Key,
					ToColumns:   r2.Sides[0].FromColumns,
					ToKey:       true,
					ToUnique:    fkUniqueMap[r2.Name][0],
					KeyNullable: fkNullableMap[r2.Name],
				},
				{
					From:        t1.Key,
					FromColumns: r1.Sides[0].FromColumns,
					To:          r1.Sides[0].To,
					ToColumns:   r1.Sides[0].ToColumns,
					ToKey:       false,
					ToUnique:    fkUniqueMap[r2.Name][1],
					KeyNullable: fkNullableMap[r1.Name],
				},
			},
		})
	}

	return relationships
}

// Returns true if the table has a unique constraint on exactly these columns
func allNullable(t Table, cols ...string) bool {
	foundNullable := 0
	for _, col := range t.Columns {
		for _, cname := range cols {
			if col.Name == cname && col.Nullable {
				foundNullable++
				if foundNullable == len(cols) {
					return true
				}
			}
		}
	}

	return false
}

// Returns true if the table has a unique constraint on exactly these columns
func hasExactUnique(t Table, cols ...string) bool {
	if len(cols) == 0 {
		return false
	}

	// Primary keys are unique
	if t.PKey != nil && sliceMatch(t.PKey.Columns, cols) {
		return true
	}

	// Check other unique constrints
	for _, u := range t.Uniques {
		if sliceMatch(u.Columns, cols) {
			return true
		}
	}

	return false
}

func sliceMatch[T comparable, Ts ~[]T](a, b Ts) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return false
	}

	var matches int
	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				matches++
			}
		}
	}

	return matches == len(a)
}

// Has no matching elements
func distinctElems[T comparable, Ts ~[]T](a, b Ts) bool {
	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				return false
			}
		}
	}

	return true
}

// A composite primary key involving two columns
// Both primary key columns are also foreign keys
func IsJoinTable(t Table) bool {
	if t.PKey == nil {
		return false
	}

	// Must have exactly 2 foreign keys
	if len(t.FKeys) != 2 {
		return false
	}

	// Number of columns must be the number of primary key columns
	if len(t.Columns) != len(t.PKey.Columns) {
		return false
	}

	// length of both foreign keys must be the total length of the columns
	if len(t.Columns) != (len(t.FKeys[0].Columns) + len(t.FKeys[1].Columns)) {
		return false
	}

	// both foreign keys must have distinct columns
	if !distinctElems(t.FKeys[0].Columns, t.FKeys[1].Columns) {
		return false
	}

	// It is a join table!!!
	return true
}
