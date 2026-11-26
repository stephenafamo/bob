//go:build !race

package bobcheck_test

import (
	"path/filepath"
	"testing"

	"github.com/stephenafamo/bob/analysis/bobcheck"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDiscardedWithMods(t *testing.T) {
	// Load the fixtures in this module to use the real Bob query interface.
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	analysistest.Run(t, root, bobcheck.New(), "./analysis/bobcheck/testdata/caller")
}
