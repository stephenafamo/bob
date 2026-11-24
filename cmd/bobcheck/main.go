// Command bobcheck reports discarded results of query-returning Apply methods.
package main

import (
	"github.com/stephenafamo/bob/analysis/bobcheck"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(bobcheck.New())
}
