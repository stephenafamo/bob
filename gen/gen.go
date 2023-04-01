package gen

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/orm"
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

	var templates []lazyTemplate

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

	modPkg, err := modelsPackage(s.Outputs)
	if err != nil {
		return fmt.Errorf("getting models pkg details: %w", err)
	}

	initInflections(s.Config.Inflections)
	processTypeReplacements(s.Config.Replacements, dbInfo.Tables)
	processRelationshipConfig(s.Config.Relationships, dbInfo.Tables)
	initAliases(&s.Config.Aliases, dbInfo.Tables)
	err = s.initTags()
	if err != nil {
		return fmt.Errorf("unable to initialize struct tags: %w", err)
	}

	data := &TemplateData[T]{
		Dialect:           driver.Dialect(),
		Tables:            dbInfo.Tables,
		Enums:             dbInfo.Enums,
		ExtraInfo:         dbInfo.ExtraInfo,
		Aliases:           s.Config.Aliases,
		NoTests:           s.Config.NoTests,
		NoBackReferencing: s.Config.NoBackReferencing,
		StructTagCasing:   s.Config.StructTagCasing,
		TagIgnore:         make(map[string]struct{}),
		Tags:              s.Config.Tags,
		RelationTag:       s.Config.RelationTag,
		ModelsPackage:     modPkg,
		CanBulkInsert:     driver.Capabilities().BulkInsert,
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

	knownKeys := make(map[string]struct{})
	for _, o := range s.Outputs {
		if _, ok := knownKeys[o.Key]; ok {
			return fmt.Errorf("Duplicate output key: %q", o.Key)
		}
		knownKeys[o.Key] = struct{}{}

		// set the package name for this output
		data.PkgName = o.PkgName

		templates, err = o.initTemplates(s.CustomTemplateFuncs, s.Config.NoTests)
		if err != nil {
			return fmt.Errorf("unable to initialize templates: %w", err)
		}

		err = o.initOutFolders(templates, s.Config.Wipe)
		if err != nil {
			return fmt.Errorf("unable to initialize the output folders: %w", err)
		}

		padding := outputPadding(o.templates.Templates(), dbInfo.Tables)

		if err := generateSingletonOutput(o, data, padding); err != nil {
			return fmt.Errorf("singleton template output: %w", err)
		}

		if !s.Config.NoTests {
			if err := generateSingletonTestOutput(o, data, padding); err != nil {
				return fmt.Errorf("unable to generate singleton test template output: %w", err)
			}
		}

		var regularDirExtMap, testDirExtMap dirExtMap
		regularDirExtMap = groupTemplates(o.templates)
		if !s.Config.NoTests {
			testDirExtMap = groupTemplates(o.testTemplates)
		}

		for _, table := range dbInfo.Tables {
			data.Table = table

			// Generate the regular templates
			if err := generateOutput(o, regularDirExtMap, data, padding); err != nil {
				return fmt.Errorf("unable to generate output: %w", err)
			}

			// Generate the test templates
			if !s.Config.NoTests {
				if err := generateTestOutput(o, testDirExtMap, data, padding); err != nil {
					return fmt.Errorf("unable to generate test output: %w", err)
				}
			}
		}
	}

	return nil
}

// initTemplates loads all template folders into the state object.
//
// If TemplateDirs is set it uses those, else it pulls from assets.
// Then it allows drivers to override, followed by replacements. Any
// user functions passed in by library users will be merged into the
// template.FuncMap.
//
// Because there's the chance for windows paths to jumped in
// all paths are converted to the native OS's slash style.
//
// Later, in order to properly look up imports the paths will
// be forced back to linux style paths.
func (o *Output) initTemplates(funcs template.FuncMap, notests bool) ([]lazyTemplate, error) {
	var err error

	templates := make(map[string]templateLoader)
	if len(o.Templates) == 0 {
		return nil, errors.New("No templates defined")
	}

	for _, tempFS := range o.Templates {
		if tempFS == nil {
			continue
		}
		err := fs.WalkDir(tempFS, ".", func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return fmt.Errorf("in walk err: %w", err)
			}

			if entry.IsDir() {
				return nil
			}

			name := entry.Name()
			if filepath.Ext(name) == ".tpl" {
				templates[normalizeSlashes(path)] = assetLoader{fs: tempFS, name: path}
			}

			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("after walk err: %w", err)
		}
	}

	// For stability, sort keys to traverse the map and turn it into a slice
	keys := make([]string, 0, len(templates))
	for k := range templates {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	lazyTemplates := make([]lazyTemplate, 0, len(templates))
	for _, k := range keys {
		lazyTemplates = append(lazyTemplates, lazyTemplate{
			Name:   k,
			Loader: templates[k],
		})
	}

	o.templates, err = loadTemplates(lazyTemplates, false, funcs)
	if err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	if !notests {
		o.testTemplates, err = loadTemplates(lazyTemplates, true, funcs)
		if err != nil {
			return nil, fmt.Errorf("loading test templates: %w", err)
		}
	}

	return lazyTemplates, nil
}

type dirExtMap map[string]map[string][]string

// groupTemplates takes templates and groups them according to their output directory
// and file extension.
func groupTemplates(templates *templateList) dirExtMap {
	tplNames := templates.Templates()
	dirs := make(map[string]map[string][]string)
	for _, tplName := range tplNames {
		normalized, isSingleton, _, _ := outputFilenameParts(tplName)
		if isSingleton {
			continue
		}

		dir := filepath.Dir(normalized)
		if dir == "." {
			dir = ""
		}

		extensions, ok := dirs[dir]
		if !ok {
			extensions = make(map[string][]string)
			dirs[dir] = extensions
		}

		ext := getLongExt(tplName)
		ext = strings.TrimSuffix(ext, ".tpl")
		slice := extensions[ext]
		extensions[ext] = append(slice, tplName)
	}

	return dirs
}

func outputPadding(tpls []string, tables []drivers.Table) int {
	longest := 0
	for _, tplName := range tpls {
		normalized, isSingleton, _, _ := outputFilenameParts(tplName)
		if isSingleton && len(normalized) > longest {
			longest = len(normalized)
		}
	}

	for _, t := range tables {
		if len(t.Name)+6 > longest { // padd for any extra suffix like _model
			longest = len(t.Name) + 6
		}
	}

	return longest
}

// processTypeReplacements checks the config for type replacements
// and performs them.
func processTypeReplacements(replacements []Replace, tables []drivers.Table) {
	for _, r := range replacements {
		for i := range tables {
			t := tables[i]

			if !shouldReplaceInTable(t, r) {
				continue
			}

			for j := range t.Columns {
				c := t.Columns[j]
				if matchColumn(c, r.Match) {
					t.Columns[j] = columnMerge(c, r.Replace)
				}
			}
		}
	}
}

// matchColumn checks if a column 'c' matches specifiers in 'm'.
// Anything defined in m is checked against a's values, the
// match is a done using logical and (all specifiers must match).
// Bool fields are only checked if a string type field matched first
// and if a string field matched they are always checked (must be defined).
//
// Doesn't care about Unique columns since those can vary independent of type.
func matchColumn(c, m drivers.Column) bool {
	matchedSomething := false

	// return true if we matched, or we don't have to match
	// if we actually matched against something, then additionally set
	// matchedSomething so we can check boolean values too.
	matches := func(matcher, value string) bool {
		if len(matcher) != 0 && matcher != value {
			return false
		}
		matchedSomething = true
		return true
	}

	if !matches(m.Name, c.Name) {
		return false
	}
	if !matches(m.Type, c.Type) {
		return false
	}
	if !matches(m.DBType, c.DBType) {
		return false
	}

	if !matches(m.DomainName, c.DomainName) {
		return false
	}

	if !matchedSomething {
		return false
	}

	if m.Generated != c.Generated {
		return false
	}
	if m.Nullable != c.Nullable {
		return false
	}

	return true
}

// columnMerge merges values from src into dst. Bools are copied regardless
// strings are copied if they have values. Name is excluded because it doesn't make
// sense to non-programatically replace a name.
func columnMerge(dst, src drivers.Column) drivers.Column {
	ret := dst
	if len(src.Type) != 0 {
		ret.Type = src.Type
		ret.Imports = src.Imports
	}
	if len(src.Imports) != 0 {
		ret.Imports = src.Imports
	}
	if len(src.DBType) != 0 {
		ret.DBType = src.DBType
	}

	return ret
}

// shouldReplaceInTable checks if tables were specified in types.match in the config.
// If tables were set, it checks if the given table is among the specified tables.
func shouldReplaceInTable(t drivers.Table, r Replace) bool {
	if len(r.Tables) == 0 {
		return true
	}

	for _, replaceInTable := range r.Tables {
		if replaceInTable == t.Key {
			return true
		}
	}

	return false
}

// processRelationshipConfig checks any user included relationships and adds them to the tables
func processRelationshipConfig(r relationships, tables []drivers.Table) {
	if len(tables) == 0 {
		return
	}

	for i, t := range tables {
		rels, ok := r[t.Key]
		if !ok {
			continue
		}

		tables[i].Relationships = mergeRelationships(tables[i].Relationships, rels)
	}
}

func mergeRelationships(srcs, extras []orm.Relationship) []orm.Relationship {
Outer:
	for _, extra := range extras {
		for i, src := range srcs {
			if src.Name == extra.Name {
				srcs[i] = mergeRelationship(src, extra)
				continue Outer
			}
		}

		// No previous relationship was found, add it as-is
		srcs = append(srcs, extra)
	}

	final := make([]orm.Relationship, 0, len(srcs))
	for _, rel := range srcs {
		if rel.Ignored || len(rel.Sides) < 1 {
			continue
		}

		final = append(final, rel)
	}

	return final
}

func mergeRelationship(src, extra orm.Relationship) orm.Relationship {
	src.Ignored = extra.Ignored
	if len(extra.Sides) > 0 {
		src.Sides = extra.Sides
	}

	return src
}

// initOutFolders creates the folders that will hold the generated output.
func (o *Output) initOutFolders(lazyTemplates []lazyTemplate, wipe bool) error {
	if wipe {
		if err := os.RemoveAll(o.OutFolder); err != nil {
			return err
		}
	}

	newDirs := make(map[string]struct{})
	for _, t := range lazyTemplates {
		// js/00_struct.js.tpl
		// js/singleton/00_struct.js.tpl
		// we want the js part only
		fragments := strings.Split(t.Name, string(os.PathSeparator))

		// Throw away the filename
		fragments = fragments[0 : len(fragments)-1]
		if len(fragments) != 0 && fragments[len(fragments)-1] == "singleton" {
			fragments = fragments[:len(fragments)-1]
		}

		if len(fragments) == 0 {
			continue
		}

		newDirs[strings.Join(fragments, string(os.PathSeparator))] = struct{}{}
	}

	if err := os.MkdirAll(o.OutFolder, os.ModePerm); err != nil {
		return err
	}

	for d := range newDirs {
		if err := os.MkdirAll(filepath.Join(o.OutFolder, d), os.ModePerm); err != nil {
			return err
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

func initAliases(a *Aliases, tables []drivers.Table) {
	FillAliases(a, tables)
}

// normalizeSlashes takes a path that was made on linux or windows and converts it
// to a native path.
func normalizeSlashes(path string) string {
	path = strings.ReplaceAll(path, `/`, string(os.PathSeparator))
	path = strings.ReplaceAll(path, `\`, string(os.PathSeparator))
	return path
}

func modelsPackage(outputs []*Output) (string, error) {
	var modelsFolder string
	for _, o := range outputs {
		if o.Key == "models" {
			modelsFolder = o.OutFolder
		}
	}

	if modelsFolder == "" {
		return "", nil
	}

	modRoot, modFile, err := goModInfo(modelsFolder)
	if err != nil {
		return "", fmt.Errorf("getting mod details: %w", err)
	}

	fullPath := modelsFolder
	if !filepath.IsAbs(modelsFolder) {
		wd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get working directory: %w", err)
		}

		fullPath = filepath.Join(wd, modelsFolder)
	}

	relPath := strings.TrimPrefix(fullPath, modRoot)
	return path.Join(modFile.Module.Mod.Path, relPath), nil
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
