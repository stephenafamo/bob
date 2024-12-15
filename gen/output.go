package gen

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen/importers"
	"mvdan.cc/gofumpt/format"
)

// Copied from the go source
// see: https://github.com/golang/go/blob/master/src/go/build/syslist.go
//
//nolint:gochecknoglobals
var (
	goosList = stringSliceToMap(strings.Fields("aix android darwin dragonfly freebsd hurd illumos ios js linux nacl netbsd openbsd plan9 solaris windows zos"))

	goarchList = stringSliceToMap(strings.Fields("386 amd64 amd64p32 arm armbe arm64 arm64be loong64 mips mipsle mips64 mips64le mips64p32 mips64p32le ppc ppc64 ppc64le riscv riscv64 s390 s390x sparc sparc64 wasm"))
)

//nolint:gochecknoglobals
var (
	noEditDisclaimerFmt = `// Code generated%s. DO NOT EDIT.
// This file is meant to be re-generated in place and/or deleted at any time.

`
	noEditDisclaimer = []byte(fmt.Sprintf(noEditDisclaimerFmt, " "))
)

//nolint:gochecknoglobals
var (
	rgxRemoveNumberedPrefix = regexp.MustCompile(`^[0-9]+_`)
	rgxSyntaxError          = regexp.MustCompile(`(\d+):\d+: `)

	testHarnessWriteFile = os.WriteFile
)

type Output struct {
	// The key has to be unique in a gen.State
	// it also makes it possible to target modifing a specific output
	// There are special keys that are reserved for internal use
	// * "models" - for model templates.
	//    This is also used to set `ModelsPackage` in the template data
	// * "factory" - for factory templates
	// * "queries" - for query templates.
	//    - This is run once for each query folder
	//    - The PkgName is set to the folder name in each run
	//    - The OutFolder is set to the same folder
	Key string

	PkgName   string
	OutFolder string
	Templates []fs.FS

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
func (o *Output) initOutFolders(wipe bool) error {
	if wipe && !strings.Contains(o.OutFolder, "quer") {
		if err := os.RemoveAll(o.OutFolder); err != nil {
			return fmt.Errorf("unable to wipe output folder: %w", err)
		}
	}

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
		return errors.New("No templates defined")
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

			if singletons {
				ext2 := filepath.Ext(name[:len(name)-len(ext)])
				fNameWithoutExts := filepath.Base(name[:len(name)-len(ext)-len(ext2)])
				if !strings.HasSuffix(fNameWithoutExts, ".bob") &&
					!strings.HasSuffix(fNameWithoutExts, ".bob_test") {
					panic(fmt.Sprintf("singleton file name must end with .bob or .bob_test: %s", name))
				}
			}

			all[name] = details{
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

	templates     *template.Template
	dirExtensions dirExtMap
}

// generateOutput builds the file output and sends it to outHandler for saving
func generateOutput[T, C, I any](o *Output, dirExts dirExtMap, tpl *template.Template, data *TemplateData[T, C, I], goVersion string, noTests bool) error {
	if err := executeTemplates(executeTemplateData[T, C, I]{
		output:        o,
		data:          data,
		templates:     tpl,
		dirExtensions: dirExts,
	}, goVersion, false); err != nil {
		return fmt.Errorf("execute templates: %w", err)
	}

	if noTests {
		return nil
	}

	if err := executeTemplates(executeTemplateData[T, C, I]{
		output:        o,
		data:          data,
		templates:     tpl,
		dirExtensions: dirExts,
	}, goVersion, true); err != nil {
		return fmt.Errorf("execute test templates: %w", err)
	}

	return nil
}

// generateSingletonOutput processes the templates that should only be run
// one time.
func generateSingletonOutput[T, C, I any](o *Output, data *TemplateData[T, C, I], goVersion string, noTests bool) error {
	if err := executeSingletonTemplates(executeTemplateData[T, C, I]{
		output:    o,
		data:      data,
		templates: o.singletonTemplates,
	}, goVersion, false); err != nil {
		return fmt.Errorf("execute singleton templates: %w", err)
	}

	if noTests {
		return nil
	}

	if err := executeSingletonTemplates(executeTemplateData[T, C, I]{
		output:    o,
		data:      data,
		templates: o.singletonTemplates,
	}, goVersion, true); err != nil {
		return fmt.Errorf("execute singleton test templates: %w", err)
	}

	return nil
}

func executeTemplates[T, C, I any](e executeTemplateData[T, C, I], goVersion string, tests bool) error {
	for dir, dirExts := range e.dirExtensions {
		for ext, tplNames := range dirExts {
			headerOut := e.output.templateHeaderByteBuffer
			headerOut.Reset()
			out := e.output.templateByteBuffer
			out.Reset()

			isGo := filepath.Ext(ext) == ".go"

			prevLen := out.Len()
			e.data.ResetImports()

			matchingTemplates := 0
			for _, tplName := range tplNames {
				if tests != strings.Contains(tplName, "_test.go") {
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

			fName := getOutputFilename(e.data.Table.Schema, e.data.Table.Name, isGo)
			fName += ".bob"
			if tests {
				fName += "_test"
			}

			fName += ext
			if len(dir) != 0 {
				fName = filepath.Join(dir, fName)
			}

			// Skip writing the file if the content is empty
			if out.Len()-prevLen < 1 {
				fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
					"== SKIPPED ==", e.output.OutFolder, fName)
				continue
			}

			fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
				fmt.Sprintf("%7d bytes", out.Len()-prevLen),
				e.output.OutFolder, fName)

			imps := e.data.Importer.ToList()
			version := ""
			if isGo {
				pkgName := e.output.PkgName
				if len(dir) != 0 {
					pkgName = filepath.Base(dir)
				}
				if tests {
					pkgName = fmt.Sprintf("%s_test", pkgName)
				}
				version = goVersion
				writeFileDisclaimer(headerOut)
				writePackageName(headerOut, pkgName)
				writeImports(headerOut, imps)
			}

			if err := writeFile(e.output.OutFolder, fName, io.MultiReader(headerOut, out), version); err != nil {
				return err
			}
		}
	}

	return nil
}

func executeSingletonTemplates[T, C, I any](e executeTemplateData[T, C, I], goVersion string, tests bool) error {
	headerOut := e.output.templateHeaderByteBuffer
	out := e.output.templateByteBuffer
	for _, tpl := range e.templates.Templates() {
		if !strings.HasSuffix(tpl.Name(), ".tpl") {
			continue
		}

		if tests != strings.Contains(tpl.Name(), "_test.go") {
			continue
		}

		normalized, isGo := outputFilenameParts(tpl.Name())

		headerOut.Reset()
		out.Reset()
		prevLen := out.Len()

		e.data.ResetImports()
		if err := executeTemplate(out, e.templates, tpl.Name(), e.data); err != nil {
			return err
		}

		// Skip writing the file if the content is empty
		if out.Len()-prevLen < 1 {
			fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
				"== SKIPPED ==", e.output.OutFolder, normalized)
			continue
		}

		fmt.Fprintf(os.Stderr, "%-20s %s/%s\n",
			fmt.Sprintf("%7d bytes", out.Len()-prevLen),
			e.output.OutFolder, normalized)

		version := ""
		if isGo {
			imps := e.data.Importer.ToList()
			version = goVersion

			writeFileDisclaimer(headerOut)
			writePackageName(headerOut, e.output.PkgName)
			writeImports(headerOut, imps)
		}

		if err := writeFile(e.output.OutFolder, normalized, io.MultiReader(headerOut, out), version); err != nil {
			return err
		}
	}

	return nil
}

// writeFileDisclaimer writes the disclaimer at the top with a trailing
// newline so the package name doesn't get attached to it.
func writeFileDisclaimer(out *bytes.Buffer) {
	_, _ = out.Write(noEditDisclaimer)
}

// writePackageName writes the package name correctly, ignores errors
// since it's to the concrete buffer type which produces none
func writePackageName(out *bytes.Buffer, pkgName string) {
	_, _ = fmt.Fprintf(out, "package %s\n\n", pkgName)
}

// writeImports writes the package imports correctly, ignores errors
// since it's to the concrete buffer type which produces none
func writeImports(out *bytes.Buffer, imps importers.List) {
	if impStr := imps.Format(); len(impStr) > 0 {
		_, _ = fmt.Fprintf(out, "%s\n", impStr)
	}
}

// writeFile writes to the given folder and filename, formatting the buffer
// given.
// If goVersion is empty, the file is not formatted.
func writeFile(outFolder string, fileName string, input io.Reader, goVersion string) error {
	var byt []byte
	var err error
	if goVersion != "" {
		byt, err = formatBuffer(input, goVersion)
		if err != nil {
			return err
		}
	} else {
		byt, err = io.ReadAll(input)
		if err != nil {
			return err
		}
	}

	path := filepath.Join(outFolder, fileName)
	if err := testHarnessWriteFile(path, byt, 0o664); err != nil {
		return fmt.Errorf("failed to write output file %s: %w", path, err)
	}

	return nil
}

// executeTemplate takes a template and returns the output of the template
// execution.
func executeTemplate[T, C, I any](buf io.Writer, t *template.Template, name string, data *TemplateData[T, C, I]) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("failed to execute template: %s\npanic: %+v\n", name, r)
		}
	}()

	if err := t.ExecuteTemplate(buf, name, data); err != nil {
		return fmt.Errorf("failed to execute template: %s: %w", name, err)
	}
	return nil
}

func formatBuffer(buf io.Reader, version string) ([]byte, error) {
	src, err := io.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	output, err := format.Source(src, format.Options{LangVersion: version})
	if err == nil {
		return output, nil
	}

	matches := rgxSyntaxError.FindStringSubmatch(err.Error())
	if matches == nil {
		return nil, fmt.Errorf("failed to format template: %w", err)
	}

	lineNum, _ := strconv.Atoi(matches[1])
	scanner := bufio.NewScanner(bytes.NewBuffer(src))
	errBuf := &bytes.Buffer{}
	line := 1
	for ; scanner.Scan(); line++ {
		if delta := line - lineNum; delta < -5 || delta > 5 {
			continue
		}

		if line == lineNum {
			errBuf.WriteString(">>>> ")
		} else {
			fmt.Fprintf(errBuf, "% 4d ", line)
		}
		errBuf.Write(scanner.Bytes())
		errBuf.WriteByte('\n')
	}

	return nil, fmt.Errorf("failed to format template\n\n%s\n:%w", errBuf.Bytes(), err)
}

func getLongExt(filename string) string {
	index := strings.IndexByte(filename, '.')
	return filename[index:]
}

func getOutputFilename(schema, tableName string, isGo bool) string {
	output := tableName
	if strings.HasPrefix(output, "_") {
		output = "und" + output
	}

	if isGo && endsWithSpecialSuffix(output) {
		output += "_model"
	}

	if schema != "" {
		output += "." + schema
	}

	return output
}

// See: https://pkg.go.dev/cmd/go#hdr-Build_constraints
func endsWithSpecialSuffix(tableName string) bool {
	parts := strings.Split(tableName, "_")

	// Not enough parts to have a special suffix
	if len(parts) < 2 {
		return false
	}

	lastPart := parts[len(parts)-1]

	if lastPart == "test" {
		return true
	}

	if _, ok := goosList[lastPart]; ok {
		return true
	}

	if _, ok := goarchList[lastPart]; ok {
		return true
	}

	return false
}

func stringSliceToMap(slice []string) map[string]struct{} {
	Map := make(map[string]struct{}, len(slice))
	for _, v := range slice {
		Map[v] = struct{}{}
	}

	return Map
}

// fileFragments will take something of the form:
// templates/singleton/hello.go.tpl
// templates_test/js/hello.js.tpl
//
//nolint:nonamedreturns
func outputFilenameParts(filename string) (normalized string, isGo bool) {
	fragments := strings.Split(filename, string(os.PathSeparator))

	newFilename := fragments[len(fragments)-1]
	newFilename = strings.TrimSuffix(newFilename, ".tpl")
	newFilename = rgxRemoveNumberedPrefix.ReplaceAllString(newFilename, "")
	ext := filepath.Ext(newFilename)
	isGo = ext == ".go"

	fragments[len(fragments)-1] = newFilename
	normalized = strings.Join(fragments, string(os.PathSeparator))

	return normalized, isGo
}

type dirExtMap map[string]map[string][]string

// groupTemplates takes templates and groups them according to their output directory
// and file extension.
func groupTemplates(templates *template.Template) dirExtMap {
	tplNames := templates.Templates()
	dirs := make(map[string]map[string][]string)
	for _, tpl := range tplNames {
		if !strings.HasSuffix(tpl.Name(), ".tpl") {
			continue
		}

		normalized, _ := outputFilenameParts(tpl.Name())
		dir := filepath.Dir(normalized)
		if dir == "." {
			dir = ""
		}

		extensions, ok := dirs[dir]
		if !ok {
			extensions = make(map[string][]string)
			dirs[dir] = extensions
		}

		ext := getLongExt(tpl.Name())
		ext = strings.TrimSuffix(ext, ".tpl")
		slice := extensions[ext]
		extensions[ext] = append(slice, tpl.Name())
	}

	for _, exts := range dirs {
		for _, tplNames := range exts {
			slices.Sort(tplNames)
		}
	}

	return dirs
}
