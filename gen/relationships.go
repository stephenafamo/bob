package gen

import (
	"fmt"
	"slices"
	"strings"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
)

const selfJoinSuffix = "__self_join_reverse"

type Relationships map[string][]orm.Relationship

// Set parameters of the relationship (unique, nullables, e.t.c.)
func initRelationships[C, I any](r Relationships, tables drivers.Tables[C, I]) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()

	for tName, rels := range r {
		for i, rel := range rels {
			for j, side := range rel.Sides {
				from := tables.Get(side.From)
				to := tables.Get(side.To)

				// Set the uniqueness
				r[tName][i].Sides[j].FromUnique = from.HasExactUnique(
					side.FromColumns...,
				)
				r[tName][i].Sides[j].ToUnique = to.HasExactUnique(
					side.ToColumns...,
				)

				if side.Modify == "" {
					side.Modify = inferModify(side, tables)
				}

				switch strings.ToLower(side.Modify) {
				case "from":
					r[tName][i].Sides[j].Modify = "from"
					r[tName][i].Sides[j].KeyNullable = allNullable(
						from, side.FromColumns...,
					)
				case "to":
					r[tName][i].Sides[j].Modify = "to"
					r[tName][i].Sides[j].KeyNullable = allNullable(
						to, side.ToColumns...,
					)
				default:
					return fmt.Errorf(`rel side modify should be "from" or "to",  got %q`, side.Modify)
				}
			}
		}
	}

	return nil
}

func (r Relationships) Get(table string) []orm.Relationship {
	return r[table]
}

func (rs Relationships) GlobalKey(r orm.Relationship) string {
	tables := []string{r.Local(), r.Foreign()}
	slices.Sort(tables)
	return fmt.Sprintf("%s.%s", strings.Join(tables, "."), strings.TrimSuffix(r.Name, selfJoinSuffix))
}

// GetInverse returns the Relationship of the other side
func (rs Relationships) GetInverse(r orm.Relationship) orm.Relationship {
	frels, ok := rs[r.Foreign()]
	if !ok {
		return orm.Relationship{}
	}

	toMatch := r.Name
	if r.Local() == r.Foreign() {
		hadSuffix := strings.HasSuffix(r.Name, selfJoinSuffix)
		toMatch = strings.TrimSuffix(r.Name, selfJoinSuffix)
		if hadSuffix {
			toMatch += selfJoinSuffix
		}
	}

	for _, r2 := range frels {
		if toMatch == r2.Name {
			return r2
		}
	}

	return orm.Relationship{}
}

func buildRelationships[C, I any](tables []drivers.Table[C, I]) Relationships {
	relationships := map[string][]orm.Relationship{}

	tableNameMap := make(map[string]drivers.Table[C, I], len(tables))
	for _, t := range tables {
		tableNameMap[t.Key] = t
	}

	for _, t1 := range tables {
		isJoinTable := t1.IsJoinTable()

		// Build BelongsTo, ToOne and ToMany
		for _, fk := range t1.Constraints.Foreign {
			t2, ok := tableNameMap[fk.ForeignTable]
			if !ok {
				continue // no matching target table
			}

			relationships[t1.Key] = append(relationships[t1.Key], orm.Relationship{
				Name: fk.Name,
				Sides: []orm.RelSide{{
					From:        t1.Key,
					FromColumns: fk.Columns,
					To:          t2.Key,
					ToColumns:   fk.ForeignColumns,
					Modify:      "from",
				}},
			})

			flipSide := orm.RelSide{
				From:        t2.Key,
				FromColumns: fk.ForeignColumns,
				To:          t1.Key,
				ToColumns:   fk.Columns,
				Modify:      "to",
			}

			switch {
			case isJoinTable:
				// Skip. Join tables are handled below
			case t1.Key == t2.Key: // Self join
				relationships[t2.Key] = append(relationships[t2.Key], orm.Relationship{
					Name:  fk.Name + selfJoinSuffix,
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
					Modify:      "to",
				},
				{
					From:        t1.Key,
					FromColumns: r2.Sides[0].FromColumns,
					To:          r2.Sides[0].To,
					ToColumns:   r2.Sides[0].ToColumns,
					Modify:      "from",
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
					Modify:      "to",
				},
				{
					From:        t1.Key,
					FromColumns: r1.Sides[0].FromColumns,
					To:          r1.Sides[0].To,
					ToColumns:   r1.Sides[0].ToColumns,
					Modify:      "from",
				},
			},
		})
	}

	return relationships
}

func flipRelationships[C, I any](relMap Relationships, tables []drivers.Table[C, I]) error {
	for _, rels := range relMap {
	RelsLoop:
		for _, rel := range rels {
			if rel.NoReverse || len(rel.Sides) < 1 {
				continue
			}
			ftable := rel.Sides[len(rel.Sides)-1].To

			// Check if the foreign table already has the
			// reverse configured
			existingRels := relMap[ftable]
			for _, existing := range existingRels {
				if existing.Name == rel.Name {
					continue RelsLoop
				}
			}

			flipped, err := flipRelationship(rel, tables)
			if err != nil {
				return err
			}

			relMap[ftable] = append(existingRels, flipped)
		}
	}

	return nil
}

func flipRelationship[C, I any](r orm.Relationship, tables []drivers.Table[C, I]) (orm.Relationship, error) {
	name := r.Name
	if r.Local() == r.Foreign() {
		name += selfJoinSuffix
	}

	sideLen := len(r.Sides)
	flipped := orm.Relationship{
		Name:          name,
		Ignored:       r.Ignored,
		Sides:         make([]orm.RelSide, sideLen),
		NeverRequired: r.NeverRequired,
	}

	for i, side := range r.Sides {
		var from, to drivers.Table[C, I]
		for _, t := range tables {
			if t.Key == side.From {
				from = t
			}
			if t.Key == side.To {
				to = t
			}
			if from.Key != "" && to.Key != "" {
				break
			}
		}

		if from.Key == "" || to.Key == "" {
			continue
		}

		newModify, err := flipModify(side, tables)
		if err != nil {
			return orm.Relationship{}, err
		}

		flippedSide := orm.RelSide{
			To:   side.From,
			From: side.To,

			ToColumns:   side.FromColumns,
			FromColumns: side.ToColumns,
			ToWhere:     side.FromWhere,
			FromWhere:   side.ToWhere,
			IgnoredColumns: [2][]string{
				side.IgnoredColumns[1], side.IgnoredColumns[0],
			},

			Modify:      newModify,
			ToUnique:    side.FromUnique,
			FromUnique:  side.ToUnique,
			KeyNullable: side.KeyNullable,
		}
		flipped.Sides[sideLen-(1+i)] = flippedSide
	}

	return flipped, nil
}

func flipModify[C, I any](side orm.RelSide, tables []drivers.Table[C, I]) (string, error) {
	side.Modify = strings.ToLower(side.Modify)

	if side.Modify == "" {
		side.Modify = inferModify(side, tables)
	}

	if side.Modify == "from" {
		return "to", nil
	}

	if side.Modify == "to" {
		return "from", nil
	}

	return "", fmt.Errorf(`rel side modify should be "from" or "to",  got %q`, side.Modify)
}

func mergeRelationships(srcs, extras []orm.Relationship) []orm.Relationship {
Outer:
	for _, extra := range extras {
		for i, src := range srcs {
			if src.Name == extra.Name {
				srcs[i] = mergeRelationship(src, extra)
				continue Outer
			}
		}

		// No previous relationship was found, add it as-is
		srcs = append(srcs, extra)
	}

	final := make([]orm.Relationship, 0, len(srcs))
	for _, rel := range srcs {
		if rel.Ignored || len(rel.Sides) < 1 {
			continue
		}

		final = append(final, rel)
	}

	return final
}

func mergeRelationship(src, extra orm.Relationship) orm.Relationship {
	src.Ignored = extra.Ignored
	if len(extra.Sides) > 0 {
		src.Sides = extra.Sides
	}

	return src
}

// Returns true if the table has a unique constraint on exactly these columns
func allNullable[C, I any](t drivers.Table[C, I], cols ...string) bool {
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

func inferModify[C, I any](side orm.RelSide, tables drivers.Tables[C, I]) string {
	t1 := tables.Get(side.From)
	t2 := tables.Get(side.To)

	isT1PK := t1.Constraints.Primary != nil && internal.SliceMatch(side.FromColumns, t1.Constraints.Primary.Columns)
	isT2PK := t2.Constraints.Primary != nil && internal.SliceMatch(side.ToColumns, t2.Constraints.Primary.Columns)

	switch {
	case isT1PK && !isT2PK:
		return "to"
	case isT2PK && !isT1PK:
		return "from"
	}

	isT1Unique := t1.HasExactUnique(side.FromColumns...)
	isT2Unique := t2.HasExactUnique(side.ToColumns...)

	switch {
	case isT1Unique && !isT2Unique:
		return "to"
	case isT2Unique && !isT1Unique:
		return "from"
	}

	// Cannot infer, default to "to"
	return "to"
}

// processRelationshipConfig checks any user included relationships and adds them to the tables
func processRelationshipConfig[C, I any](config *Config[C], tables []drivers.Table[C, I], relMap Relationships) error {
	if len(tables) == 0 {
		return nil
	}

	setColumns(config.Relationships)
	if err := flipRelationships(config.Relationships, tables); err != nil {
		return err
	}

	for _, t := range tables {
		rels, ok := config.Relationships[t.Key]
		if !ok {
			continue
		}

		relMap[t.Key] = mergeRelationships(relMap[t.Key], rels)
	}

	return initRelationships(relMap, tables)
}

func validateRelationships(rels Relationships) error {
	for table, tableRels := range rels {
		for _, r := range tableRels {
			if err := r.Validate(); err != nil {
				return fmt.Errorf("%s: %w", table, err)
			}
		}
	}

	return nil
}

func setColumns(relMap Relationships) {
	for table, rels := range relMap {
		for relIdx, rel := range rels {
			for sideIdx, side := range rel.Sides {
				from := make([]string, 0, len(side.Columns))
				to := make([]string, 0, len(side.Columns))
				var ignored [2][]string

				for _, colpairs := range side.Columns {
					if colpairs[0] == "" {
						ignored[1] = append(ignored[1], colpairs[1])
						continue
					}

					if colpairs[1] == "" {
						ignored[0] = append(ignored[0], colpairs[0])
						continue
					}

					from = append(from, colpairs[0])
					to = append(to, colpairs[1])
				}

				relMap[table][relIdx].Sides[sideIdx].FromColumns = from
				relMap[table][relIdx].Sides[sideIdx].ToColumns = to
				relMap[table][relIdx].Sides[sideIdx].IgnoredColumns = ignored
			}
		}
	}
}
