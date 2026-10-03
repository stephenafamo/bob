package language

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPackageForFolderWithIgnoreDirective(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	goMod := "module example.com/app\n\ngo 1.25.0\n\nignore data\n"
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	pkg, version, err := PackageForFolder(filepath.Join(root, "models"))
	if err != nil {
		t.Fatalf("PackageForFolder: %v", err)
	}

	if pkg != "example.com/app/models" {
		t.Errorf("package = %q, want %q", pkg, "example.com/app/models")
	}
	if version != "go1.25.0" {
		t.Errorf("go version = %q, want %q", version, "go1.25.0")
	}
}
