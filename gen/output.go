package gen

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"text/template"

	"github.com/stephenafamo/bob/gen/importers"
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

type executeTemplateData[T any] struct {
	output *Output
	data   *templateData[T]

	templates     *templateList
	dirExtensions dirExtMap

	combineImportsOnType bool
	isTest               bool
}

// generateOutput builds the file output and sends it to outHandler for saving
func generateOutput[T any](o *Output, dirExts dirExtMap, data *templateData[T]) error {
	return executeTemplates(executeTemplateData[T]{
		output:               o,
		data:                 data,
		templates:            o.templates,
		combineImportsOnType: true,
		dirExtensions:        dirExts,
	})
}

// generateTestOutput builds the test file output and sends it to outHandler for saving
func generateTestOutput[T any](o *Output, dirExts dirExtMap, data *templateData[T]) error {
	return executeTemplates(executeTemplateData[T]{
		output:               o,
		data:                 data,
		templates:            o.testTemplates,
		combineImportsOnType: false,
		isTest:               true,
		dirExtensions:        dirExts,
	})
}

// generateSingletonOutput processes the templates that should only be run
// one time.
func generateSingletonOutput[T any](o *Output, data *templateData[T]) error {
	return executeSingletonTemplates(executeTemplateData[T]{
		output:    o,
		data:      data,
		templates: o.templates,
	})
}

// generateSingletonTestOutput processes the templates that should only be run
// one time.
func generateSingletonTestOutput[T any](o *Output, data *templateData[T]) error {
	return executeSingletonTemplates(executeTemplateData[T]{
		output:    o,
		data:      data,
		templates: o.testTemplates,
		isTest:    true,
	})
}

func executeTemplates[T any](e executeTemplateData[T]) error {
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
				fmt.Fprintf(os.Stderr, "skipping empty file: %s/%s\n", e.output.OutFolder, fName)
				continue
			}

			imps := e.data.Importer.ToList()
			if isGo {
				pkgName := e.output.PkgName
				if len(dir) != 0 {
					pkgName = filepath.Base(dir)
				}
				writeFileDisclaimer(headerOut)
				writePackageName(headerOut, pkgName)
				writeImports(headerOut, imps)
			}

			if err := writeFile(e.output.OutFolder, fName, io.MultiReader(headerOut, out), isGo); err != nil {
				return err
			}
		}
	}

	return nil
}

func executeSingletonTemplates[T any](e executeTemplateData[T]) error {
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
			continue
		}

		if isGo {
			imps := e.data.Importer.ToList()

			pkgName := e.output.PkgName
			if !usePkg {
				pkgName = filepath.Base(dir)
			}
			writeFileDisclaimer(headerOut)
			writePackageName(headerOut, pkgName)
			writeImports(headerOut, imps)
		}

		if err := writeFile(e.output.OutFolder, normalized, io.MultiReader(headerOut, out), isGo); err != nil {
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
func writeFile(outFolder string, fileName string, input io.Reader, format bool) error {
	var byt []byte
	var err error
	if format {
		byt, err = formatBuffer(input)
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
func executeTemplate[T any](buf io.Writer, t *template.Template, name string, data *templateData[T]) (err error) {
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

func formatBuffer(buf io.Reader) ([]byte, error) {
	src, err := io.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	output, err := format.Source(src)
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
