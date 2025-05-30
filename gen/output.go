package gen

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen/language"
)

type Output struct {
	// The key has to be unique in a gen.State
	// it also makes it possible to target modifing a specific output
	// There are special keys that are reserved for internal use
	// * "models" - for model templates.
	// * "factory" - for factory templates
	// * "queries" - for query templates.
	//    - This is run once for each query folder
	//    - The PkgName is set to the folder name in each run
	//    - The OutFolder is set to the same folder
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
		return errors.New("no templates defined")
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

// generateOutput builds the file output and sends it to outHandler for saving
func generateOutput[T, C, I any](o *Output, dirExts extMap, tpl *template.Template, data *TemplateData[T, C, I], langs language.Languages, noTests bool) error {
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
func generateSingletonOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], langs language.Languages, noTests bool) error {
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
				"== SKIPPED ==", e.output.OutFolder, fName)
			continue
		}

		fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
			fmt.Sprintf("%7d bytes", out.Len()-prevLen),
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
		if err := executeTemplate(out, e.templates, tpl.Name(), e.data); err != nil {
			return err
		}

		// Skip writing the file if the content is empty
		if out.Len()-prevLen < 1 {
			fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
				"== SKIPPED ==", e.output.OutFolder, fileName)
			continue
		}

		fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
			fmt.Sprintf("%7d bytes", out.Len()-prevLen),
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
