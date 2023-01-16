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
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/importers"
	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

//go:embed templates
var templates embed.FS

//go:embed bobgen-mysql/templates
var mysqlTemplates embed.FS

//go:embed bobgen-prisma/templates
var prismaTemplates embed.FS

//nolint:gochecknoglobals
var (
	ModelTemplates, _       = fs.Sub(templates, "templates/models")
	FactoryTemplates, _     = fs.Sub(templates, "templates/factory")
	MySQLModelTemplates, _  = fs.Sub(mysqlTemplates, "bobgen-mysql/templates/models")
	PrismaModelTemplates, _ = fs.Sub(prismaTemplates, "bobgen-prisma/templates/models")
)

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
	Enums   []drivers.Enum
	Aliases Aliases

	// Controls what names are output
	PkgName string

	// Control various generation features
	AddSoftDeletes    bool
	AddEnumTypes      bool
	EnumNullPrefix    string
	NoTests           bool
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
	ExtraInfo     T
	ModelsPackage string

	// If the driver cannot bulk insert and return concrete objects
	// then we have to insert relationships one-by-one
	CanBulkInsert bool
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
	"getTable":           drivers.GetTable,
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
		if c.AutoIncr {
			tag += ",autoincr"
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
	"uniqueColPairs":        uniqueColPairs,
	"relArgs":               relArgs,
	"relDependencies":       relDependencies,
	"relDependenciesTyp":    relDependenciesTyp,
	"relDependenciesTypSet": relDependenciesTypSet,
	"relIsRequired":         relIsRequired,
	"createDeps":            createDeps,
	"insertDeps":            insertDeps,
	"setModelDeps":          setModelDeps,
	"setFactoryDeps":        setFactoryDeps,
	"relatedUpdateValues":   relatedUpdateValues,
}

func getColumn(t []drivers.Table, table string, a TableAlias, column string) drivers.Column {
	for _, t := range t {
		if t.Key != table {
			continue
		}

		return t.GetColumn(column)
	}

	panic("unknown table " + table)
}

func columnGetter(tables []drivers.Table, table string, a TableAlias, column string) string {
	for _, t := range tables {
		if t.Key != table {
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

func columnSetter(i Importer, aliases Aliases, tables []drivers.Table, fromTName, toTName, fromColName, toColName, varName string, fromOpt, toOpt bool) string {
	fromTable := drivers.GetTable(tables, fromTName)
	fromCol := fromTable.GetColumn(fromColName)

	toTable := drivers.GetTable(tables, toTName)
	toCol := toTable.GetColumn(toColName)
	to := fmt.Sprintf("%s.%s", varName, aliases.Tables[toTName].Columns[toColName])

	switch {
	case (fromOpt == toOpt) && (toCol.Nullable == fromCol.Nullable):
		// If both type match, return it plainly
		return to

	case !fromOpt && !fromCol.Nullable:
		// if from is concrete, then use MustGet()
		return fmt.Sprintf("%s.MustGet()", to)

	case fromOpt && fromCol.Nullable && !toOpt && !toCol.Nullable:
		i.Import("github.com/aarondl/opt/omitnull")
		return fmt.Sprintf("omitnull.From(%s)", to)

	case fromOpt && fromCol.Nullable && !toOpt && toCol.Nullable:
		i.Import("github.com/aarondl/opt/omitnull")
		return fmt.Sprintf("omitnull.FromNull(%s)", to)

	case fromOpt && fromCol.Nullable && toOpt && !toCol.Nullable:
		i.Import("github.com/aarondl/opt/omitnull")
		return fmt.Sprintf("omitnull.FromOmit(%s)", to)

	default:
		// from is either omit or null
		val := "omit"
		if fromCol.Nullable {
			val = "null"
		}

		i.Import(fmt.Sprintf("github.com/aarondl/opt/%s", val))

		switch {
		case !toOpt && !toCol.Nullable:
			return fmt.Sprintf("%s.From(%s)", val, to)

		default:
			return fmt.Sprintf("%s.FromCond(%s.GetOrZero(), %s.IsSet())", val, to, to)
		}
	}
}

func relIsRequired(t drivers.Table, r orm.Relationship) bool {
	// // multi sided relationships are always non-required
	// if len(r.Sides) > 1 {
	// return false
	// }
	firstSide := r.Sides[0]
	if firstSide.ToKey {
		return false
	}

	for _, colName := range firstSide.FromColumns {
		if t.GetColumn(colName).Nullable {
			return false
		}
	}

	return true
}

func relArgs(aliases Aliases, r orm.Relationship, preSuf ...string) string {
	var prefix, suffix string
	if len(preSuf) > 0 {
		prefix = preSuf[0]
	}
	if len(preSuf) > 1 {
		suffix = preSuf[1]
	}

	ma := []string{}
	for _, need := range r.NeededColumns() {
		ma = append(ma, fmt.Sprintf(
			"%s%s%s,", aliases.Tables[need].DownSingular, prefix, suffix,
		))
	}

	return strings.Join(ma, "")
}

func relDependencies(aliases Aliases, r orm.Relationship, preSuf ...string) string {
	var prefix, suffix string
	if len(preSuf) > 0 {
		prefix = preSuf[0]
	}
	if len(preSuf) > 1 {
		suffix = preSuf[1]
	}
	ma := []string{}
	for _, need := range r.NeededColumns() {
		alias := aliases.Tables[need]
		ma = append(ma, fmt.Sprintf(
			"%s *%s%s%s,", alias.DownSingular, alias.UpSingular, prefix, suffix,
		))
	}

	return strings.Join(ma, "")
}

func relDependenciesTyp(aliases Aliases, r orm.Relationship) string {
	ma := []string{}

	for _, need := range r.NeededColumns() {
		alias := aliases.Tables[need]
		ma = append(ma, fmt.Sprintf("%s *%sTemplate", alias.DownSingular, alias.UpSingular))
	}

	return strings.Join(ma, "\n")
}

func relDependenciesTypSet(aliases Aliases, r orm.Relationship) string {
	ma := []string{}

	for _, need := range r.NeededColumns() {
		alias := aliases.Tables[need]
		ma = append(ma, fmt.Sprintf("%s: %s,", alias.DownSingular, alias.DownSingular))
	}

	return strings.Join(ma, "\n")
}

func createDeps(aliases Aliases, r orm.Relationship, many bool) string {
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

		objVarName := getVarName(aliases, kside.TableName, local, foreign, many)
		oalias := aliases.Tables[kside.TableName]

		if many {
			created = append(created, fmt.Sprintf(`var %s []*%sSetter`,
				objVarName,
				oalias.UpSingular,
			))
		} else {
			created = append(created, fmt.Sprintf(`%s := &%sSetter{}`,
				objVarName,
				oalias.UpSingular,
			))
		}
	}

	return strings.Join(created, "\n")
}

func insertDeps(aliases Aliases, r orm.Relationship, many bool) string {
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

		objVarName := getVarName(aliases, kside.TableName, local, foreign, many)
		oalias := aliases.Tables[kside.TableName]

		if many {
			insert = append(insert, fmt.Sprintf(`
			  _, err = %sTable.InsertMany(ctx, exec, %s...)
			  if err != nil {
				  return fmt.Errorf("inserting related objects: %%w", err)
			  }
			`,
				oalias.UpPlural,
				objVarName,
			))
		} else {
			insert = append(insert, fmt.Sprintf(`
			  _, err = %sTable.Insert(ctx, exec, %s)
			  if err != nil {
				  return fmt.Errorf("inserting related object: %%w", err)
			  }
			`,
				oalias.UpPlural,
				objVarName,
			))
		}
	}

	return strings.Join(insert, "\n")
}

func setModelDeps(i Importer, tables []drivers.Table, aliases Aliases, r orm.Relationship, inLoop, optional bool) string {
	opts := setDepsOptions{
		i:       i,
		tables:  tables,
		aliases: aliases,
		r:       r,
		inLoop:  inLoop,
		fromOpt: func(side orm.RelSetDetails) bool { return false },
		toOpt:   false,
	}

	if optional {
		foreign := r.Foreign()
		opts.fromOpt = func(side orm.RelSetDetails) bool { return side.TableName == foreign }
	}

	return setDeps(opts)
}

func setFactoryDeps(i Importer, tables []drivers.Table, aliases Aliases, r orm.Relationship, inLoop bool) string {
	local := r.Local()
	foreign := r.Foreign()
	ksides := r.ValuedSides()

	ret := make([]string, 0, len(ksides))
	for _, kside := range ksides {
		switch kside.TableName {
		case local, foreign:
		default:
			continue
		}

		mret := make([]string, 0, len(kside.Mapped))

		for _, mapp := range kside.Mapped {
			switch mapp.ExternalTable {
			case local, foreign:
			default:
				continue
			}

			oalias := aliases.Tables[kside.TableName]
			objVarName := getVarName(aliases, kside.TableName, local, foreign, false)

			if mapp.Value != "" {
				oGetter := columnGetter(tables, kside.TableName, oalias, mapp.Column)

				if kside.TableName == r.Local() {
					i.Import("github.com/stephenafamo/bob/orm")
					mret = append(mret, fmt.Sprintf(`if %s.%s != %s {
								return &orm.RelationshipChainError{
									Table1: %q, Column1: %q, Value: %q,
								}
							}`,
						objVarName, oGetter, mapp.Value,
						kside.TableName, mapp.Column, mapp.Value,
					))
					continue
				}

				mret = append(mret, fmt.Sprintf(`%s.%s = %s`,
					objVarName,
					oalias.Columns[mapp.Column],
					mapp.Value,
				))
				continue
			}

			extObjVarName := getVarName(aliases, mapp.ExternalTable, local, foreign, false)

			oSetter := columnSetter(i, aliases, tables,
				kside.TableName, mapp.ExternalTable,
				mapp.Column, mapp.ExternalColumn,
				extObjVarName, false, false)

			mret = append(mret, fmt.Sprintf(`%s.%s = %s`,
				objVarName,
				oalias.Columns[mapp.Column],
				oSetter,
			))
		}

		ret = append(ret, strings.Join(mret, "\n"))
	}

	return strings.Join(ret, "\n")
}

type setDepsOptions struct {
	i           Importer
	tables      []drivers.Table
	aliases     Aliases
	r           orm.Relationship
	inLoop      bool
	fromOpt     func(orm.RelSetDetails) bool
	toOpt       bool
	skipCreated bool
}

func setDeps(o setDepsOptions) string {
	local := o.r.Local()
	foreign := o.r.Foreign()
	ksides := o.r.ValuedSides()
	needed := o.r.NeededColumns()

	ret := make([]string, 0, len(ksides))
	for _, kside := range ksides {
		mret := make([]string, 0, len(kside.Mapped))
		objVarName := getVarName(o.aliases, kside.TableName, local, foreign, false)
		oalias := o.aliases.Tables[kside.TableName]

		objVarNamePlural := getVarName(o.aliases, kside.TableName, local, foreign, true)
		shouldCreate := shouldCreateObjs(kside.TableName, local, foreign, needed)

		if shouldCreate && o.skipCreated {
			continue
		}

		if shouldCreate && o.inLoop {
			mret = append(mret, fmt.Sprintf("%s := %s[i]",
				objVarName, objVarNamePlural,
			))
		}

		for _, mapp := range kside.Mapped {
			oGetter := columnGetter(o.tables, kside.TableName, oalias, mapp.Column)
			fromOpt := shouldCreate || (o.fromOpt != nil && o.fromOpt(kside))

			if mapp.Value != "" {
				if kside.TableName == o.r.Local() {
					o.i.Import("github.com/stephenafamo/bob/orm")
					mret = append(mret, fmt.Sprintf(`if %s.%s != %s {
								return &orm.RelationshipChainError{
									Table1: %q, Column1: %q, Value: %q,
								}
							}`,
						objVarName, oGetter, mapp.Value,
						kside.TableName, mapp.Column, mapp.Value,
					))
					continue
				}

				oSetter := mapp.Value
				if fromOpt {
					oSetter = fmt.Sprintf("omit.From(%s)", oSetter)
				}

				mret = append(mret, fmt.Sprintf(`%s.%s = %s`,
					objVarName,
					oalias.Columns[mapp.Column],
					oSetter,
				))
				continue
			}

			extObjVarName := getVarName(o.aliases, mapp.ExternalTable, local, foreign, false)

			oSetter := columnSetter(o.i, o.aliases, o.tables,
				kside.TableName, mapp.ExternalTable,
				mapp.Column, mapp.ExternalColumn,
				extObjVarName, fromOpt, o.toOpt)

			mret = append(mret, fmt.Sprintf(`%s.%s = %s`,
				objVarName,
				oalias.Columns[mapp.Column],
				oSetter,
			))
		}

		ret = append(ret, strings.Join(mret, "\n"))
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
			if mapp.Value != "" {
				mret = append(mret, fmt.Sprintf("%s: omit.From(%s),",
					oalias.Columns[mapp.Column],
					mapp.Value,
				))
				continue
			}

			extObjVarName := getVarName(aliases, mapp.ExternalTable, local, foreign, false)

			oSetter := columnSetter(i, aliases, tables,
				kside.TableName, mapp.ExternalTable,
				mapp.Column, mapp.ExternalColumn,
				extObjVarName, true, false)

			mret = append(mret, fmt.Sprintf("%s: %s,",
				oalias.Columns[mapp.Column],
				oSetter,
			))
		}

		return strings.Join(mret, "\n")
	}

	return ""
}

func getVarName(aliases Aliases, tableName, local, foreign string, many bool) string {
	switch {
	case tableName == local:
		return "o"

	case tableName == foreign:
		if many {
			return "rels"
		}
		return "rel"

	default:
		alias := aliases.Tables[tableName]
		if many {
			return alias.DownPlural
		}
		return alias.DownSingular
	}
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

func uniqueColPairs(t drivers.Table) string {
	ret := make([]string, 0, len(t.Uniques)+1)
	if t.PKey != nil {
		ret = append(ret, fmt.Sprintf("%#v", t.PKey.Columns))
	}

	for _, unique := range t.Uniques {
		ret = append(ret, fmt.Sprintf("%#v", unique.Columns))
	}

	return strings.Join(ret, ", ")
}
