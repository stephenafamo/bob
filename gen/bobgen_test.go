package gen

import (
	"bufio"
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stephenafamo/bob/gen/drivers/mocks"
)

var rgxHasSpaces = regexp.MustCompile(`^\s+`)

func TestNew(t *testing.T) {
	testNew(t, Aliases{})
}

func TestNewWithAliases(t *testing.T) {
	aliases := Aliases{Tables: make(map[string]TableAlias)}
	mockDriver := mocks.MockDriver{}
	info, err := mockDriver.Assemble()
	if err != nil {
		t.Fatal(err)
	}

	for i, table := range info.Tables {
		tableAlias := TableAlias{
			UpPlural:     fmt.Sprintf("Alias%vThings", i),
			UpSingular:   fmt.Sprintf("Alias%vThing", i),
			DownPlural:   fmt.Sprintf("alias%vThings", i),
			DownSingular: fmt.Sprintf("alias%vThing", i),
		}
		_, _, columns, err := mockDriver.TableColumns(drivers.TableInfo{
			Key:    table.Key,
			Schema: table.Schema,
			Name:   table.Name,
		}, drivers.ColumnFilter{})
		if err != nil {
			t.Fatal(err)
		}
		tableAlias.Columns = make(map[string]string)
		for j, column := range columns {
			tableAlias.Columns[column.Name] = fmt.Sprintf("Alias%vThingColumn%v", i, j)
		}
		tableAlias.Relationships = make(map[string]string)

		aliases.Tables[table.Key] = tableAlias
	}

	testNew(t, aliases)
}

func testNew(t *testing.T, aliases Aliases) {
	t.Helper()

	if testing.Short() {
		t.SkipNow()
	}

	var err error
	out, err := os.MkdirTemp("", "bobgen_templates")
	if err != nil {
		t.Fatalf("unable to create tempdir: %s", err)
	}

	outputs := []*Output{
		{
			PkgName:   "models",
			OutFolder: filepath.Join(out, "models"),
			Templates: []fs.FS{ModelTemplates},
		},
		{
			PkgName:   "factory",
			OutFolder: filepath.Join(out, "factory"),
			Templates: []fs.FS{FactoryTemplates},
		},
	}
	// Defer cleanup of the tmp folder
	defer func() {
		if t.Failed() {
			t.Log("template test output:", out)
			return
		}
		for _, o := range outputs {
			os.RemoveAll(o.OutFolder)
		}
	}()

	module := "github.com/stephenafamo/bob/orm/bob-gen-test"
	config := &Config{
		NoTests:   true,
		TagIgnore: []string{"pass"},
		Aliases:   aliases,
		Generator: "BobGen",
	}

	state := &State[any]{
		Dialect:   "psql",
		Config:    config,
		Outputs:   outputs,
		ModelsPkg: path.Join(module, "models"),
	}
	if err != nil {
		t.Fatalf("Unable to create State using config: %s", err)
	}

	if err = state.Run(&mocks.MockDriver{}); err != nil {
		t.Errorf("Unable to execute State.Run: %s", err)
	}

	buf := &bytes.Buffer{}

	cmd := exec.Command("go", "env", "GOMOD")
	goModFilePath, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOMOD cmd execution failed: %s", err)
	}

	cmd = exec.Command("go", "mod", "init", module)
	cmd.Dir = out
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		t.Errorf("go mod init cmd execution failed: %s", err)
		outputCompileErrors(buf, out)
		fmt.Println()
	}

	cmd = exec.Command("go", "mod", "edit", fmt.Sprintf("-replace=github.com/stephenafamo/bob=%s", filepath.Dir(string(goModFilePath))))
	cmd.Dir = out
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		t.Errorf("go mod init cmd execution failed: %s", err)
		outputCompileErrors(buf, out)
		fmt.Println()
	}

	// From go1.16 dependencies are not auto downloaded
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = out
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		t.Errorf("go mod tidy cmd execution failed: %s", err)
		outputCompileErrors(buf, out)
		fmt.Println()
	}

	cmd = exec.Command("go", "test", "-run", "xxxxxxx", "./...")
	cmd.Dir = out
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		t.Errorf("go test cmd execution failed: %s", err)
		outputCompileErrors(buf, out)
		fmt.Println()
	}
}

func outputCompileErrors(buf *bytes.Buffer, outFolder string) {
	type errObj struct {
		errMsg     string
		fileName   string
		lineNumber int
	}

	var errObjects []errObj
	lineBuf := &bytes.Buffer{}

	bufLines := bytes.Split(buf.Bytes(), []byte{'\n'})
	for i := 0; i < len(bufLines); i++ {
		lineBuf.Reset()
		if !bytes.Contains(bufLines[i], []byte(".go:")) {
			continue
		}

		fmt.Fprintf(lineBuf, "%s\n", bufLines[i])

		splits := bytes.Split(bufLines[i], []byte{':'})
		lineNum, err := strconv.Atoi(string(splits[1]))
		if err != nil {
			panic(fmt.Sprintf("Cant convert line number to int: %s", bufLines[i]))
		}

		eObj := errObj{
			fileName:   string(splits[0]),
			lineNumber: lineNum,
		}

		for y := i; y < len(bufLines); y++ {
			if !rgxHasSpaces.Match(bufLines[y]) {
				break
			}
			fmt.Fprintf(lineBuf, "%s\n", bufLines[y])
			i++
		}

		eObj.errMsg = lineBuf.String()

		errObjects = append(errObjects, eObj)
	}

	for _, eObj := range errObjects {
		fmt.Printf("-----------------\n")
		fmt.Println(eObj.errMsg)

		filePath := filepath.Join(outFolder, eObj.fileName)
		fh, err := os.Open(filePath)
		if err != nil {
			panic(fmt.Sprintf("Cant open the file: %#v", eObj))
		}

		scanner := bufio.NewScanner(fh)
		throwaway := eObj.lineNumber - 5
		for throwaway > 0 && scanner.Scan() {
			throwaway--
		}

		for i := 0; i < 6; i++ {
			if scanner.Scan() {
				b := scanner.Bytes()
				if len(b) != 0 {
					fmt.Printf("%s\n", b)
				} else {
					i--
				}
			}
		}

		fh.Close()
	}
}

func TestProcessTypeReplacements(t *testing.T) {
	s := new(State[any])
	s.Config = &Config{}
	domainStr := "a_domain"
	s.tables = []drivers.Table{
		{
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: false,
				},
				{
					Name:     "name",
					Type:     "null.String",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: true,
				},
				{
					Name:       "domain",
					Type:       "int",
					DBType:     "numeric",
					Default:    "some db nonsense",
					DomainName: domainStr,
					Nullable:   false,
				},
			},
		},
		{
			Key: "named_table",
			Columns: []drivers.Column{
				{
					Name:     "id",
					Type:     "int",
					DBType:   "serial",
					Default:  "some db nonsense",
					Nullable: false,
				},
			},
		},
	}

	s.Config.Replacements = []Replace{
		{
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: drivers.Column{
				Type:    "excellent.Type",
				Imports: []string{`"rock.com/excellent"`},
			},
		},
		{
			Tables: []string{"named_table"},
			Match: drivers.Column{
				DBType: "serial",
			},
			Replace: drivers.Column{
				Type:    "excellent.NamedType",
				Imports: []string{`"rock.com/excellent-name"`},
			},
		},
		{
			Match: drivers.Column{
				Type:     "null.String",
				Nullable: true,
			},
			Replace: drivers.Column{
				Type:    "int",
				Imports: []string{`"context"`},
			},
		},
		{
			Match: drivers.Column{
				DomainName: domainStr,
			},
			Replace: drivers.Column{
				Type:    "big.Int",
				Imports: []string{`"math/big"`},
			},
		},
	}

	s.processTypeReplacements()

	if typ := s.tables[0].Columns[0].Type; typ != "excellent.Type" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[0].Imports[0]; i != `"rock.com/excellent"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[0].Columns[1].Type; typ != "int" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[1].Imports[0]; i != `"context"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[0].Columns[2].Type; typ != "big.Int" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[0].Columns[2].Imports[0]; i != `"math/big"` {
		t.Error("imports were not adjusted")
	}

	if typ := s.tables[1].Columns[0].Type; typ != "excellent.NamedType" {
		t.Error("type was wrong:", typ)
	}
	if i := s.tables[1].Columns[0].Imports[0]; i != `"rock.com/excellent-name"` {
		t.Error("imports were not adjusted")
	}
}
