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
	"columnGetter": func(t []drivers.Table, table string, a TableAlias, column string) string {
		for _, t := range t {
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
	},
	"getColumn": func(t []drivers.Table, table string, a TableAlias, column string) drivers.Column {
		for _, t := range t {
			if t.Name != table {
				continue
			}

			return t.GetColumn(column)
		}

		panic("unknown table " + table)
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
}
