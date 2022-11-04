package gen

import (
	"embed"
	"encoding"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/stephenafamo/bob/orm"
	"github.com/stephenafamo/bob/orm/gen/drivers"
	"github.com/stephenafamo/bob/orm/gen/importers"
	"github.com/volatiletech/strmangle"
)

//go:embed templates
var Templates embed.FS

type Importer map[string]struct{}

// To be used inside templates to record an import.
// Always returns an empty string
func (i Importer) Import(pkgs ...string) string {
	if len(pkgs) < 1 {
		return ""
	}
	pkg := fmt.Sprintf("%q", pkgs[0])
	if len(pkgs) > 1 {
		pkg = fmt.Sprintf("%s %q", pkgs[0], pkgs[1])
	}

	i[pkg] = struct{}{}
	return ""
}

func (i Importer) ImportList(list importers.List) string {
	for _, p := range list {
		i[p] = struct{}{}
	}
	return ""
}

func (i Importer) ToList() importers.List {
	var list importers.List
	for pkg := range i {
		list = append(list, pkg)
	}

	return list
}

type templateData[T any] struct {
	Dialect  string
	Importer Importer

	Table   drivers.Table
	Tables  []drivers.Table
	Aliases Aliases

	// Controls what names are output
	PkgName string
	Schema  string

	// Control various generation features
	AddSoftDeletes    bool
	AddEnumTypes      bool
	EnumNullPrefix    string
	NoTests           bool
	NoHooks           bool
	NoAutoTimestamps  bool
	NoBackReferencing bool

	// Tags control which tags are added to the struct
	Tags []string
	// RelationTag controls the value of the tags for the Relationship struct
	RelationTag string
	// Generate struct tags as camelCase or snake_case
	StructTagCasing string
	// Contains field names that should have tags values set to '-'
	TagIgnore map[string]struct{}

	// Supplied by the driver
	ExtraInfo T
}

func (t *templateData[T]) ResetImports() {
	t.Importer = make(Importer)
}

type templateList struct {
	*template.Template
}

type templateNameList []string

func (t templateNameList) Len() int {
	return len(t)
}

func (t templateNameList) Swap(k, j int) {
	t[k], t[j] = t[j], t[k]
}

func (t templateNameList) Less(k, j int) bool {
	// Make sure "struct" goes to the front
	if t[k] == "struct.tpl" {
		return true
	}

	res := strings.Compare(t[k], t[j])
	return res <= 0
}

// Templates returns the name of all the templates defined in the template list
func (t templateList) Templates() []string {
	tplList := t.Template.Templates()

	if len(tplList) == 0 {
		return nil
	}

	ret := make([]string, 0, len(tplList))
	for _, tpl := range tplList {
		if name := tpl.Name(); strings.HasSuffix(name, ".tpl") {
			ret = append(ret, name)
		}
	}

	sort.Sort(templateNameList(ret))

	return ret
}

func loadTemplates(lazyTemplates []lazyTemplate, testTemplates bool, customFuncs template.FuncMap) (*templateList, error) {
	tpl := template.New("")

	for _, t := range lazyTemplates {
		firstDir := strings.Split(t.Name, string(filepath.Separator))[0]
		isTest := firstDir == "test" || strings.HasSuffix(firstDir, "_test")
		if testTemplates && !isTest || !testTemplates && isTest {
			continue
		}

		byt, err := t.Loader.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load template: %s: %w", t.Name, err)
		}

		_, err = tpl.New(t.Name).
			Funcs(sprig.GenericFuncMap()).
			Funcs(templateFunctions).
			Funcs(customFuncs).
			Parse(string(byt))
		if err != nil {
			return nil, fmt.Errorf("failed to parse template: %s: %w", t.Name, err)
		}
	}

	return &templateList{Template: tpl}, nil
}

type lazyTemplate struct {
	Name   string         `json:"name"`
	Loader templateLoader `json:"loader"`
}

type templateLoader interface {
	encoding.TextMarshaler
	Load() ([]byte, error)
}

type assetLoader struct {
	fs   fs.FS
	name string
}

func (a assetLoader) Load() ([]byte, error) {
	return fs.ReadFile(a.fs, string(a.name))
}

func (a assetLoader) MarshalText() ([]byte, error) {
	return []byte(a.String()), nil
}

func (a assetLoader) String() string {
	return "asset:" + string(a.name)
}

// templateFunctions is a map of some helper functions that get passed into the
// templates. If you wish to pass a new function into your own template,
// you can add that with Config.CustomTemplateFuncs
//
//nolint:gochecknoglobals
var templateFunctions = template.FuncMap{
	"titleCase":          strmangle.TitleCase,
	"ignore":             strmangle.Ignore,
	"generateTags":       strmangle.GenerateTags,
	"generateIgnoreTags": strmangle.GenerateIgnoreTags,
	"dbTag": func(t drivers.Table, c drivers.Column) string {
		tag := c.Name
		if t.PKey != nil {
			for _, pkc := range t.PKey.Columns {
				if pkc == c.Name {
					tag += ",pk"
				}
			}
		}
		if c.Generated {
			tag += ",generated"
		}
		return tag
	},
	"columnTagName": func(casing, name, alias string) string {
		switch casing {
		case "camel":
			return strmangle.CamelCase(name)
		case "title":
			return strmangle.TitleCase(name)
		case "alias":
			return alias
		default:
			return name
		}
	},
	"columnGetter": columnGetter,
	"getColumn":    getColumn,
	"quoteAndJoin": func(s1, s2 string) string {
		if s1 == "" && s2 == "" {
			return ""
		}

		if s1 == "" {
			return fmt.Sprintf("%q", s2)
		}

		if s2 == "" {
			return fmt.Sprintf("%q", s1)
		}

		return fmt.Sprintf("%q, %q", s1, s2)
	},
	"relDependencies":     relDependencies,
	"createDeps":          createDeps,
	"insertDeps":          insertDeps,
	"setDeps":             setDeps,
	"relatedUpdateValues": relatedUpdateValues,
}

func getColumn(t []drivers.Table, table string, a TableAlias, column string) drivers.Column {
	for _, t := range t {
		if t.Name != table {
			continue
		}

		return t.GetColumn(column)
	}

	panic("unknown table " + table)
}

func columnGetter(tables []drivers.Table, table string, a TableAlias, column string) string {
	for _, t := range tables {
		if t.Name != table {
			continue
		}

		col := t.GetColumn(column)
		colAlias := a.Column(column)
		if !col.Nullable {
			return colAlias
		}

		return fmt.Sprintf("%s.GetOrZero()", colAlias)
	}

	panic("unknown table " + table)
}

func columnSetter(i Importer, tables []drivers.Table, table string, column, to string) string {
	for _, t := range tables {
		if t.Name != table {
			continue
		}

		col := t.GetColumn(column)
		if !col.Nullable {
			return to
		}

		i.Import("github.com/aarondl/opt/null")
		return fmt.Sprintf("null.From(%s)", to)
	}

	panic("unknown table " + table)
}

func relDependencies(aliases Aliases, r orm.Relationship) string {
	ma := []string{}
	for _, need := range r.NeededColumns() {
		alias := aliases.Tables[need]
		ma = append(ma, fmt.Sprintf("%s *%s,", alias.DownSingular, alias.UpSingular))
	}

	return strings.Join(ma, "")
}

func createDeps(aliases Aliases, r orm.Relationship) string {
	local := r.Local()
	foreign := r.Foreign()
	ksides := r.ValuedSides()
	needed := r.NeededColumns()

	created := make([]string, 0, len(ksides))
	for _, kside := range ksides {
		shouldCreate := shouldCreateObjs(kside.TableName, local, foreign, needed)
		if !shouldCreate {
			continue
		}

		objVarName := getVarName(aliases, kside.TableName, local, foreign, true)
		oalias := aliases.Tables[kside.TableName]

		created = append(created, fmt.Sprintf(`var %s []*Optional%s`,
			objVarName,
			oalias.UpSingular,
		))
	}

	return strings.Join(created, "\n")
}

func insertDeps(aliases Aliases, r orm.Relationship) string {
	local := r.Local()
	foreign := r.Foreign()
	ksides := r.ValuedSides()
	needed := r.NeededColumns()

	insert := make([]string, 0, len(ksides))
	for _, kside := range ksides {
		shouldCreate := shouldCreateObjs(kside.TableName, local, foreign, needed)
		if !shouldCreate {
			continue
		}

		objVarName := getVarName(aliases, kside.TableName, local, foreign, true)
		oalias := aliases.Tables[kside.TableName]

		insert = append(insert, fmt.Sprintf(`
			  _, err = %sTable.InsertMany(ctx, exec, %s...)
			  if err != nil {
				  return fmt.Errorf("inserting related objects: %%w", err)
			  }
			`,
			oalias.UpPlural,
			objVarName,
		))
	}

	return strings.Join(insert, "\n")
}

func setDeps(i Importer, tables []drivers.Table, aliases Aliases, r orm.Relationship, skipForeign bool) string {
	local := r.Local()
	foreign := r.Foreign()
	ksides := r.ValuedSides()
	needed := r.NeededColumns()

	ret := make([]string, 0, len(ksides))
	for _, kside := range ksides {
		if skipForeign && kside.TableName == foreign {
			continue
		}

		mret := make([]string, 0, len(kside.Mapped))
		objVarName := getVarName(aliases, kside.TableName, local, foreign, false)
		oalias := aliases.Tables[kside.TableName]

		switch shouldSetObjs(kside.TableName, local, needed) {
		case false:
			i.Import("github.com/stephenafamo/bob/orm")
			for _, mapp := range kside.Mapped {
				oGetter := columnGetter(tables, kside.TableName, oalias, mapp.Column)

				if mapp.Value != "" {
					mret = append(mret, fmt.Sprintf(`if %s.%s != %s {
						return &orm.BadRelationshipChainError{
						    Table1: %q, Column1: %q, Value: %q,
						}
					}`,
						objVarName, oGetter, mapp.Value,
						kside.TableName, mapp.Column, mapp.Value,
					))
					continue
				}

				extObjVarName := getVarName(aliases, mapp.ExternalTable, local, foreign, false)
				malias := aliases.Tables[mapp.ExternalTable]

				mGetter := columnGetter(tables, mapp.ExternalTable, malias, mapp.ExternalColumn)

				mret = append(mret, fmt.Sprintf(`if %s.%s != %s.%s {
						return &orm.BadRelationshipChainError{
						    Table1: %q, Column1: %q,
						    Table2: %q, Column2: %q,
						}
					}`,
					objVarName, oGetter,
					extObjVarName, mGetter,
					kside.TableName, mapp.Column,
					mapp.ExternalTable, mapp.ExternalColumn,
				))
			}

			ret = append(ret, strings.Join(mret, "\n"))
		case true:
			objVarNamePlural := getVarName(aliases, kside.TableName, local, foreign, true)
			shouldCreate := shouldCreateObjs(kside.TableName, local, foreign, needed)
			if shouldCreate {
				mret = append(mret, fmt.Sprintf("%s := %s[i]",
					objVarName, objVarNamePlural,
				))
			}

			for _, mapp := range kside.Mapped {
				if mapp.Value != "" {
					mret = append(mret, fmt.Sprintf(`%s.%s = %s`,
						objVarName,
						oalias.Columns[mapp.Column],
						mapp.Value,
					))
					continue
				}

				extObjVarName := getVarName(aliases, mapp.ExternalTable, local, foreign, false)
				malias := aliases.Tables[mapp.ExternalTable]
				oSetter := columnSetter(i, tables, kside.TableName, mapp.Column, fmt.Sprintf(
					"%s.%s",
					extObjVarName,
					malias.Columns[mapp.ExternalColumn],
				))

				mret = append(mret, fmt.Sprintf(`%s.%s = %s`,
					objVarName,
					oalias.Columns[mapp.Column],
					oSetter,
				))
			}

			ret = append(ret, strings.Join(mret, "\n"))
		}
	}

	return strings.Join(ret, "\n")
}

func relatedUpdateValues(i Importer, tables []drivers.Table, aliases Aliases, r orm.Relationship, skipForeign bool) string {
	local := r.Local()
	foreign := r.Foreign()
	ksides := r.ValuedSides()

	for _, kside := range ksides {
		if kside.TableName != foreign {
			continue
		}

		oalias := aliases.Tables[kside.TableName]

		mret := make([]string, 0, len(kside.Mapped))
		for _, mapp := range kside.Mapped {
			malias := aliases.Tables[mapp.ExternalTable]
			extObjVarName := getVarName(aliases, mapp.ExternalTable, local, foreign, false)

			oSetter := columnSetter(i, tables, kside.TableName, mapp.Column, fmt.Sprintf(
				"%s.%s",
				extObjVarName,
				malias.Columns[mapp.ExternalColumn],
			))

			mret = append(mret, fmt.Sprintf("%s: %s,",
				oalias.Columns[mapp.Column],
				oSetter,
			))
		}

		return strings.Join(mret, "\n")
	}

	return ""
}

func getVarName(aliases Aliases, tableName, local, foreign string, plural bool) string {
	switch {
	case tableName == local:
		return "o"

	case tableName == foreign:
		if plural {
			return "rels"
		}
		return "rel"

	default:
		alias := aliases.Tables[tableName]
		if plural {
			return alias.DownPlural
		}
		return alias.DownSingular
	}
}

func shouldSetObjs(tableName, local string, needed []string) bool {
	if tableName == local {
		return false
	}

	for _, n := range needed {
		if tableName == n {
			return false
		}
	}

	return true
}

func shouldCreateObjs(tableName, local, foreign string, needed []string) bool {
	if tableName == local {
		return false
	}

	if tableName == foreign {
		return false
	}

	for _, n := range needed {
		if tableName == n {
			return false
		}
	}

	return true
}
