package drivers

import (
	"fmt"

	"github.com/stephenafamo/bob/orm"
)

const SelfJoinSuffix = "__self_join_reverse"

func BuildRelationships(tables []Table) map[string][]orm.Relationship {
	relationships := map[string][]orm.Relationship{}

	tableNameMap := make(map[string]Table, len(tables))
	for _, t := range tables {
		tableNameMap[t.Key] = t
	}

	for _, t1 := range tables {
		isJoinTable := IsJoinTable(t1)
		fkUniqueMap := make(map[string][2]bool, len(t1.Constraints.Foreign))
		fkNullableMap := make(map[string]bool, len(t1.Constraints.Foreign))

		// Build BelongsTo, ToOne and ToMany
		for _, fk := range t1.Constraints.Foreign {
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
					FromUnique:  localUnique,
					ToUnique:    foreignUnique,
					ToKey:       false,
					KeyNullable: localNullable,
				}},
			})

			flipSide := orm.RelSide{
				From:        t2.Key,
				FromColumns: fk.ForeignColumns,
				To:          t1.Key,
				ToColumns:   fk.Columns,
				FromUnique:  foreignUnique,
				ToUnique:    localUnique,
				ToKey:       true,
				KeyNullable: localNullable,
			}

			switch {
			case isJoinTable:
				// Skip. Join tables are handled below
			case t1.Key == t2.Key: // Self join
				relationships[t2.Key] = append(relationships[t2.Key], orm.Relationship{
					Name:  fk.Name + SelfJoinSuffix,
					Sides: []orm.RelSide{flipSide},
				})
			default:
				relationships[t2.Key] = append(relationships[t2.Key], orm.Relationship{
					Name:  fk.Name,
					Sides: []orm.RelSide{flipSide},
				})
			}
		}

		if !isJoinTable {
			continue
		}

		// Build ManyToMany
		rels := relationships[t1.Key]
		if len(rels) != 2 {
			panic(fmt.Sprintf("join table %s does not have 2 relationships, has %d", t1.Key, len(rels)))
		}
		r1, r2 := rels[0], rels[1]

		relationships[r1.Sides[0].To] = append(relationships[r1.Sides[0].To], orm.Relationship{
			Name: r1.Name + r2.Name,
			Sides: []orm.RelSide{
				{
					From:        r1.Sides[0].To,
					FromColumns: r1.Sides[0].ToColumns,
					To:          t1.Key,
					ToColumns:   r1.Sides[0].FromColumns,
					FromUnique:  fkUniqueMap[r1.Name][1],
					ToUnique:    fkUniqueMap[r1.Name][0],
					ToKey:       true,
					KeyNullable: fkNullableMap[r1.Name],
				},
				{
					From:        t1.Key,
					FromColumns: r2.Sides[0].FromColumns,
					To:          r2.Sides[0].To,
					ToColumns:   r2.Sides[0].ToColumns,
					FromUnique:  fkUniqueMap[r2.Name][0],
					ToUnique:    fkUniqueMap[r2.Name][1],
					ToKey:       false,
					KeyNullable: fkNullableMap[r2.Name],
				},
			},
		})
		// It is a many-to-many self join no need to duplicate the relationship
		if r1.Sides[0].To == r2.Sides[0].To {
			continue
		}
		relationships[r2.Sides[0].To] = append(relationships[r2.Sides[0].To], orm.Relationship{
			Name: r1.Name + r2.Name,
			Sides: []orm.RelSide{
				{
					From:        r2.Sides[0].To,
					FromColumns: r2.Sides[0].ToColumns,
					To:          t1.Key,
					ToColumns:   r2.Sides[0].FromColumns,
					FromUnique:  fkUniqueMap[r2.Name][1],
					ToUnique:    fkUniqueMap[r2.Name][0],
					ToKey:       true,
					KeyNullable: fkNullableMap[r2.Name],
				},
				{
					From:        t1.Key,
					FromColumns: r1.Sides[0].FromColumns,
					To:          r1.Sides[0].To,
					ToColumns:   r1.Sides[0].ToColumns,
					FromUnique:  fkUniqueMap[r1.Name][0],
					ToUnique:    fkUniqueMap[r1.Name][1],
					ToKey:       false,
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
	if t.Constraints.Primary != nil && sliceMatch(t.Constraints.Primary.Columns, cols) {
		return true
	}

	// Check other unique constrints
	for _, u := range t.Constraints.Uniques {
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
	if t.Constraints.Primary == nil {
		return false
	}

	// Must have exactly 2 foreign keys
	if len(t.Constraints.Foreign) != 2 {
		return false
	}

	// Number of columns must be the number of primary key columns
	if len(t.Columns) != len(t.Constraints.Primary.Columns) {
		return false
	}

	// length of both foreign keys must be the total length of the columns
	if len(t.Columns) != (len(t.Constraints.Foreign[0].Columns) + len(t.Constraints.Foreign[1].Columns)) {
		return false
	}

	// both foreign keys must have distinct columns
	if !distinctElems(t.Constraints.Foreign[0].Columns, t.Constraints.Foreign[1].Columns) {
		return false
	}

	// It is a join table!!!
	return true
}

// A composite primary key involving two columns
// Both primary key columns are also foreign keys
func IsJoinTable2(t Table, r orm.Relationship, position int) bool {
	if position == 0 || len(r.Sides) < 2 {
		return false
	}

	if position == len(r.Sides) {
		return false
	}

	if t.Key != r.Sides[position-1].To {
		panic(fmt.Sprintf(
			"table name does not match relationship position, expected %s got %s",
			t.Key, r.Sides[position-1].To,
		))
	}

	relevantSides := r.Sides[position-1 : position+1]

	// If the external mappings are not unique, it is not a join table
	if !relevantSides[0].FromUnique || !relevantSides[1].ToUnique {
		return false
	}

	// All columns in the table must be in one of the sides
	// if not, it is not a join table
ColumnsLoop:
	for _, col := range t.Columns {
		for _, sideCol := range relevantSides[0].ToColumns {
			if col.Name == sideCol {
				continue ColumnsLoop
			}
		}
		for _, sideCol := range relevantSides[1].FromColumns {
			if col.Name == sideCol {
				continue ColumnsLoop
			}
		}
		return false
	}

	// It is a join table!!!
	return true
}
