package language

import (
	"regexp"
	"strings"
	"sync"

	"golang.org/x/tools/go/packages"
)

//nolint:gochecknoglobals
var (
	pkgRgx           = regexp.MustCompile(`"([^"]+)"`)
	standardPackages = make(map[string]struct{})
	stdPkgOnce       sync.Once
)

func getStandardPackages() map[string]struct{} {
	stdPkgOnce.Do(func() {
		pkgs, err := packages.Load(nil, "std")
		if err != nil {
			panic(err)
		}

		for _, p := range pkgs {
			standardPackages[p.PkgPath] = struct{}{}
		}
	})

	return standardPackages
}

func combineStringSlices(a, b []string) []string {
	c := make([]string, len(a)+len(b))
	if len(a) > 0 {
		copy(c, a)
	}
	if len(b) > 0 {
		copy(c[len(a):], b)
	}

	return c
}

type Importer interface {
	Import(...string) string
	ImportList(list []string) string
	ToList() []string
}

type defaultImporter map[string]struct{}

// To be used inside templates to record an import.
// Always returns an empty string
func (i defaultImporter) Import(pkgs ...string) string {
	pkg := strings.Join(pkgs, " ")

	i[pkg] = struct{}{}
	return ""
}

func (i defaultImporter) ImportList(list []string) string {
	for _, p := range list {
		i[p] = struct{}{}
	}
	return ""
}

func (i defaultImporter) ToList() []string {
	list := make([]string, 0, len(i))
	for pkg := range i {
		list = append(list, pkg)
	}

	return list
}
