package gen

import (
	"strings"

	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

// returns the last part of a dot.separated.string
func last(s string) string {
	ss := strings.Split(s, ".")
	return ss[len(ss)-1]
}

func relAlias(rels []orm.Relationship) map[string]string {
	aliases := map[string]string{}

	for _, rel := range rels {
		// When not a direct relationship we just use the table name
		if len(rel.Sides) > 1 {
			aliases[rel.Name] = formatRelAlias(rel, last(rel.Sides[len(rel.Sides)-1].To))
			continue
		}

		side := rel.Sides[0]

		// Just cop out and use the table name if there are multiple colummns
		if len(rel.Sides[0].FromColumns) > 1 {
			aliases[rel.Name] = formatRelAlias(rel, last(side.To))
			continue
		}
		var lcol, fcol string
		for i, l := range side.FromColumns {
			lcol = l
			fcol = side.ToColumns[i]
		}

		lcolTrimmed := strmangle.Singular(trimSuffixes(lcol))
		fcolTrimmed := strmangle.Singular(trimSuffixes(fcol))

		singularLocalTable := strmangle.Singular(last(side.From))
		singularForeignTable := strmangle.Singular(last(side.To))

		if lcolTrimmed == singularForeignTable || fcolTrimmed == singularLocalTable {
			aliases[rel.Name] = formatRelAlias(rel, last(side.To))
			continue
		}

		// Just use the longer column name
		// Anything more specific and the user should just set it up
		colToUse := lcolTrimmed
		if len(fcolTrimmed) > len(lcolTrimmed) {
			colToUse = fcolTrimmed
		}

		if side.To == side.From {
			// Handle special case of self-join
			aliases[rel.Name] = formatRelAlias(rel, colToUse)
			continue
		}

		aliases[rel.Name] = formatRelAlias(rel, colToUse+"_"+last(side.To))
	}

	for k, alias := range aliases {
		if strings.HasSuffix(k, selfJoinSuffix) {
			aliases[k] = "Reverse" + alias
		}
	}

	return aliases
}

//nolint:gochecknoglobals
var identifierSuffixes = []string{"_id", "_uuid", "_guid", "_oid"}

// trimSuffixes from the identifier
func trimSuffixes(str string) string {
	ln := len(str)
	for _, s := range identifierSuffixes {
		str = strings.TrimSuffix(str, s)
		if len(str) != ln {
			break
		}
	}

	return str
}

func formatRelAlias(rel orm.Relationship, name string) string {
	if rel.IsToMany() {
		return strmangle.TitleCase(strmangle.Plural(name))
	}

	return strmangle.TitleCase(strmangle.Singular(name))
}
