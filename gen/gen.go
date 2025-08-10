package gen

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/orm"
	"github.com/volatiletech/strmangle"
)

var (
	// Tags must be in a format like: json, xml, etc.
	rgxValidTag = regexp.MustCompile(`[a-zA-Z_\.]+`)
	// Column names must be in format column_name or table_name.column_name
	rgxValidTableColumn = regexp.MustCompile(`^[\w]+\.[\w]+$|^[\w]+$`)
)

// State holds the global data needed by most pieces to run
type State[ConstraintExtra any] struct {
	Config              Config[ConstraintExtra]
	Outputs             []*Output
	CustomTemplateFuncs template.FuncMap
}

// Run executes the templates and outputs them to files based on the
// state given.
func Run[T, C, I any](ctx context.Context, s *State[C], driver drivers.Interface[T, C, I], plugins ...Plugin) error {
	if driver.Dialect() == "" {
		return fmt.Errorf("no dialect specified")
	}

	// For StatePlugins
	for _, plugin := range plugins {
		if statePlug, ok := plugin.(StatePlugin[C]); ok {
			err := statePlug.PlugState(s)
			if err != nil {
				return fmt.Errorf("StatePlugin Error [%s]: %w", statePlug.Name(), err)
			}
		}
	}

	dbInfo, err := driver.Assemble(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch table data: %w", err)
	}

	// For DBInfoPlugins
	for _, plugin := range plugins {
		if dbPlug, ok := plugin.(DBInfoPlugin[T, C, I]); ok {
			err := dbPlug.PlugDBInfo(dbInfo)
			if err != nil {
				return fmt.Errorf("StatePlugin Error [%s]: %w", dbPlug.Name(), err)
			}
		}
	}

	if len(dbInfo.Tables) == 0 {
		return errors.New("no tables found in database")
	}

	pkgMap, err := buildPkgMap(s.Outputs)
	if err != nil {
		return fmt.Errorf("getting models pkg details: %w", err)
	}

	// Merge in the user-configured types
	types := driver.Types()
	types.RegisterAll(s.Config.Types)

	switch s.Config.TypeSystem {
	case "", "github.com/aarondl/opt":
		types.SetTypeModifier(drivers.AarondlNull{})
	case "github.com/aarondl/opt/null":
		types.SetTypeModifier(drivers.AarondlNullPointers{})
	case "database/sql":
		types.SetTypeModifier(drivers.DatabaseSqlNull{})
	default:
		panic(fmt.Sprintf("unknown type system %q", s.Config.TypeSystem))
	}

	initInflections(s.Config.Inflections)
	processConstraintConfig(dbInfo.Tables, s.Config.Constraints)
	processTypeReplacements(types, s.Config.Replacements, dbInfo.Tables)
	types.SetOutputImports(pkgMap)

	relationships := buildRelationships(dbInfo.Tables)
	if err := processRelationshipConfig(&s.Config, dbInfo.Tables, relationships); err != nil {
		return fmt.Errorf("processing relationships: %w", err)
	}
	if err := validateRelationships(relationships); err != nil {
		return fmt.Errorf("validating relationships: %w", err)
	}

	// Lets sort the relationships so that we can have a consistent output
	for _, rels := range relationships {
		slices.SortFunc(rels, func(a, b orm.Relationship) int {
			return strings.Compare(a.Name, b.Name)
		})
	}

	if s.Config.Aliases == nil {
		s.Config.Aliases = make(map[string]drivers.TableAlias)
	}
	if err := initAliases(s.Config.Aliases, dbInfo.Tables, relationships); err != nil {
		return fmt.Errorf("initializing aliases: %w\nSee: https://bob.stephenafamo.com/docs/code-generation/configuration#aliases", err)
	}
	if err = s.initTags(); err != nil {
		return fmt.Errorf("unable to initialize struct tags: %w", err)
	}

	data := &TemplateData[T, C, I]{
		Dialect:           driver.Dialect(),
		Tables:            dbInfo.Tables,
		QueryFolders:      dbInfo.QueryFolders,
		Enums:             dbInfo.Enums,
		ExtraInfo:         dbInfo.ExtraInfo,
		Aliases:           s.Config.Aliases,
		Types:             types,
		Relationships:     relationships,
		NoTests:           s.Config.NoTests,
		NoBackReferencing: s.Config.NoBackReferencing,
		StructTagCasing:   s.Config.StructTagCasing,
		TagIgnore:         make(map[string]struct{}),
		Tags:              s.Config.Tags,
		RelationTag:       s.Config.RelationTag,
		OutputPackages:    pkgMap,
		Driver:            dbInfo.Driver,
	}

	for _, v := range s.Config.TagIgnore {
		if !rgxValidTableColumn.MatchString(v) {
			return errors.New("invalid column name %q supplied, only specify column name or table.column, eg: created_at, user.password")
		}
		data.TagIgnore[v] = struct{}{}
	}

	// For TemplateDataPlugins
	for _, plugin := range plugins {
		if tdPlug, ok := plugin.(TemplateDataPlugin[T, C, I]); ok {
			err = tdPlug.PlugTemplateData(data)
			if err != nil {
				return fmt.Errorf("TemplateDataPlugin Error [%s]: %w", tdPlug.Name(), err)
			}
		}
	}

	return generate(s, data)
}

func generate[T, C, I any](s *State[C], data *TemplateData[T, C, I]) error {
	knownKeys := make(map[string]struct{})

	for _, o := range s.Outputs {
		if _, ok := knownKeys[o.Key]; ok {
			return fmt.Errorf("duplicate output key: %q", o.Key)
		}
		knownKeys[o.Key] = struct{}{}

		if err := o.initTemplates(s.CustomTemplateFuncs); err != nil {
			return fmt.Errorf("unable to initialize templates: %w", err)
		}

		// Has a stable output folder
		if o.OutFolder != "" {
			if err := generateSingletonOutput(o, data, s.Config.Generator, s.Config.NoTests); err != nil {
				return fmt.Errorf("singleton template output: %w", err)
			}

			if err := generateTableOutput(o, data, s.Config.Generator, s.Config.NoTests); err != nil {
				return fmt.Errorf("unable to generate output: %w", err)
			}
		}

		if o.queryTemplates != nil && len(o.queryTemplates.Templates()) > 0 {
			// If the output is for queries, we need to iterate over each query folder
			for _, folder := range data.QueryFolders {
				o.PkgName = filepath.Base(folder.Path)
				o.OutFolder = folder.Path
				data.QueryFolder = folder

				if err := generateSingletonOutput(o, data, s.Config.Generator, s.Config.NoTests); err != nil {
					return fmt.Errorf("singleton template output: %w", err)
				}

				if err := generateQueryOutput(o, data, s.Config.Generator, s.Config.NoTests); err != nil {
					return fmt.Errorf("unable to generate output: %w", err)
				}
			}
		}
	}

	return nil
}

// initInflections adds custom inflections to strmangle's ruleset
func initInflections(i Inflections) {
	ruleset := strmangle.GetBoilRuleset()

	for k, v := range i.Plural {
		ruleset.AddPlural(k, v)
	}
	for k, v := range i.PluralExact {
		ruleset.AddPluralExact(k, v, true)
	}

	for k, v := range i.Singular {
		ruleset.AddSingular(k, v)
	}
	for k, v := range i.SingularExact {
		ruleset.AddSingularExact(k, v, true)
	}

	for k, v := range i.Irregular {
		ruleset.AddIrregular(k, v)
	}
}

// initTags removes duplicate tags and validates the format
// of all user tags are simple strings without quotes: [a-zA-Z_\.]+
func (s *State[C]) initTags() error {
	s.Config.Tags = strmangle.RemoveDuplicates(s.Config.Tags)
	for _, v := range s.Config.Tags {
		if !rgxValidTag.MatchString(v) {
			return errors.New("invalid tag format %q supplied, only specify name, eg: xml")
		}
	}

	return nil
}

func buildPkgMap(outputs []*Output) (map[string]string, error) {
	pkgMap := make(map[string]string)

	for _, o := range outputs {
		if o.Disabled {
			continue // skip disabled outputs
		}

		if o.OutFolder == "" {
			// skip outputs with no fixed output folder
			// such as with the "queries" plugin
			continue
		}

		pkg, _, err := language.PackageForFolder(o.OutFolder)
		if err != nil {
			return nil, fmt.Errorf("getting package for folder %q: %w", o.OutFolder, err)
		}
		pkgMap[o.Key] = pkg
	}

	return pkgMap, nil
}
