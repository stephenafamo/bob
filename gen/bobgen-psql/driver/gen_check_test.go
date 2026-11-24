package driver_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/bobgen-psql/driver"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/language"
	"github.com/stephenafamo/bob/gen/plugins"
)

func TestBobcheckAfterGenerationFromSubdirectory(t *testing.T) {
	bobRoot, err := language.ModuleRoot(".")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("GOWORK", "off")
	t.Setenv("GOFLAGS", "")
	t.Chdir(root)
	gomod := fmt.Sprintf(
		"module example.org/app\n\ngo 1.26.0\n\nrequire github.com/stephenafamo/bob v0.0.0\nreplace github.com/stephenafamo/bob => %q\n",
		filepath.ToSlash(bobRoot),
	)
	if err := os.WriteFile("go.mod", []byte(gomod), 0o600); err != nil {
		t.Fatal(err)
	}
	// The caller lives outside the generator's directory and imports a package
	// that does not exist until the generator finishes.
	const caller = `package app
import (
	_ "example.org/app/db/dbinfo"
	"github.com/stephenafamo/bob"
)
func use(q interface{ Apply() bob.Query }) { q.Apply() }`
	if err := os.WriteFile("caller_test.go", []byte(caller), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir("db", 0o700); err != nil {
		t.Fatal(err)
	}
	t.Chdir("db")
	d := stubDriver{types: helpers.Types(), info: &driver.DBInfo{
		Tables: drivers.Tables[any, driver.IndexExtra]{{Name: "jobs", Key: "jobs"}},
	}}
	run := func() error {
		config, _, _, err := helpers.GetConfigFromFile[any, driver.Config](helpers.DefaultConfigPath, "psql")
		if err != nil {
			return err
		}
		s := &gen.State[any]{Config: config}
		return gen.Run(t.Context(), s, d, plugins.DBInfo[any](plugins.OutputConfig{}))
	}
	const hint = "Files were generated. Run `go mod tidy` from the module root, then rerun the generator"
	if err := run(); err == nil || strings.Count(err.Error(), hint) != 1 {
		t.Fatalf("want one dependency hint for the fresh module, got %v", err)
	}
	tidy := exec.CommandContext(t.Context(), "go", "mod", "tidy")
	tidy.Dir = root
	if out, err := tidy.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy: %v\n%s", err, out)
	}
	if err := run(); err == nil || !strings.Contains(err.Error(), "caller_test.go:6:48: result of Apply is discarded") ||
		strings.Contains(err.Error(), hint) {
		t.Fatalf("want a diagnostic in the module root, got %v", err)
	}
	if err := os.WriteFile(helpers.DefaultConfigPath, []byte("disable_bobcheck: true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := run(); err != nil {
		t.Fatalf("check disabled: %v", err)
	}
}
