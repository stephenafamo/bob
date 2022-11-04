package gen

import (
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/spf13/cast"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/bob/orm/gen/drivers"
)

// Config for the running of the commands
type Config[T any] struct {
	Driver drivers.Interface[T] `toml:"driver,omitempty" json:"driver,omitempty"`

	PkgName           string   `toml:"pkg_name,omitempty" json:"pkg_name,omitempty"`
	OutFolder         string   `toml:"out_folder,omitempty" json:"out_folder,omitempty"`
	Tags              []string `toml:"tags,omitempty" json:"tags,omitempty"`
	NoTests           bool     `toml:"no_tests,omitempty" json:"no_tests,omitempty"`
	NoBackReferencing bool     `toml:"no_back_reference,omitempty" json:"no_back_reference,omitempty"`
	Wipe              bool     `toml:"wipe,omitempty" json:"wipe,omitempty"`
	StructTagCasing   string   `toml:"struct_tag_casing,omitempty" json:"struct_tag_casing,omitempty"`
	RelationTag       string   `toml:"relation_tag,omitempty" json:"relation_tag,omitempty"`
	TagIgnore         []string `toml:"tag_ignore,omitempty" json:"tag_ignore,omitempty"`

	Templates           []fs.FS          `toml:"-" json:"-"`
	CustomTemplateFuncs template.FuncMap `toml:"-" json:"-"`

	Aliases       Aliases                       `toml:"aliases,omitempty" json:"aliases,omitempty"`
	Replacements  []Replace                     `toml:"replacements,omitempty" json:"replacements,omitempty"`
	Relationships map[string][]orm.Relationship `toml:"relationships,omitempty" json:"relationships,omitempty"`
	Inflections   Inflections                   `toml:"inflections,omitempty" json:"inflections,omitempty"`

	Generator string `toml:"generator" json:"generator"`
}

// Replace replaces a column type with something else
type Replace struct {
	Tables  []string       `toml:"tables,omitempty" json:"tables,omitempty"`
	Match   drivers.Column `toml:"match,omitempty" json:"match,omitempty"`
	Replace drivers.Column `toml:"replace,omitempty" json:"replace,omitempty"`
}

type Inflections struct {
	Plural        map[string]string
	PluralExact   map[string]string
	Singular      map[string]string
	SingularExact map[string]string
	Irregular     map[string]string
}

// OutputDirDepth returns depth of output directory
func (c *Config[T]) OutputDirDepth() int {
	d := filepath.ToSlash(filepath.Clean(c.OutFolder))
	if d == "." {
		return 0
	}

	return strings.Count(d, "/") + 1
}

// ConvertAliases is necessary because viper
//
// It also supports two different syntaxes, because of viper:
//
//	[aliases.tables.table_name]
//	fields... = "values"
//	  [aliases.tables.columns]
//	  colname = "alias"
//	  [aliases.tables.relationships.fkey_name]
//	  local   = "x"
//	  foreign = "y"
//
// Or alternatively (when toml key names or viper's
// lowercasing of key names gets in the way):
//
//	[[aliases.tables]]
//	name = "table_name"
//	fields... = "values"
//	  [[aliases.tables.columns]]
//	  name  = "colname"
//	  alias = "alias"
//	  [[aliases.tables.relationships]]
//	  name    = "fkey_name"
//	  local   = "x"
//	  foreign = "y"
func ConvertAliases(i interface{}) Aliases {
	var a Aliases

	if i == nil {
		return a
	}

	topLevel := cast.ToStringMap(i)

	tablesIntf := topLevel["tables"]

	iterateMapOrSlice(tablesIntf, func(name string, tIntf interface{}) {
		if a.Tables == nil {
			a.Tables = make(map[string]TableAlias)
		}

		t := cast.ToStringMap(tIntf)

		var ta TableAlias

		if s := t["up_plural"]; s != nil {
			ta.UpPlural = s.(string)
		}
		if s := t["up_singular"]; s != nil {
			ta.UpSingular = s.(string)
		}
		if s := t["down_plural"]; s != nil {
			ta.DownPlural = s.(string)
		}
		if s := t["down_singular"]; s != nil {
			ta.DownSingular = s.(string)
		}

		if colsIntf, ok := t["columns"]; ok {
			ta.Columns = make(map[string]string)

			iterateMapOrSlice(colsIntf, func(name string, colIntf interface{}) {
				var alias string
				switch col := colIntf.(type) {
				case map[string]interface{}, map[interface{}]interface{}:
					cmap := cast.ToStringMap(colIntf)
					alias = cmap["alias"].(string)
				case string:
					alias = col
				}
				ta.Columns[name] = alias
			})
		}

		if relsIntf, ok := t["relationships"]; ok {
			ta.Relationships = make(map[string]string)

			iterateMapOrSlice(relsIntf, func(name string, relIntf interface{}) {
				var alias string
				switch rel := relIntf.(type) {
				case map[string]interface{}, map[interface{}]interface{}:
					cmap := cast.ToStringMap(relIntf)
					alias = cmap["alias"].(string)
				case string:
					alias = rel
				}
				ta.Relationships[name] = alias
			})
		}

		a.Tables[name] = ta
	})

	return a
}

func iterateMapOrSlice(mapOrSlice interface{}, fn func(name string, obj interface{})) {
	switch t := mapOrSlice.(type) {
	case map[string]interface{}, map[interface{}]interface{}:
		tmap := cast.ToStringMap(mapOrSlice)
		for name, table := range tmap {
			fn(name, table)
		}
	case []interface{}:
		for _, intf := range t {
			obj := cast.ToStringMap(intf)
			name := obj["name"].(string)
			fn(name, intf)
		}
	}
}

// ConvertReplacements is necessary because viper
func ConvertReplacements(i interface{}) []Replace {
	if i == nil {
		return nil
	}

	intfArray := i.([]interface{})
	replaces := make([]Replace, 0, len(intfArray))
	for _, r := range intfArray {
		replaceIntf := cast.ToStringMap(r)
		replace := Replace{}

		if replaceIntf["match"] == nil || replaceIntf["replace"] == nil {
			panic("replace types must specify both match and replace")
		}

		replace.Tables = tablesOfReplace(replaceIntf["match"])
		replace.Match = columnFromInterface(replaceIntf["match"])
		replace.Replace = columnFromInterface(replaceIntf["replace"])

		replaces = append(replaces, replace)
	}

	return replaces
}

// ConvertRelationships is necessary because viper
func ConvertRelationships(i interface{}) map[string][]orm.Relationship {
	if i == nil {
		return nil
	}

	config := cast.ToStringMap(i)
	if len(config) == 0 {
		return nil
	}

	relationships := make(map[string][]orm.Relationship, len(config))

	for table, relsIntf := range config {
		relsInterfaceSlice := cast.ToSlice(relsIntf)
		rels := make([]orm.Relationship, 0, len(relsInterfaceSlice))

		for _, relIntf := range relsInterfaceSlice {
			relMap := cast.ToStringMap(relIntf)
			rel := orm.Relationship{
				Name:        cast.ToString(relMap["name"]),
				ByJoinTable: cast.ToBool(relMap["by_join_table"]),
				Ignored:     cast.ToBool(relMap["ignored"]),
				Sides:       relSideSliceFromInterface(relMap["sides"]),
			}

			rels = append(rels, rel)
		}

		relationships[table] = rels
	}

	return relationships
}

func relSideSliceFromInterface(i any) []orm.RelSide {
	slice := cast.ToSlice(i)
	if len(slice) == 0 {
		return nil
	}

	sides := make([]orm.RelSide, len(slice))
	for i, sideIntf := range slice {
		sides[i] = relSideFromInterface(sideIntf)
	}

	return sides
}

func relSideFromInterface(i any) orm.RelSide {
	sideMap := cast.ToStringMap(i)
	return orm.RelSide{
		From:        cast.ToString(sideMap["from"]),
		FromColumns: cast.ToStringSlice(sideMap["from_columns"]),
		To:          cast.ToString(sideMap["to"]),
		ToColumns:   cast.ToStringSlice(sideMap["to_columns"]),
		ToKey:       cast.ToBool(sideMap["to_key"]),
		ToUnique:    cast.ToBool(sideMap["to_unique"]),
		KeyNullable: cast.ToBool(sideMap["key_nullable"]),
		FromWhere:   relWhereSliceFromInterface(sideMap["from_where"]),
		ToWhere:     relWhereSliceFromInterface(sideMap["to_where"]),
	}
}

func relWhereSliceFromInterface(i any) []orm.RelWhere {
	slice := cast.ToSlice(i)
	if len(slice) == 0 {
		return nil
	}

	wheres := make([]orm.RelWhere, len(slice))
	for i, whereIntf := range slice {
		wheres[i] = relWhereFromInterface(whereIntf)
	}

	return wheres
}

func relWhereFromInterface(i any) orm.RelWhere {
	relMap := cast.ToStringMap(i)
	return orm.RelWhere{
		Column: cast.ToString(relMap["column"]),
		Value:  cast.ToString(relMap["value"]),
	}
}

func tablesOfReplace(i interface{}) []string {
	tables := []string{}

	m := cast.ToStringMap(i)
	if s := m["tables"]; s != nil {
		tables = cast.ToStringSlice(s)
	}

	return tables
}

func columnFromInterface(i interface{}) drivers.Column {
	var col drivers.Column

	m := cast.ToStringMap(i)
	if s := m["name"]; s != nil {
		col.Name = s.(string)
	}
	if s := m["type"]; s != nil {
		col.Type = s.(string)
	}
	if s := m["db_type"]; s != nil {
		col.DBType = s.(string)
	}
	if s := m["udt_name"]; s != nil {
		col.UDTName = s.(string)
	}
	if s := m["full_db_type"]; s != nil {
		col.FullDBType = s.(string)
	}
	if s := m["arr_type"]; s != nil {
		col.ArrType = new(string)
		*col.ArrType = s.(string)
	}
	if s := m["domain_name"]; s != nil {
		col.DomainName = new(string)
		*col.DomainName = s.(string)
	}
	if b := m["auto_generated"]; b != nil {
		col.Generated = b.(bool)
	}
	if b := m["nullable"]; b != nil {
		col.Nullable = b.(bool)
	}
	if list := m["imports"]; list != nil {
		col.Imports = cast.ToStringSlice(list)
	}

	return col
}
