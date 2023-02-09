package driver

import (
	"fmt"
	"io/fs"
	"path/filepath"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func parseHCLPaths(files fs.FS) (*hclparse.Parser, error) {
	p := hclparse.NewParser()

	dir, err := fs.ReadDir(files, ".")
	if err != nil {
		return nil, err
	}
	for _, f := range dir {
		// Skip nested dirs.
		if f.IsDir() {
			continue
		}
		if err := mayParse(p, files, f.Name()); err != nil {
			return nil, err
		}
	}

	if len(p.Files()) == 0 {
		return nil, fmt.Errorf("no schema files found")
	}
	return p, nil
}

// mayParse will parse the file in path if it is an HCL file. If the file is an Atlas
// project file an error is returned.
func mayParse(p *hclparse.Parser, f fs.FS, path string) error {
	if n := filepath.Base(path); filepath.Ext(n) != ".hcl" {
		return nil
	}
	fileContents, err := fs.ReadFile(f, path)
	if err != nil {
		return err
	}
	switch f, diag := p.ParseHCL(fileContents, path); {
	case diag.HasErrors():
		return diag
	case isProjectFile(f):
		return fmt.Errorf("cannot parse project file %q as a schema file", path)
	default:
		return nil
	}
}

func isProjectFile(f *hcl.File) bool {
	for _, blk := range f.Body.(*hclsyntax.Body).Blocks {
		if blk.Type == "env" {
			return true
		}
	}
	return false
}
