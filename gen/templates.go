package gen

import (
	"embed"
	"fmt"
	"io/fs"
	"strings"
	"text/template"
	"unicode"

	"github.com/Masterminds/sprig/v3"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/internal"
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
	DBInfo   fs.FS
}

type TemplateData[T, C, I any] struct {
	Dialect  string
	Importer language.Importer

	Table         drivers.Table[C, I]
	Tables        drivers.Tables[C, I]
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
	ExtraInfo T

	// Package information
	CurrentPackage string            // the current package being generated
	OutputPackages map[string]string // map of output keys to package paths

	// Driver is the module name of the underlying `database/sql` driver
	Driver   string
	Language language.Language
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

func enumValToIdentifier(val string) string {
	val = strings.ToLower(val)
	val = strings.ReplaceAll(val, "-", "_")
	val = strings.ReplaceAll(val, " ", "_")

	var newval strings.Builder
	for _, r := range val {
		if r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) {
			newval.WriteRune(r)
			continue
		}
		newval.WriteString(fmt.Sprintf("U%x", r))
	}

	// Title case after doing unicode replacements or they will be stripped
	return strmangle.TitleCase(newval.String())
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
