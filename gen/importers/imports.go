// Package importers helps with dynamic imports for templating
package importers

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
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

// List of imports
type List []string

// Len implements sort.Interface.Len
func (l List) Len() int {
	return len(l)
}

// Swap implements sort.Interface.Swap
func (l List) Swap(i, j int) {
	l[i], l[j] = l[j], l[i]
}

// Less implements sort.Interface.Less
func (l List) Less(i, j int) bool {
	res := strings.Compare(strings.TrimLeft(l[i], "_ "), strings.TrimLeft(l[j], "_ "))
	return res <= 0
}

func (l List) GetSorted() (List, List) {
	var std, third List //nolint:prealloc
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

		third = append(third, pkg)

	}

	// Make sure the lists are sorted, so that the output is consistent
	sort.Sort(std)
	sort.Sort(third)

	return std, third
}

// Format the set into Go syntax (compatible with go imports)
func (l List) Format() []byte {
	if len(l) < 1 {
		return []byte{}
	}

	if len(l) == 1 {
		return []byte(fmt.Sprintf("import %s", l[0]))
	}

	standard, thirdparty := l.GetSorted()

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
