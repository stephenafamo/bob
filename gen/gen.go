package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/volatiletech/strmangle"
	"golang.org/x/mod/modfile"
)

var (
	// Tags must be in a format like: json, xml, etc.
	rgxValidTag = regexp.MustCompile(`[a-zA-Z_\.]+`)
	// Column names must be in format column_name or table_name.column_name
	rgxValidTableColumn = regexp.MustCompile(`^[\w]+\.[\w]+$|^[\w]+$`)
)

// State holds the global data needed by most pieces to run
type State struct {
	Config              Config
	Outputs             []*Output
	CustomTemplateFuncs template.FuncMap
}

// Run executes the templates and outputs them to files based on the
// state given.
func Run[T any](ctx context.Context, s *State, driver drivers.Interface[T], plugins ...Plugin) error {
	if driver.Dialect() == "" {
		return fmt.Errorf("no dialect specified")
	}

	// For StatePlugins
	for _, plugin := range plugins {
		if statePlug, ok := plugin.(StatePlugin); ok {
			err := statePlug.PlugState(s)
			if err != nil {
				return fmt.Errorf("StatePlugin Error [%s]: %w", statePlug.Name(), err)
			}
		}
	}

	if len(s.Config.Generator) > 0 {
		noEditDisclaimer = []byte(
			fmt.Sprintf(noEditDisclaimerFmt, " by "+s.Config.Generator),
		)
	}

	dbInfo, err := driver.Assemble(ctx)
	if err != nil {
		return fmt.Errorf("unable to fetch table data: %w", err)
	}

	// For DBInfoPlugins
	for _, plugin := range plugins {
		if dbPlug, ok := plugin.(DBInfoPlugin[T]); ok {
			err := dbPlug.PlugDBInfo(dbInfo)
			if err != nil {
				return fmt.Errorf("StatePlugin Error [%s]: %w", dbPlug.Name(), err)
			}
		}
	}

	if len(dbInfo.Tables) == 0 {
		return errors.New("no tables found in database")
	}

	modPkg, version, err := modelsPackage(s.Outputs)
	if err != nil {
		return fmt.Errorf("getting models pkg details: %w", err)
	}

	// Merge in the user-configured types
	types := driver.Types()
	if types == nil {
		types = make(drivers.Types)
	}
	for name, def := range s.Config.Types {
		types[name] = def
	}

	initInflections(s.Config.Inflections)
	processConstraintConfig(dbInfo.Tables, s.Config.Constraints)
	processTypeReplacements(types, s.Config.Replacements, dbInfo.Tables)

	relationships := buildRelationships(dbInfo.Tables)
	if err := processRelationshipConfig(&s.Config, dbInfo.Tables, relationships); err != nil {
		return fmt.Errorf("processing relationships: %w", err)
	}
	if err := validateRelationships(relationships); err != nil {
		return fmt.Errorf("validating relationships: %w", err)
	}

	if s.Config.Aliases == nil {
		s.Config.Aliases = make(map[string]TableAlias)
	}
	initAliases(s.Config.Aliases, dbInfo.Tables, relationships)
	if err = s.initTags(); err != nil {
		return fmt.Errorf("unable to initialize struct tags: %w", err)
	}

	data := &TemplateData[T]{
		Dialect:           driver.Dialect(),
		Tables:            dbInfo.Tables,
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
		ModelsPackage:     modPkg,
	}

	for _, v := range s.Config.TagIgnore {
		if !rgxValidTableColumn.MatchString(v) {
			return errors.New("Invalid column name %q supplied, only specify column name or table.column, eg: created_at, user.password")
		}
		data.TagIgnore[v] = struct{}{}
	}

	// For TemplateDataPlugins
	for _, plugin := range plugins {
		if tdPlug, ok := plugin.(TemplateDataPlugin[T]); ok {
			err = tdPlug.PlugTemplateData(data)
			if err != nil {
				return fmt.Errorf("TemplateDataPlugin Error [%s]: %w", tdPlug.Name(), err)
			}
		}
	}

	return generate(s, data, version)
}

func generate[T any](s *State, data *TemplateData[T], goVersion string) error {
	knownKeys := make(map[string]struct{})

	for _, o := range s.Outputs {
		if len(o.Templates) == 0 {
			continue
		}

		if _, ok := knownKeys[o.Key]; ok {
			return fmt.Errorf("Duplicate output key: %q", o.Key)
		}
		knownKeys[o.Key] = struct{}{}

		// set the package name for this output
		data.PkgName = o.PkgName

		templates, err := o.initTemplates(s.CustomTemplateFuncs, s.Config.NoTests)
		if err != nil {
			return fmt.Errorf("unable to initialize templates: %w", err)
		}

		tplCount := 0
		if o.templates != nil {
			tplCount += len(o.templates.Templates())
		}
		if o.testTemplates != nil {
			tplCount += len(o.testTemplates.Templates())
		}
		if tplCount == 0 {
			continue
		}

		err = o.initOutFolders(templates, s.Config.Wipe)
		if err != nil {
			return fmt.Errorf("unable to initialize the output folders: %w", err)
		}

		if err := generateSingletonOutput(o, data, goVersion); err != nil {
			return fmt.Errorf("singleton template output: %w", err)
		}

		if !s.Config.NoTests {
			if err := generateSingletonTestOutput(o, data, goVersion); err != nil {
				return fmt.Errorf("unable to generate singleton test template output: %w", err)
			}
		}

		var regularDirExtMap, testDirExtMap dirExtMap
		regularDirExtMap = groupTemplates(o.templates)
		if !s.Config.NoTests {
			testDirExtMap = groupTemplates(o.testTemplates)
		}

		for _, table := range data.Tables {
			data.Table = table

			// Generate the regular templates
			if err := generateOutput(o, regularDirExtMap, data, goVersion); err != nil {
				return fmt.Errorf("unable to generate output: %w", err)
			}

			// Generate the test templates
			if !s.Config.NoTests {
				if err := generateTestOutput(o, testDirExtMap, data, goVersion); err != nil {
					return fmt.Errorf("unable to generate test output: %w", err)
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
func (s *State) initTags() error {
	s.Config.Tags = strmangle.RemoveDuplicates(s.Config.Tags)
	for _, v := range s.Config.Tags {
		if !rgxValidTag.MatchString(v) {
			return errors.New("Invalid tag format %q supplied, only specify name, eg: xml")
		}
	}

	return nil
}

// Returns the pkg name, and the go version
func modelsPackage(outputs []*Output) (string, string, error) {
	var modelsFolder string
	for _, o := range outputs {
		if o.Key == "models" {
			modelsFolder = o.OutFolder
		}
	}

	if modelsFolder == "" {
		return "", "", nil
	}

	modRoot, modFile, err := goModInfo(modelsFolder)
	if err != nil {
		return "", "", fmt.Errorf("getting mod details: %w", err)
	}

	fullPath := modelsFolder
	if !filepath.IsAbs(modelsFolder) {
		wd, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("could not get working directory: %w", err)
		}

		fullPath = filepath.Join(wd, modelsFolder)
	}

	relPath := strings.TrimPrefix(fullPath, modRoot)
	return path.Join(modFile.Module.Mod.Path, filepath.ToSlash(relPath)), modFile.Go.Version, nil
}

// goModInfo returns the main module's root directory
// and the parsed contents of the go.mod file.
func goModInfo(path string) (string, *modfile.File, error) {
	goModPath, err := findGoMod(path)
	if err != nil {
		return "", nil, fmt.Errorf("cannot find main module: %w", err)
	}

	if goModPath == os.DevNull {
		return "", nil, fmt.Errorf("destination is not in a go module")
	}

	data, err := os.ReadFile(goModPath)
	if err != nil {
		return "", nil, fmt.Errorf("cannot read main go.mod file: %w", err)
	}

	modf, err := modfile.Parse(goModPath, data, nil)
	if err != nil {
		return "", nil, fmt.Errorf("could not parse go.mod: %w", err)
	}

	return filepath.Dir(goModPath), modf, nil
}

func findGoMod(path string) (string, error) {
	var outData, errData bytes.Buffer

	err := os.MkdirAll(path, 0o755)
	if err != nil {
		return "", fmt.Errorf("could not create destination folder %q: %w", path, err)
	}

	c := exec.Command("go", "env", "GOMOD")
	c.Stdout = &outData
	c.Stderr = &errData
	c.Dir = path
	err = c.Run()
	if err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) && errData.Len() > 0 {
			return "", errors.New(strings.TrimSpace(errData.String()))
		}

		return "", fmt.Errorf("cannot run go env GOMOD: %w", err)
	}

	out := strings.TrimSpace(outData.String())
	if out == "" {
		return "", errors.New("no go.mod file found in any parent directory")
	}

	return out, nil
}
