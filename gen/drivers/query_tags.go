package drivers

import (
	"slices"
	"strconv"
	"strings"

	"github.com/huandu/xstrings"
	"github.com/volatiletech/strmangle"
)

// makeNamesDistinct reserves the original names first: if a query returns
// id, id, id2, the duplicate must not take the field name of the last column.
func makeNamesDistinct(names []string, sep string) {
	counts := make(map[string]int, len(names))
	for _, name := range names {
		counts[name] = 0
	}
	for i, name := range names {
		if name == "" {
			continue
		}
		counts[name]++
		if counts[name] == 1 {
			continue
		}
		for {
			candidate := name + sep + strconv.Itoa(counts[name])
			if _, used := counts[candidate]; !used {
				names[i] = candidate
				counts[candidate] = 1
				break
			}
			counts[name]++
		}
	}
}

// QueryResultTags configures the serialization tags on generated query rows.
type QueryResultTags struct {
	Keys   []string
	Casing string
	Ignore map[string]struct{}
}

// FieldTags returns the extra tags for one struct, using its resolved Go field
// names. DBName is only used for matching ignored result labels.
func (t QueryResultTags) FieldTags(cols QueryCols) []string {
	names := make([]string, len(cols))
	keys := t.Keys
	if slices.Contains(keys, "db") {
		keys = slices.DeleteFunc(slices.Clone(keys), func(k string) bool { return k == "db" })
	}
	if len(keys) == 0 {
		return names
	}

	for i, col := range cols {
		if t.ignored(col.DBName) {
			continue
		}
		switch t.Casing {
		case "alias", "title":
			names[i] = col.Name
		case "camel":
			names[i] = strmangle.CamelCase(xstrings.ToSnakeCase(col.Name))
		default:
			names[i] = xstrings.ToSnakeCase(col.Name)
		}
	}
	sep := ""
	if t.Casing == "" || t.Casing == "snake" {
		sep = "_"
	}
	makeNamesDistinct(names, sep)
	for i, name := range names {
		if name == "" {
			name = "-"
		}
		names[i] = strings.TrimSpace(strmangle.GenerateTags(keys, name))
	}
	return names
}

func (t QueryResultTags) ignored(label string) bool {
	if _, ok := t.Ignore[label]; ok {
		return true
	}
	// A nested result label can use either "." (to-many) or "__" (to-one).
	leaf := label[strings.LastIndexByte(label, '.')+1:]
	if i := strings.LastIndex(leaf, "__"); i >= 0 {
		leaf = leaf[i+2:]
	}
	_, ok := t.Ignore[leaf]
	return ok
}
