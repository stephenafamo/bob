package gen

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
	"text/template"
	"unicode"

	"github.com/Masterminds/sprig/v3"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/internal"
	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

//go:embed templates
var templates embed.FS

//go:embed bobgen-mysql/templates
var mysqlTemplates embed.FS

//go:embed bobgen-psql/templates
var psqlTemplates embed.FS

//go:embed bobgen-sqlite/templates
var sqliteTemplates embed.FS

//nolint:gochecknoglobals
var (
	BaseTemplates   = buildTemplatesFromKnownDirStructure(templates, "")
	MySQLTemplates  = buildTemplatesFromKnownDirStructure(mysqlTemplates, "bobgen-mysql")
	PSQLTemplates   = buildTemplatesFromKnownDirStructure(psqlTemplates, "bobgen-psql")
	SQLiteTemplates = buildTemplatesFromKnownDirStructure(sqliteTemplates, "bobgen-sqlite")
)

func buildTemplatesFromKnownDirStructure(templates fs.FS, dir string) Templates {
	if dir != "" {
		templates, _ = fs.Sub(templates, dir)
	}

	DBInfoTemplates, _ := fs.Sub(templates, "templates/dbinfo")
	EnumTemplates, _ := fs.Sub(templates, "templates/enums")
	ModelTemplates, _ := fs.Sub(templates, "templates/models")
	FactoryTemplates, _ := fs.Sub(templates, "templates/factory")
	QueriesTemplates, _ := fs.Sub(templates, "templates/queries")
	DBErrorTemplates, _ := fs.Sub(templates, "templates/dberrors")
	WhereTemplates, _ := fs.Sub(templates, "templates/where")
	LoadersTemplates, _ := fs.Sub(templates, "templates/loaders")
	JoinsTemplates, _ := fs.Sub(templates, "templates/joins")
	CountsTemplates, _ := fs.Sub(templates, "templates/counts")

	return Templates{
		DBInfo:   DBInfoTemplates,
		Enums:    EnumTemplates,
		Models:   ModelTemplates,
		Factory:  FactoryTemplates,
		Queries:  QueriesTemplates,
		DBErrors: DBErrorTemplates,
		Where:    WhereTemplates,
		Loaders:  LoadersTemplates,
		Joins:    JoinsTemplates,
		Counts:   CountsTemplates,
	}
}

type Templates struct {
	Enums    fs.FS
	Models   fs.FS
	Factory  fs.FS
	Queries  fs.FS
	DBErrors fs.FS
	Where    fs.FS
	Loaders  fs.FS
	Joins    fs.FS
	Counts   fs.FS
	DBInfo   fs.FS
}

type TemplateData[T, C, I any] struct {
	Dialect  string
	Importer language.Importer

	Table         drivers.Table[C, I]
	Tables        drivers.Tables[C, I]
	AllTables     drivers.Tables[C, I]
	TableNames    []string
	QueryFile     drivers.QueryFile
	QueryFolder   drivers.QueryFolder
	QueryFolders  []drivers.QueryFolder
	Enums         []drivers.Enum
	Aliases       drivers.Aliases
	Types         drivers.Types
	Relationships Relationships

	// Controls what names are output
	PkgName string

	// Control various generation features
	NoTests                     bool
	NoBackReferencing           bool
	SliceMutationMethods        bool
	RelationshipMutationMethods bool

	// Tags control which tags are added to the struct
	Tags []string
	// RelationTag controls the value of the tags for the Relationship struct
	RelationTag string
	// Generate struct tags as camelCase or snake_case
	StructTagCasing string
	// Contains field names that should have tags values set to '-'
	TagIgnore map[string]struct{}
	// Format for enum value identifiers: "title_case" or "screaming_snake_case"
	EnumFormat string

	// Supplied by the driver
	ExtraInfo T

	// Package information
	CurrentPackage string            // the current package being generated
	OutputPackages map[string]string // map of output keys to package paths
	ModelSplit     *ModelSplitData

	// Driver is the module name of the underlying `database/sql` driver
	Driver   string
	Language language.Language
}

func (d *TemplateData[T, C, I]) splitRef(tableKey, name string) string {
	if d.ModelSplit == nil || !d.ModelSplit.Enabled {
		return name
	}

	component := d.ModelSplit.TableComponents[tableKey]
	if component == nil {
		return name
	}

	if d.ModelSplit.Generation == modelSplitGenerationComponent &&
		d.ModelSplit.CurrentComponent != nil &&
		component.ID == d.ModelSplit.CurrentComponent.ID {
		return name
	}

	d.Importer.Import(component.PackagePath)
	return path.Base(component.PackagePath) + "." + name
}

func (d *TemplateData[T, C, I]) TableAlias(tableKey string) drivers.TableAlias {
	return d.Aliases.Table(tableKey)
}

func (d *TemplateData[T, C, I]) ModelType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular)
}

func (d *TemplateData[T, C, I]) SliceType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Slice")
}

func (d *TemplateData[T, C, I]) SetterType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Setter")
}

func (d *TemplateData[T, C, I]) QueryType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpPlural+"Query")
}

func (d *TemplateData[T, C, I]) TableVar(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpPlural)
}

func (d *TemplateData[T, C, I]) ColumnsType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Columns")
}

func (d *TemplateData[T, C, I]) BuildColumnsFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"Columns")
}

func (d *TemplateData[T, C, I]) WhereType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Where")
}

func (d *TemplateData[T, C, I]) BuildWhereFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"Where")
}

func (d *TemplateData[T, C, I]) JoinType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Joins")
}

func (d *TemplateData[T, C, I]) BuildJoinFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"Joins")
}

func (d *TemplateData[T, C, I]) PreloaderType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Preloader")
}

func (d *TemplateData[T, C, I]) BuildPreloaderFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"Preloader")
}

func (d *TemplateData[T, C, I]) ThenLoaderType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"ThenLoader")
}

func (d *TemplateData[T, C, I]) BuildThenLoaderFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"ThenLoader")
}

func (d *TemplateData[T, C, I]) CountPreloaderType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"CountPreloader")
}

func (d *TemplateData[T, C, I]) BuildCountPreloaderFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"CountPreloader")
}

func (d *TemplateData[T, C, I]) CountThenLoaderType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"CountThenLoader")
}

func (d *TemplateData[T, C, I]) BuildCountThenLoaderFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "Build"+alias.UpSingular+"CountThenLoader")
}

func (d *TemplateData[T, C, I]) FactoryTemplateType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Template")
}

func (d *TemplateData[T, C, I]) FactoryModType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Mod")
}

func (d *TemplateData[T, C, I]) FactoryModFuncType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"ModFunc")
}

func (d *TemplateData[T, C, I]) FactoryModSliceType(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"ModSlice")
}

func (d *TemplateData[T, C, I]) FactoryModsVar(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, alias.UpSingular+"Mods")
}

func (d *TemplateData[T, C, I]) FactoryNewWithContextFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "New"+alias.UpSingular+"WithContext")
}

func (d *TemplateData[T, C, I]) FactoryFromExistingFunc(tableKey string) string {
	alias := d.TableAlias(tableKey)
	return d.splitRef(tableKey, "FromExisting"+alias.UpSingular)
}

func (d *TemplateData[T, C, I]) FactoryRelDependencies(r orm.Relationship) string {
	needed := d.AllTables.NeededBridgeRels(r)
	ma := make([]string, len(needed))

	for i, need := range needed {
		alias := d.TableAlias(need.Table)
		ma[i] = fmt.Sprintf("%s *%s,", alias.DownSingular, d.FactoryTemplateType(need.Table))
	}

	return strings.Join(ma, "")
}

func (d *TemplateData[T, C, I]) FactoryRelDependenciesTyp(r orm.Relationship) string {
	needed := d.AllTables.NeededBridgeRels(r)
	ma := make([]string, len(needed))

	for i, need := range needed {
		alias := d.TableAlias(need.Table)
		ma[i] = fmt.Sprintf("%s *%s", alias.DownSingular, d.FactoryTemplateType(need.Table))
	}

	return strings.Join(ma, "\n")
}

func (d *TemplateData[T, C, I]) FactoryDependencyMethods(tableKey string) string {
	deps := make(map[string]struct{})
	for _, rel := range d.Relationships.Get(tableKey) {
		deps[rel.Foreign()] = struct{}{}
		for _, need := range d.AllTables.NeededBridgeRels(rel) {
			deps[need.Table] = struct{}{}
		}
	}

	tableKeys := make([]string, 0, len(deps))
	for tableKey := range deps {
		tableKeys = append(tableKeys, tableKey)
	}
	slices.Sort(tableKeys)

	methods := make([]string, 0, len(tableKeys)*2)
	for _, dep := range tableKeys {
		alias := d.TableAlias(dep)
		methods = append(methods,
			fmt.Sprintf("New%sWithContext(context.Context, ...%s) *%s", alias.UpSingular, d.FactoryModType(dep), d.FactoryTemplateType(dep)),
			fmt.Sprintf("FromExisting%s(*models.%s) *%s", alias.UpSingular, alias.UpSingular, d.FactoryTemplateType(dep)),
		)
	}

	return strings.Join(methods, "\n")
}

func (d *TemplateData[T, C, I]) RelDependenciesPos(r orm.Relationship) string {
	needed := d.AllTables.NeededBridgeRels(r)
	ma := make([]string, len(needed))

	for i, need := range needed {
		alias := d.TableAlias(need.Table)
		if need.Many {
			ma[i] = fmt.Sprintf(
				"%s%d %s,", alias.DownPlural, need.Position, d.SliceType(need.Table),
			)
		} else {
			ma[i] = fmt.Sprintf(
				"%s%d *%s,", alias.DownSingular, need.Position, d.ModelType(need.Table),
			)
		}
	}

	return strings.Join(ma, "")
}

func (d *TemplateData[T, C, I]) RelDependenciesPosArgs(r orm.Relationship) string {
	needed := d.AllTables.NeededBridgeRels(r)
	ma := make([]string, len(needed))

	for i, need := range needed {
		alias := d.TableAlias(need.Table)
		if need.Many {
			ma[i] = fmt.Sprintf("%s%d,", alias.DownPlural, need.Position)
		} else {
			ma[i] = fmt.Sprintf("%s%d,", alias.DownSingular, need.Position)
		}
	}

	return strings.Join(ma, "")
}

func loadTemplate(tpl *template.Template, customFuncs template.FuncMap, name, content string) error {
	_, err := tpl.New(name).
		Funcs(sprig.GenericFuncMap()).
		Funcs(templateFunctions).
		Funcs(customFuncs).
		Parse(content)
	if err != nil {
		return fmt.Errorf("failed to parse template: %s: %w", name, err)
	}

	return nil
}

// templateFunctions is a map of some helper functions that get passed into the
// templates. If you wish to pass a new function into your own template,
// you can add that with Config.CustomTemplateFuncs
//
//nolint:gochecknoglobals
var templateFunctions = template.FuncMap{
	"titleCase":          strmangle.TitleCase,
	"camelCase":          strmangle.CamelCase,
	"ignore":             strmangle.Ignore,
	"generateTags":       strmangle.GenerateTags,
	"generateIgnoreTags": strmangle.GenerateIgnoreTags,
	"normalizeType":      NormalizeType,
	"enumVal":            enumValToIdentifier,
	"enumValScreaming":   enumValToScreamingSnakeCase,
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
	"isPrimitiveType":    isPrimitiveType,
	"relQueryMethodName": relQueryMethodName,
}

func enumValNormalize(val string) string {
	val = strings.ReplaceAll(val, "-", "_")
	val = strings.ReplaceAll(val, " ", "_")

	var newval strings.Builder
	for _, r := range val {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			newval.WriteRune(r)
			continue
		}
		fmt.Fprintf(&newval, "U%x", r)
	}

	return newval.String()
}

func enumValToIdentifier(val string) string {
	val = strings.ToLower(val)
	val = enumValNormalize(val)

	// Title case after doing unicode replacements or they will be stripped
	return strmangle.TitleCase(val)
}

func enumValToScreamingSnakeCase(val string) string {
	val = strings.ToUpper(val)
	val = enumValNormalize(val)

	return val
}

func relQueryMethodName(tAlias drivers.TableAlias, relAlias string) string {
	for _, colAlias := range tAlias.Columns {
		// clash with field name
		if colAlias == relAlias {
			return "Related" + relAlias
		}
	}

	return relAlias
}

func NormalizeType(val string) string {
	return internal.TypesReplacer.Replace(val)
}
