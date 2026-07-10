package gen

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"text/template"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
)

type Output struct {
	// If true, new files are not generated, but existing files are deleted
	Disabled bool

	// The key has to be unique in a gen.State
	// it also makes it possible to target modifing a specific output
	Key string

	PkgName                 string
	OutFolder               string
	Templates               []fs.FS
	SeparatePackageForTests bool

	singletonTemplates *template.Template
	tableTemplates     *template.Template
	queryTemplates     *template.Template

	// Scratch buffers used as staging area for preparing parsed template data
	templateByteBuffer       *bytes.Buffer
	templateHeaderByteBuffer *bytes.Buffer
}

func (o *Output) numTemplates() int {
	return 0 +
		len(o.singletonTemplates.Templates()) +
		len(o.tableTemplates.Templates()) +
		len(o.queryTemplates.Templates())
}

// initOutFolders creates the folders that will hold the generated output.
func (o *Output) initOutFolders() error {
	files, err := os.ReadDir(o.OutFolder)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("unable to read output folder: %w", err)
	}

	for _, d := range files {
		if d.IsDir() {
			continue
		}

		name := d.Name()
		name = name[:len(name)-len(filepath.Ext(name))]

		if !strings.HasSuffix(name, ".bob") && !strings.HasSuffix(name, ".bob_test") {
			continue
		}

		if err := os.Remove(filepath.Join(o.OutFolder, d.Name())); err != nil {
			return fmt.Errorf("unable to remove old file: %w", err)
		}
	}

	// Do not create the output folder if it is disabled
	// However, we do this after cleaning up any old `.bob` files
	if o.Disabled {
		fmt.Fprintf(os.Stderr, "%-20s %s\n", "== DISABLED ==", o.OutFolder)
		return nil
	}

	if err := os.MkdirAll(o.OutFolder, os.ModePerm); err != nil {
		return fmt.Errorf("unable to create output folder %q: %w", o.OutFolder, err)
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
func (o *Output) initTemplates(funcs template.FuncMap) error {
	if len(o.Templates) == 0 {
		return nil
	}

	o.singletonTemplates = template.New("")
	o.tableTemplates = template.New("")
	o.queryTemplates = template.New("")

	if err := addTemplates(o.singletonTemplates, o.Templates, funcs, ".", true); err != nil {
		return fmt.Errorf("failed to add singleton templates: %w", err)
	}

	if err := addTemplates(o.tableTemplates, o.Templates, funcs, "table", false); err != nil {
		return fmt.Errorf("failed to add table templates: %w", err)
	}

	if err := addTemplates(o.queryTemplates, o.Templates, funcs, "query", false); err != nil {
		return fmt.Errorf("failed to add query templates: %w", err)
	}

	return nil
}

func addTemplates(tpl *template.Template, tempFSs []fs.FS, funcs template.FuncMap, dir string, singletons bool) error {
	type details struct {
		fs       fs.FS
		fullPath string
	}
	all := make(map[string]details)

	for _, tempFS := range tempFSs {
		if tempFS == nil {
			continue
		}

		if dir != "" {
			tempFS, _ = fs.Sub(tempFS, dir)
			if tempFS == nil {
				continue
			}
		}

		entries, err := fs.ReadDir(tempFS, ".")
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return fmt.Errorf("failed to read dir %q: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			ext := filepath.Ext(name)
			if ext != ".tpl" {
				continue
			}

			// Remove the ".tpl" suffix
			name = strings.TrimSuffix(name, ext)
			// The actual extension
			ext = filepath.Ext(name)

			if singletons {
				fNameWithoutExts := filepath.Base(name[:len(name)-len(ext)])
				if !strings.HasSuffix(fNameWithoutExts, ".bob") &&
					!strings.HasSuffix(fNameWithoutExts, ".bob_test") {
					panic(fmt.Sprintf("singleton file name must end with .bob.tpl or .bob_test.tpl: %s", name))
				}
			}

			all[entry.Name()] = details{
				fs:       tempFS,
				fullPath: filepath.Join(dir, name),
			}
		}
	}

	paths := slices.Collect(maps.Keys(all))
	slices.Sort(paths)

	for _, path := range paths {
		details := all[path]
		content, err := fs.ReadFile(details.fs, path)
		if err != nil {
			return fmt.Errorf("failed to read template: %s: %w", details.fullPath, err)
		}

		err = loadTemplate(tpl, funcs, path, string(content))
		if err != nil {
			return fmt.Errorf("failed to load template: %s: %w", details.fullPath, err)
		}
	}

	return nil
}

type executeTemplateData[T, C, I any] struct {
	output *Output
	data   *TemplateData[T, C, I]

	templates    *template.Template
	extTemplates extMap
	langs        language.Languages
}

func generateTableOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], generator string, noTests bool) error {
	if o.tableTemplates == nil || len(o.tableTemplates.Templates()) == 0 {
		return nil
	}

	dirExtMap := groupTemplatesByExtension(o.tableTemplates)
	langs := language.Languages{
		GeneratorName:           generator,
		SeparatePackageForTests: o.SeparatePackageForTests,
	}
	for _, table := range data.Tables {
		data.Table = table

		// Generate the regular templates
		if err := generateOutput(o, dirExtMap, o.tableTemplates, data, langs, noTests); err != nil {
			return fmt.Errorf("unable to generate output: %w", err)
		}
	}

	return nil
}

func cleanGeneratedSubdirectories(root string) error {
	var directories []string
	err := filepath.WalkDir(root, func(path string, item fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if item.IsDir() {
			if path != root {
				directories = append(directories, path)
			}
			return nil
		}
		name := item.Name()
		name = name[:len(name)-len(filepath.Ext(name))]
		if strings.HasSuffix(name, ".bob") || strings.HasSuffix(name, ".bob_test") {
			return os.Remove(path)
		}
		return nil
	})
	if err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err := os.Remove(directories[i]); err != nil &&
			!errors.Is(err, fs.ErrNotExist) &&
			!errors.Is(err, syscall.ENOTEMPTY) {
			return err
		}
	}
	return nil
}

func generateSplitModelOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], generator string, noTests bool) error {
	if data.ModelSplit == nil || !data.ModelSplit.Enabled {
		return generateTableOutput(o, data, generator, noTests)
	}

	originalTables := data.Tables
	originalPkgName := o.PkgName
	originalOutFolder := o.OutFolder
	originalGeneration := data.ModelSplit.Generation
	originalComponent := data.ModelSplit.CurrentComponent
	defer func() {
		data.Tables = originalTables
		o.PkgName = originalPkgName
		o.OutFolder = originalOutFolder
		data.ModelSplit.Generation = originalGeneration
		data.ModelSplit.CurrentComponent = originalComponent
	}()

	if data.ModelSplit.GeneratesFacade() {
		if err := o.initOutFolders(); err != nil {
			return fmt.Errorf("unable to initialize root model output folder: %w", err)
		}
		if err := os.RemoveAll(filepath.Join(o.OutFolder, filepath.FromSlash(data.ModelSplit.InternalDir))); err != nil {
			return fmt.Errorf("removing old split model output: %w", err)
		}

		data.Tables = originalTables
		data.ModelSplit.Generation = modelSplitGenerationFacade
		data.ModelSplit.CurrentComponent = nil
		if err := generateSingletonOutput(o, data, generator, noTests); err != nil {
			return fmt.Errorf("root facade singleton output: %w", err)
		}
	} else {
		if err := o.initOutFolders(); err != nil {
			return fmt.Errorf("unable to initialize root model output folder: %w", err)
		}
		if err := cleanGeneratedSubdirectories(o.OutFolder); err != nil {
			return fmt.Errorf("cleaning old table-package model output: %w", err)
		}
	}

	for _, component := range data.ModelSplit.Components {
		componentOutput := *o
		componentOutput.PkgName = component.Package
		componentOutput.OutFolder = component.OutFolder
		data.Tables = filterTablesForComponent(originalTables, component)
		data.ModelSplit.Generation = modelSplitGenerationComponent
		data.ModelSplit.CurrentComponent = component

		if err := generateSingletonOutput(&componentOutput, data, generator, noTests); err != nil {
			return fmt.Errorf("component %s singleton output: %w", component.ID, err)
		}
		if err := generateTableOutput(&componentOutput, data, generator, noTests); err != nil {
			return fmt.Errorf("component %s table output: %w", component.ID, err)
		}
	}

	return nil
}

func generateSplitFactoryOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], generator string, noTests bool) error {
	if data.ModelSplit == nil || !data.ModelSplit.Enabled {
		return generateTableOutput(o, data, generator, noTests)
	}

	originalTables := data.Tables
	originalPkgName := o.PkgName
	originalOutFolder := o.OutFolder
	originalSplit := data.ModelSplit
	originalModelsPackage := data.OutputPackages["models"]
	defer func() {
		data.Tables = originalTables
		o.PkgName = originalPkgName
		o.OutFolder = originalOutFolder
		data.ModelSplit = originalSplit
		data.OutputPackages["models"] = originalModelsPackage
	}()

	factoryModelsFolder := filepath.Join(filepath.Dir(originalSplit.RootOutFolder), "internal", "factorymodels")
	factoryModelsPackage := path.Join(path.Dir(originalSplit.RootPackagePath), "internal", "factorymodels")
	if err := generateFactoryModelsFacade(factoryModelsFolder, originalSplit, originalTables, data); err != nil {
		return fmt.Errorf("generating factory models facade: %w", err)
	}
	data.OutputPackages["models"] = factoryModelsPackage

	factorySplit := modelSplitForOutput(originalSplit, o.OutFolder, data.OutputPackages[o.Key])
	data.ModelSplit = factorySplit

	if err := o.initOutFolders(); err != nil {
		return fmt.Errorf("unable to initialize root factory output folder: %w", err)
	}
	if factorySplit.Mode == modelPackageSplitModeTablePackages {
		if err := cleanGeneratedSubdirectories(o.OutFolder); err != nil {
			return fmt.Errorf("cleaning old schema/table factory output: %w", err)
		}
	} else if err := os.RemoveAll(filepath.Join(o.OutFolder, filepath.FromSlash(factorySplit.InternalDir))); err != nil {
		return fmt.Errorf("removing old split factory output: %w", err)
	}

	data.Tables = originalTables
	data.ModelSplit.Generation = modelSplitGenerationFacade
	data.ModelSplit.CurrentComponent = nil
	if err := generateSingletonOutput(o, data, generator, noTests); err != nil {
		return fmt.Errorf("root facade singleton output: %w", err)
	}

	for _, component := range data.ModelSplit.Components {
		componentOutput := *o
		componentOutput.PkgName = component.Package
		componentOutput.OutFolder = component.OutFolder
		data.Tables = filterTablesForComponent(originalTables, component)
		data.ModelSplit.Generation = modelSplitGenerationComponent
		data.ModelSplit.CurrentComponent = component

		if err := generateSingletonOutput(&componentOutput, data, generator, noTests); err != nil {
			return fmt.Errorf("component %s singleton output: %w", component.ID, err)
		}
		if err := generateTableOutput(&componentOutput, data, generator, noTests); err != nil {
			return fmt.Errorf("component %s table output: %w", component.ID, err)
		}
	}

	return nil
}

func generateFactoryModelsFacade[T, C, I any](
	outFolder string,
	modelSplit *ModelSplitData,
	tables drivers.Tables[C, I],
	data *TemplateData[T, C, I],
) error {
	if err := os.RemoveAll(outFolder); err != nil {
		return err
	}
	if err := os.MkdirAll(outFolder, os.ModePerm); err != nil {
		return err
	}

	var source strings.Builder
	source.WriteString("// Code generated by BobGen. DO NOT EDIT.\n\npackage factorymodels\n\nimport (\n")
	for _, component := range modelSplit.Components {
		fmt.Fprintf(&source, "	%s %q\n", component.ImportAlias, component.PackagePath)
	}
	source.WriteString(")\n\n")
	for _, table := range tables {
		component := modelSplit.TableComponents[table.Key]
		if component == nil {
			continue
		}
		alias := data.Aliases.Table(table.Key)
		fmt.Fprintf(&source, "type %s = %s.%s\n", alias.UpSingular, component.ImportAlias, alias.UpSingular)
		fmt.Fprintf(&source, "type %sSlice = %s.%sSlice\n", alias.UpSingular, component.ImportAlias, alias.UpSingular)
		if table.Constraints.Primary != nil || len(data.Relationships.Get(table.Key)) > 0 {
			fmt.Fprintf(&source, "type %sSetter = %s.%sSetter\n", alias.UpSingular, component.ImportAlias, alias.UpSingular)
		}
		fmt.Fprintf(&source, "var %s = %s.%s\n\n", alias.UpPlural, component.ImportAlias, alias.UpPlural)
	}

	formatted, err := format.Source([]byte(source.String()))
	if err != nil {
		return fmt.Errorf("formatting facade: %w", err)
	}
	return os.WriteFile(filepath.Join(outFolder, "bob_factory_models.bob.go"), formatted, 0o644)
}

func generateQueryOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], generator string, noTests bool) error {
	if o.queryTemplates == nil || len(o.queryTemplates.Templates()) == 0 {
		return nil
	}

	dirExtMap := groupTemplatesByExtension(o.queryTemplates)
	langs := language.Languages{
		GeneratorName:           generator,
		SeparatePackageForTests: o.SeparatePackageForTests,
	}
	for _, file := range data.QueryFolder.Files {
		data.QueryFile = file

		// We do this so that the name of the file is correct
		data.Table = drivers.Table[C, I]{
			Name: file.BaseName(),
		}

		// Generate the regular templates
		if err := generateOutput(o, dirExtMap, o.queryTemplates, data, langs, noTests); err != nil {
			return fmt.Errorf("unable to generate output: %w", err)
		}
	}

	return nil
}

// generateOutput builds the file output and sends it to outHandler for saving
func generateOutput[T, C, I any](o *Output, dirExts extMap, tpl *template.Template, data *TemplateData[T, C, I], langs language.Languages, noTests bool) error {
	if o.Disabled {
		return nil // skip disabled outputs
	}

	// assign reusable scratch buffers to provided Output
	o.templateByteBuffer = &bytes.Buffer{}
	o.templateHeaderByteBuffer = &bytes.Buffer{}

	if err := executeTemplates(executeTemplateData[T, C, I]{
		output:       o,
		data:         data,
		templates:    tpl,
		extTemplates: dirExts,
		langs:        langs,
	}, false); err != nil {
		return fmt.Errorf("execute templates: %w", err)
	}

	if noTests {
		return nil
	}

	if err := executeTemplates(executeTemplateData[T, C, I]{
		output:       o,
		data:         data,
		templates:    tpl,
		extTemplates: dirExts,
		langs:        langs,
	}, true); err != nil {
		return fmt.Errorf("execute test templates: %w", err)
	}

	return nil
}

// generateSingletonOutput processes the templates that should only be run
// one time.
func generateSingletonOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], generator string, noTests bool) error {
	// set the package name for this output
	data.PkgName = o.PkgName

	if err := o.initOutFolders(); err != nil {
		return fmt.Errorf("unable to initialize the output folders: %w", err)
	}

	if o.Disabled {
		return nil // skip disabled outputs
	}

	if o.numTemplates() == 0 {
		return fmt.Errorf("no templates found for output %q", o.Key)
	}

	// assign reusable scratch buffers to provided Output
	o.templateByteBuffer = &bytes.Buffer{}
	o.templateHeaderByteBuffer = &bytes.Buffer{}

	langs := language.Languages{
		GeneratorName:           generator,
		SeparatePackageForTests: o.SeparatePackageForTests,
	}

	if err := executeSingletonTemplates(executeTemplateData[T, C, I]{
		output:    o,
		data:      data,
		templates: o.singletonTemplates,
		langs:     langs,
	}, false); err != nil {
		return fmt.Errorf("execute singleton templates: %w", err)
	}

	if noTests {
		return nil
	}

	if err := executeSingletonTemplates(executeTemplateData[T, C, I]{
		output:    o,
		data:      data,
		templates: o.singletonTemplates,
		langs:     langs,
	}, true); err != nil {
		return fmt.Errorf("execute singleton test templates: %w", err)
	}

	return nil
}

func executeTemplates[T, C, I any](e executeTemplateData[T, C, I], tests bool) error {
	for ext, tplNames := range e.extTemplates {
		headerOut := e.output.templateHeaderByteBuffer
		headerOut.Reset()
		out := e.output.templateByteBuffer
		out.Reset()

		prevLen := out.Len()

		lang := e.langs.GetOutputLanguage(ext)
		e.data.Language = lang
		e.data.Importer = lang.Importer()
		e.data.CurrentPackage = e.data.OutputPackages[e.output.Key]
		if e.data.ModelSplit != nil &&
			e.data.ModelSplit.Enabled &&
			e.data.ModelSplit.Generation == modelSplitGenerationComponent &&
			e.data.ModelSplit.CurrentComponent != nil {
			e.data.CurrentPackage = e.data.ModelSplit.CurrentComponent.PackagePath
		}

		matchingTemplates := 0
		for _, tplName := range tplNames {
			if !strings.HasSuffix(tplName, ".tpl") {
				continue
			}

			if tests != lang.IsTest(tplName) {
				continue
			}
			matchingTemplates++

			if err := executeTemplate(out, e.templates, tplName, e.data); err != nil {
				return err
			}
		}

		if matchingTemplates == 0 {
			continue
		}

		fName := lang.OutputFileName(e.data.Table.Schema, e.data.Table.Name, tests)

		// Skip writing the file if the content is empty
		if out.Len()-prevLen < 1 {
			fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
				"==  SKIPPED ==", e.output.OutFolder, fName)
			continue
		}

		fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
			fmt.Sprintf("%8d bytes", out.Len()-prevLen),
			e.output.OutFolder, fName)

		path := filepath.Join(e.output.OutFolder, fName)

		// MAKE SURE TO CLOSE THE FILE ON EVERY EXIT PATH!!!!!!
		dest, err := os.Create(path) // Ensure the file exists
		if err != nil {
			return fmt.Errorf("creating output file %s: %w", path, err)
		}

		// MAKE SURE TO CLOSE THE FILE ON EVERY EXIT PATH!!!!!!
		if err := lang.Write(e.data.Importer, e.output.PkgName, e.output.OutFolder, out, tests, dest); err != nil {
			dest.Close()
			return fmt.Errorf("writing file: %w", err)
		}

		dest.Close()
	}

	return nil
}

func executeSingletonTemplates[T, C, I any](e executeTemplateData[T, C, I], tests bool) error {
	headerOut := e.output.templateHeaderByteBuffer
	out := e.output.templateByteBuffer
	for _, tpl := range e.templates.Templates() {
		if !strings.HasSuffix(tpl.Name(), ".tpl") {
			continue
		}

		fileName := strings.TrimSuffix(tpl.Name(), ".tpl")
		lang := e.langs.GetOutputLanguage(filepath.Ext(fileName))
		if tests != lang.IsTest(tpl.Name()) {
			continue
		}

		headerOut.Reset()
		out.Reset()
		prevLen := out.Len()

		e.data.Language = lang
		e.data.Importer = lang.Importer()
		e.data.CurrentPackage = e.data.OutputPackages[e.output.Key]
		if e.data.ModelSplit != nil &&
			e.data.ModelSplit.Enabled &&
			e.data.ModelSplit.Generation == modelSplitGenerationComponent &&
			e.data.ModelSplit.CurrentComponent != nil {
			e.data.CurrentPackage = e.data.ModelSplit.CurrentComponent.PackagePath
		}
		if err := executeTemplate(out, e.templates, tpl.Name(), e.data); err != nil {
			return err
		}

		// Skip writing the file if the content is empty
		if out.Len()-prevLen < 1 {
			fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
				"==  SKIPPED ==", e.output.OutFolder, fileName)
			continue
		}

		fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
			fmt.Sprintf("%8d bytes", out.Len()-prevLen),
			e.output.OutFolder, fileName)

		path := filepath.Join(e.output.OutFolder, fileName)

		// MAKE SURE TO CLOSE THE FILE ON EVERY EXIT PATH!!!!!!
		dest, err := os.Create(path) // Ensure the file exists
		if err != nil {
			return fmt.Errorf("creating output file %s: %w", path, err)
		}

		// MAKE SURE TO CLOSE THE FILE ON EVERY EXIT PATH!!!!!!
		if err := lang.Write(e.data.Importer, e.output.PkgName, e.output.OutFolder, out, tests, dest); err != nil {
			dest.Close()
			return fmt.Errorf("writing file: %w", err)
		}

		dest.Close()
	}

	return nil
}

// executeTemplate takes a template and returns the output of the template
// execution.
func executeTemplate[T, C, I any](buf io.Writer, t *template.Template, name string, data *TemplateData[T, C, I]) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to execute template: %s\npanic: %+v", name, r)
		}
	}()

	if err := t.ExecuteTemplate(buf, name, data); err != nil {
		return fmt.Errorf("failed to execute template: %s: %w", name, err)
	}
	return nil
}

type extMap map[string][]string

// groupTemplatesByExtension takes templates and groups them according to their output directory
// and file extension.
func groupTemplatesByExtension(templates *template.Template) extMap {
	tplNames := templates.Templates()
	extensions := make(map[string][]string)
	for _, tpl := range tplNames {
		if !strings.HasSuffix(tpl.Name(), ".tpl") {
			continue
		}

		ext := filepath.Ext(strings.TrimSuffix(tpl.Name(), ".tpl"))
		slice := extensions[ext]
		extensions[ext] = append(slice, tpl.Name())
	}

	for _, tplNames := range extensions {
		slices.Sort(tplNames)
	}

	return extensions
}
