package language

import (
	"fmt"
	"io"
	"regexp"
	"strings"
)

const noEditDisclaimerFmt = `Code generated%s. DO NOT EDIT.
This file is meant to be re-generated in place and/or deleted at any time.`

var rgxSyntaxError = regexp.MustCompile(`(\d+):\d+: `)

type Languages struct {
	GeneratorName           string
	SeparatePackageForTests bool
}

func (o Languages) GetOutputLanguage(ext string) Language {
	switch ext {
	case ".go":
		return goOutputLanguage{
			Generator:       o.GeneratorName,
			separateTestPkg: o.SeparatePackageForTests,
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
	Write(
		imps Importer,
		pkgName string, folder string,
		contents io.Reader, isTest bool,
		destination io.Writer,
	) error
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

func (unknownOutputLanguage) Write(
	imps Importer,
	pkgName string, folder string,
	contents io.Reader, isTest bool,
	destination io.Writer,
) error {
	if _, err := io.Copy(destination, contents); err != nil {
		return fmt.Errorf("writing contents: %w", err)
	}

	return nil
}

// Disclaimer implements outputLanguage.
func (u unknownOutputLanguage) Disclaimer() string {
	return ""
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

func (g sqlOutputLanguage) Write(
	imps Importer,
	pkgName string, folder string,
	contents io.Reader, isTest bool,
	destination io.Writer,
) error {
	// Write disclaimer
	if _, err := fmt.Fprint(destination, g.Disclaimer()); err != nil {
		return fmt.Errorf("writing disclaimer: %w", err)
	}

	if _, err := io.Copy(destination, contents); err != nil {
		return fmt.Errorf("writing to destination: %w", err)
	}

	return nil
}

func (g sqlOutputLanguage) Disclaimer() string {
	noEditDisclaimer := fmt.Sprintf(noEditDisclaimerFmt, " ")
	if g.Generator != "" {
		noEditDisclaimer = fmt.Sprintf(noEditDisclaimerFmt, " by "+g.Generator)
	}

	noEditDisclaimer = strings.ReplaceAll(noEditDisclaimer, "\n", "\n-- ")
	return fmt.Sprintf("-- %s\n\n", noEditDisclaimer)
}
