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
	ModelTemplates, _       = fs.Sub(templates, "templates/models")
	FactoryTemplates, _     = fs.Sub(templates, "templates/factory")
	QueriesTemplates, _     = fs.Sub(templates, "templates/queries")
	MySQLModelTemplates, _  = fs.Sub(mysqlTemplates, "bobgen-mysql/templates/models")
	PSQLModelTemplates, _   = fs.Sub(psqlTemplates, "bobgen-psql/templates/models")
	SQLiteModelTemplates, _ = fs.Sub(sqliteTemplates, "bobgen-sqlite/templates/models")
)

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
	NoFactory         bool
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
	"enumVal": func(val string) string {
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
