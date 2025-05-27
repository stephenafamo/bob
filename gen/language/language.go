package language

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"mvdan.cc/gofumpt/format"
)

var rgxSyntaxError = regexp.MustCompile(`(\d+):\d+: `)

type Languages struct {
	GeneratorName           string
	GoVersion               string
	SeparatePackageForTests bool
}

func (o Languages) GetOutputLanguage(ext string) Language {
	switch ext {
	case ".go":
		return goOutputLanguage{
			Generator:               o.GeneratorName,
			Version:                 o.GoVersion,
			SeparatePackageForTests: o.SeparatePackageForTests,
		}
	case ".sql":
		return sqlOutputLanguage{
			Generator: o.GeneratorName,
		}
	default:
		return unknownOutputLanguage{extension: ext}
	}
}

type Language interface {
	Importer() Importer
	IsTest(templateName string) bool
	OutputFileName(schema, tableName string, isTest bool) string
	WriteHeader(
		out *bytes.Buffer, imps ImportList,
		pkgName string, isTest bool,
	) error
	Format(buf io.Reader) ([]byte, error)
	Disclaimer() string
}

type unknownOutputLanguage struct {
	extension string
}

func (u unknownOutputLanguage) Importer() Importer {
	// No imports for unknown language
	return defaultImporter{}
}

func (u unknownOutputLanguage) IsTest(templateName string) bool {
	return false
}

func (u unknownOutputLanguage) OutputFileName(schema, tableName string, isTest bool) string {
	name := fmt.Sprintf("%s.bob%s", tableName, u.extension)

	if schema != "" {
		name = schema + "." + name
	}

	return name
}

func (unknownOutputLanguage) WriteHeader(
	out *bytes.Buffer, imps ImportList,
	pkgName string, isTest bool,
) error {
	// Cannot write header for unknown language
	return nil
}

func (unknownOutputLanguage) Format(buf io.Reader) ([]byte, error) {
	// Cannot format unknown language
	return io.ReadAll(buf)
}

// Disclaimer implements outputLanguage.
func (u unknownOutputLanguage) Disclaimer() string {
	return ""
}

// Copied from the go source
// see: https://github.com/golang/go/blob/master/src/internal/syslist/syslist.go
//
//nolint:gochecknoglobals
var (
	goosList = strings.Fields("aix android darwin dragonfly freebsd hurd illumos ios js linux nacl netbsd openbsd plan9 solaris windows zos")

	goarchList = strings.Fields("386 amd64 amd64p32 arm armbe arm64 arm64be loong64 mips mipsle mips64 mips64le mips64p32 mips64p32le ppc ppc64 ppc64le riscv riscv64 s390 s390x sparc sparc64 wasm")
)

const noEditDisclaimerFmt = `Code generated%s. DO NOT EDIT.
This file is meant to be re-generated in place and/or deleted at any time.`

type goOutputLanguage struct {
	Generator               string
	Version                 string
	SeparatePackageForTests bool
}

func (g goOutputLanguage) Importer() Importer {
	// No imports for Go
	return goImporter{}
}

func (g goOutputLanguage) IsTest(templateName string) bool {
	return strings.HasSuffix(templateName, "_test.go.tpl")
}

func (g goOutputLanguage) OutputFileName(schema, tableName string, isTest bool) string {
	output := tableName
	if strings.HasPrefix(output, "_") {
		output = "und" + output
	}

	if g.endsWithSpecialSuffix(output) {
		output += "_model"
	}

	if schema != "" {
		output = schema + "." + output
	}

	if isTest {
		return output + ".bob_test.go"
	}

	return output + ".bob.go"
}

// See: https://pkg.go.dev/cmd/go#hdr-Build_constraints
func (g goOutputLanguage) endsWithSpecialSuffix(tableName string) bool {
	parts := strings.Split(tableName, "_")

	// Not enough parts to have a special suffix
	if len(parts) < 2 {
		return false
	}

	lastPart := parts[len(parts)-1]

	if lastPart == "test" {
		return true
	}

	if slices.Contains(goosList, lastPart) {
		return true
	}
	if slices.Contains(goarchList, lastPart) {
		return true
	}

	return false
}

// WriteHeader implements OutputLanguage.
func (g goOutputLanguage) WriteHeader(
	out *bytes.Buffer, imps ImportList,
	pkgName string, isTest bool,
) error {
	// Write disclaimer
	if _, err := fmt.Fprint(out, g.Disclaimer()); err != nil {
		return fmt.Errorf("writing disclaimer: %w", err)
	}

	// Write package name
	if isTest && g.SeparatePackageForTests {
		pkgName = fmt.Sprintf("%s_test", pkgName)
	}
	if _, err := fmt.Fprintf(out, "package %s\n\n", pkgName); err != nil {
		return fmt.Errorf("writing package name: %w", err)
	}

	// Write imports
	if impStr := imps.Format(); len(impStr) > 0 {
		if _, err := fmt.Fprintf(out, "%s\n", impStr); err != nil {
			return fmt.Errorf("writing imports: %w", err)
		}
	}

	return nil
}

// Format implements OutputLanguage.
func (g goOutputLanguage) Format(buf io.Reader) ([]byte, error) {
	src, err := io.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	output, err := format.Source(src, format.Options{LangVersion: g.Version})
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

func (g goOutputLanguage) Disclaimer() string {
	// Write disclaimer
	noEditDisclaimer := fmt.Sprintf(noEditDisclaimerFmt, " ")
	if g.Generator != "" {
		noEditDisclaimer = fmt.Sprintf(noEditDisclaimerFmt, " by "+g.Generator)
	}

	noEditDisclaimer = strings.ReplaceAll(noEditDisclaimer, "\n", "\n// ")

	return fmt.Sprintf("// %s\n\n", noEditDisclaimer)
}

type sqlOutputLanguage struct {
	Generator string
}

// Importer implements outputLanguage.
func (s sqlOutputLanguage) Importer() Importer {
	// No imports for SQL
	return defaultImporter{}
}

// Format implements outputLanguage.
func (s sqlOutputLanguage) Format(buf io.Reader) ([]byte, error) {
	return io.ReadAll(buf)
}

// IsTest implements outputLanguage.
func (s sqlOutputLanguage) IsTest(templateName string) bool {
	return false
}

// OutputFileName implements outputLanguage.
func (s sqlOutputLanguage) OutputFileName(schema string, tableName string, isTest bool) string {
	if schema != "" {
		return fmt.Sprintf("%s.%s.bob.sql", schema, tableName)
	}

	return fmt.Sprintf("%s.bob.sql", tableName)
}

// WriteHeader implements outputLanguage.
func (g sqlOutputLanguage) WriteHeader(out *bytes.Buffer, imps ImportList, pkgName string, isTest bool) error {
	// Write disclaimer
	if _, err := fmt.Fprint(out, g.Disclaimer()); err != nil {
		return fmt.Errorf("writing disclaimer: %w", err)
	}

	return nil
}

// WriteHeader implements outputLanguage.
func (g sqlOutputLanguage) Disclaimer() string {
	noEditDisclaimer := fmt.Sprintf(noEditDisclaimerFmt, " ")
	if g.Generator != "" {
		noEditDisclaimer = fmt.Sprintf(noEditDisclaimerFmt, " by "+g.Generator)
	}

	noEditDisclaimer = strings.ReplaceAll(noEditDisclaimer, "\n", "\n-- ")
	return fmt.Sprintf("-- %s\n\n", noEditDisclaimer)
}
