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
	"github.com/stephenafamo/bob/gen/importers"
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
	typesReplacer           = strings.NewReplacer(
		" ", "_",
		".", "_",
		",", "_",
		"*", "_",
		"[", "_",
		"]", "_",
	)
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

type TemplateData[T, C, I any] struct {
	Dialect  string
	Importer Importer

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
	AddSoftDeletes    bool
	AddEnumTypes      bool
	EnumNullPrefix    string
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
	ExtraInfo     T
	ModelsPackage string

	// DriverName is the module name of the underlying `database/sql` driver
	DriverName string
}

func (t *TemplateData[T, C, I]) ResetImports() {
	t.Importer = make(Importer)
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
	"getType":            getType,
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
	return typesReplacer.Replace(val)
}

// Gets the type for a db column. Used if you have types defined inside the
// models dir, which needs the models prefix in the factory files.
func getType(columnType string, typedef drivers.Type) string {
	prefix := ""
	if typedef.InGeneratedPackage {
		prefix = "models."
	}

	if typedef.AliasOf != "" {
		return prefix + typedef.AliasOf
	}

	return prefix + columnType
}
