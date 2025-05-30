package language

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/mod/modfile"
	"mvdan.cc/gofumpt/format"
)

// Copied from the go source
// see: https://github.com/golang/go/blob/master/src/internal/syslist/syslist.go
//
//nolint:gochecknoglobals
var (
	goosList = strings.Fields("aix android darwin dragonfly freebsd hurd illumos ios js linux nacl netbsd openbsd plan9 solaris windows zos")

	goarchList = strings.Fields("386 amd64 amd64p32 arm armbe arm64 arm64be loong64 mips mipsle mips64 mips64le mips64p32 mips64p32le ppc ppc64 ppc64le riscv riscv64 s390 s390x sparc sparc64 wasm")
)

type goOutputLanguage struct {
	Generator       string
	separateTestPkg bool
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

func (g goOutputLanguage) Write(
	imps Importer,
	pkgName string, folder string,
	contents io.Reader, inTest bool,
	destination io.Writer,
) error {
	packagePath, version, err := PackageForFolder(folder)
	if err != nil {
		return fmt.Errorf("getting package for folder %q: %w", folder, err)
	}

	header := &bytes.Buffer{}

	// Write disclaimer
	if _, err := fmt.Fprint(header, g.Disclaimer()); err != nil {
		return fmt.Errorf("writing disclaimer: %w", err)
	}

	// Write package name
	if inTest && g.separateTestPkg {
		pkgName = fmt.Sprintf("%s_test", pkgName)
	}
	if _, err := fmt.Fprintf(header, "package %s\n\n", pkgName); err != nil {
		return fmt.Errorf("writing package name: %w", err)
	}

	// Write imports
	if impStr := g.formatGoImports(imps.ToList(), packagePath, inTest); len(impStr) > 0 {
		if _, err := fmt.Fprintf(header, "%s\n", impStr); err != nil {
			return fmt.Errorf("writing imports: %w", err)
		}
	}

	formatted, err := g.format(io.MultiReader(header, contents), version)
	if err != nil {
		return fmt.Errorf("formatting: %w", err)
	}

	if _, err := destination.Write(formatted); err != nil {
		return fmt.Errorf("writing to destination: %w", err)
	}

	return nil
}

// format implements OutputLanguage.
func (g goOutputLanguage) format(buf io.Reader, version string) ([]byte, error) {
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

func (g goOutputLanguage) Disclaimer() string {
	// Write disclaimer
	noEditDisclaimer := fmt.Sprintf(noEditDisclaimerFmt, " ")
	if g.Generator != "" {
		noEditDisclaimer = fmt.Sprintf(noEditDisclaimerFmt, " by "+g.Generator)
	}

	noEditDisclaimer = strings.ReplaceAll(noEditDisclaimer, "\n", "\n// ")

	return fmt.Sprintf("// %s\n\n", noEditDisclaimer)
}

func PackageForFolder(folder string) (string, string, error) {
	modRoot, modFile, err := goModInfo(folder)
	if err != nil {
		return "", "", fmt.Errorf("getting mod details: %w", err)
	}

	fullPath := folder
	if !filepath.IsAbs(folder) {
		wd, err := os.Getwd()
		if err != nil {
			return "", "", fmt.Errorf("could not get working directory: %w", err)
		}

		fullPath = filepath.Join(wd, folder)
	}

	relPath := strings.TrimPrefix(fullPath, modRoot)

	return path.Join(modFile.Module.Mod.Path, filepath.ToSlash(relPath)), getGoVersion(modFile), nil
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

// getGoVersion returns the required go version from the package
func getGoVersion(modFile *modfile.File) string {
	if modFile.Toolchain != nil {
		return modFile.Toolchain.Name
	}

	return strings.Join(modFile.Go.Syntax.Token, "")
}

type goImporter map[string]struct{}

// To be used inside templates to record an import.
// Always returns an empty string
func (i goImporter) Import(pkgs ...string) string {
	switch len(pkgs) {
	case 1:
		i[fmt.Sprintf("%q", pkgs[0])] = struct{}{}
	case 2:
		i[fmt.Sprintf("%s %q", pkgs[0], pkgs[1])] = struct{}{}
	}

	return ""
}

func (i goImporter) ImportList(list []string) string {
	for _, p := range list {
		i[p] = struct{}{}
	}
	return ""
}

func (i goImporter) ToList() []string {
	list := make([]string, 0, len(i))
	for pkg := range i {
		list = append(list, pkg)
	}

	return list
}

// Format the set into Go syntax (compatible with go imports)
func (g goOutputLanguage) formatGoImports(l []string, current string, inTest bool) []byte {
	if len(l) < 1 {
		return []byte{}
	}

	if len(l) == 1 {
		return fmt.Appendf(nil, "import %s", l[0])
	}

	standard, thirdparty := g.sortGoImports(l, current, inTest)

	buf := &bytes.Buffer{}
	buf.WriteString("import (")
	for _, std := range standard {
		fmt.Fprintf(buf, "\n\t%s", std)
	}
	if len(standard) != 0 && len(thirdparty) != 0 {
		buf.WriteString("\n")
	}
	for _, third := range thirdparty {
		fmt.Fprintf(buf, "\n\t%s", third)
	}
	buf.WriteString("\n)\n")

	return buf.Bytes()
}

func (g goOutputLanguage) sortGoImports(l []string, current string, inTest bool) ([]string, []string) {
	quotedCurrent := fmt.Sprintf("%q", current)
	var std, third []string //nolint:prealloc
	for _, pkg := range l {
		if pkg == "" {
			continue
		}

		var pkgName string
		if pkgSlice := pkgRgx.FindStringSubmatch(pkg); len(pkgSlice) > 0 {
			pkgName = pkgSlice[1]
		}

		if _, ok := getStandardPackages()[pkgName]; ok {
			std = append(std, pkg)
			continue
		}

		isTestPkg := strings.Contains(pkg, "_test")
		hasSuffix := strings.HasSuffix(pkg, quotedCurrent)

		// If in the current package, skip it
		if hasSuffix && (inTest && g.separateTestPkg) == isTestPkg {
			continue
		}

		third = append(third, pkg)
	}

	// Make sure the lists are sorted, so that the output is consistent
	slices.SortFunc(std, goImportSorter)
	slices.SortFunc(third, goImportSorter)

	return std, third
}

func goImportSorter(a, b string) int {
	return strings.Compare(strings.TrimLeft(a, "_ "), strings.TrimLeft(b, "_ "))
}
