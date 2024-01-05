package gen

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
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
	// templateByteBuffer is re-used by all template construction to avoid
	// allocating more memory than is needed. This will later be a problem for
	// concurrency, address it then.
	templateByteBuffer       = &bytes.Buffer{}
	templateHeaderByteBuffer = &bytes.Buffer{}

	rgxRemoveNumberedPrefix = regexp.MustCompile(`^[0-9]+_`)
	rgxSyntaxError          = regexp.MustCompile(`(\d+):\d+: `)

	testHarnessWriteFile = os.WriteFile
)

type Output struct {
	// The key has to be unique in a gen.State
	// it also makes it possible to target modifing a specific output
	Key string

	PkgName   string
	OutFolder string
	Templates []fs.FS

	templates     *templateList
	testTemplates *templateList
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

type executeTemplateData[T any] struct {
	output *Output
	data   *TemplateData[T]

	templates     *templateList
	dirExtensions dirExtMap

	isTest bool
}

// generateOutput builds the file output and sends it to outHandler for saving
func generateOutput[T any](o *Output, dirExts dirExtMap, data *TemplateData[T], goVersion string) error {
	return executeTemplates(executeTemplateData[T]{
		output:        o,
		data:          data,
		templates:     o.templates,
		dirExtensions: dirExts,
	}, goVersion)
}

// generateTestOutput builds the test file output and sends it to outHandler for saving
func generateTestOutput[T any](o *Output, dirExts dirExtMap, data *TemplateData[T], goVersion string) error {
	return executeTemplates(executeTemplateData[T]{
		output:        o,
		data:          data,
		templates:     o.testTemplates,
		isTest:        true,
		dirExtensions: dirExts,
	}, goVersion)
}

// generateSingletonOutput processes the templates that should only be run
// one time.
func generateSingletonOutput[T any](o *Output, data *TemplateData[T], goVersion string) error {
	return executeSingletonTemplates(executeTemplateData[T]{
		output:    o,
		data:      data,
		templates: o.templates,
	}, goVersion)
}

// generateSingletonTestOutput processes the templates that should only be run
// one time.
func generateSingletonTestOutput[T any](o *Output, data *TemplateData[T], goVersion string) error {
	return executeSingletonTemplates(executeTemplateData[T]{
		output:    o,
		data:      data,
		templates: o.testTemplates,
		isTest:    true,
	}, goVersion)
}

func executeTemplates[T any](e executeTemplateData[T], goVersion string) error {
	for dir, dirExts := range e.dirExtensions {
		for ext, tplNames := range dirExts {
			headerOut := templateHeaderByteBuffer
			headerOut.Reset()
			out := templateByteBuffer
			out.Reset()

			isGo := filepath.Ext(ext) == ".go"

			prevLen := out.Len()
			e.data.ResetImports()
			for _, tplName := range tplNames {
				if err := executeTemplate(out, e.templates.Template, tplName, e.data); err != nil {
					return err
				}
			}

			fName := getOutputFilename(e.data.Table.Schema, e.data.Table.Name, e.isTest, isGo)
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

func executeSingletonTemplates[T any](e executeTemplateData[T], goVersion string) error {
	headerOut := templateHeaderByteBuffer
	out := templateByteBuffer
	for _, tplName := range e.templates.Templates() {
		normalized, isSingleton, isGo, usePkg := outputFilenameParts(tplName)
		if !isSingleton {
			continue
		}

		dir, _ := filepath.Split(normalized)

		headerOut.Reset()
		out.Reset()
		prevLen := out.Len()

		e.data.ResetImports()
		if err := executeTemplate(out, e.templates.Template, tplName, e.data); err != nil {
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

			pkgName := e.output.PkgName
			if !usePkg {
				pkgName = filepath.Base(dir)
			}
			writeFileDisclaimer(headerOut)
			writePackageName(headerOut, pkgName)
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
func executeTemplate[T any](buf io.Writer, t *template.Template, name string, data *TemplateData[T]) (err error) {
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

func getOutputFilename(schema, tableName string, isTest, isGo bool) string {
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

	if isTest {
		output += "_test"
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
func outputFilenameParts(filename string) (normalized string, isSingleton, isGo, usePkg bool) {
	fragments := strings.Split(filename, string(os.PathSeparator))
	isSingleton = len(fragments) > 1 && fragments[len(fragments)-2] == "singleton"

	var remainingFragments []string
	for _, f := range fragments {
		if f != "singleton" {
			remainingFragments = append(remainingFragments, f)
		}
	}

	newFilename := remainingFragments[len(remainingFragments)-1]
	newFilename = strings.TrimSuffix(newFilename, ".tpl")
	newFilename = rgxRemoveNumberedPrefix.ReplaceAllString(newFilename, "")
	remainingFragments[len(remainingFragments)-1] = newFilename

	ext := filepath.Ext(newFilename)
	isGo = ext == ".go"

	usePkg = len(remainingFragments) == 1
	normalized = strings.Join(remainingFragments, string(os.PathSeparator))

	return normalized, isSingleton, isGo, usePkg
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

// normalizeSlashes takes a path that was made on linux or windows and converts it
// to a native path.
func normalizeSlashes(path string) string {
	path = strings.ReplaceAll(path, `/`, string(os.PathSeparator))
	path = strings.ReplaceAll(path, `\`, string(os.PathSeparator))
	return path
}
