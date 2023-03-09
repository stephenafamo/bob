package testutils

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
	"github.com/stretchr/testify/require"
)

const module = "github.com/stephenafamo/bob/orm/bob-gen-test"

var rgxHasSpaces = regexp.MustCompile(`^\s+`)

type driverWrapper[T any] struct {
	drivers.Interface[T]
	info            *drivers.DBInfo[T]
	overwriteGolden bool
	goldenFile      string
}

func (d *driverWrapper[T]) Assemble(context.Context) (*drivers.DBInfo[T], error) {
	var err error

	d.info, err = d.Interface.Assemble(context.Background())
	if err != nil {
		return nil, err
	}

	return d.info, nil
}

func (d *driverWrapper[T]) TestAssemble(t *testing.T) {
	t.Helper()

	_, err := d.Assemble(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	sort.Slice(d.info.Tables, func(i, j int) bool {
		return d.info.Tables[i].Key < d.info.Tables[j].Key
	})

	got, err := json.MarshalIndent(d.info, "", "\t")
	if err != nil {
		t.Fatal(err)
	}

	if d.overwriteGolden {
		if err = os.WriteFile(d.goldenFile, got, 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}

	want, err := os.ReadFile(d.goldenFile)
	if err != nil {
		t.Fatal(err)
	}

	require.JSONEq(t, string(want), string(got))
}

type DriverTestConfig[T any] struct {
	Root            string
	Templates       *helpers.Templates
	OverwriteGolden bool
	GoldenFile      string
	GetDriver       func(path string) drivers.Interface[T]
}

func TestDriver[T any](t *testing.T, config DriverTestConfig[T]) {
	t.Helper()

	defaultFolder := filepath.Join(config.Root, "default")
	err := os.Mkdir(defaultFolder, os.ModePerm)
	if err != nil {
		t.Fatalf("unable to create default folder: %s", err)
	}

	d := &driverWrapper[T]{
		Interface:       config.GetDriver(defaultFolder),
		overwriteGolden: config.OverwriteGolden,
		goldenFile:      config.GoldenFile,
	}

	t.Run("assemble", func(t *testing.T) {
		d.TestAssemble(t)
	})

	if testing.Short() {
		// skip testing generation
		t.SkipNow()
	}

	cmd := exec.Command("go", "env", "GOMOD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOMOD cmd execution failed: %s", err)
	}

	goModFilePath := strings.TrimSpace(string(output))

	if string(goModFilePath) == os.DevNull {
		t.Fatalf("go env GOMOD cmd execution failed: %s", "not in a go module")
	}

	t.Run("generate", func(t *testing.T) {
		testDriver[T](t, config.Templates, gen.Config{}, d, goModFilePath)
	})

	aliases := gen.Aliases{Tables: make(map[string]gen.TableAlias)}
	for i, table := range d.info.Tables {
		tableAlias := gen.TableAlias{
			UpPlural:     fmt.Sprintf("Alias%dThings", i),
			UpSingular:   fmt.Sprintf("Alias%dThing", i),
			DownPlural:   fmt.Sprintf("alias%dThings", i),
			DownSingular: fmt.Sprintf("alias%dThing", i),
		}

		tableAlias.Columns = make(map[string]string)
		for j, column := range table.Columns {
			tableAlias.Columns[column.Name] = fmt.Sprintf("Alias%dThingColumn%d", i, j)
		}

		tableAlias.Relationships = make(map[string]string)
		for j, rel := range table.Relationships {
			tableAlias.Relationships[rel.Name] = fmt.Sprintf("Alias%dThingRel%d", i, j)
		}

		aliases.Tables[table.Key] = tableAlias
	}

	aliasesFolder := filepath.Join(config.Root, "aliases")
	err = os.Mkdir(aliasesFolder, os.ModePerm)
	if err != nil {
		t.Fatalf("unable to create aliases folder: %s", err)
	}

	d = &driverWrapper[T]{
		Interface:       config.GetDriver(aliasesFolder),
		overwriteGolden: config.OverwriteGolden,
		goldenFile:      config.GoldenFile,
	}

	t.Run("generate with aliases", func(t *testing.T) {
		testDriver[T](t, config.Templates, gen.Config{Aliases: aliases}, d, goModFilePath)
	})
}

func testDriver[T any](t *testing.T, tpls *helpers.Templates, config gen.Config, d drivers.Interface[T], modPath string) {
	t.Helper()

	dst := d.Destination()
	buf := &bytes.Buffer{}

	cmd := exec.Command("go", "mod", "init", module)
	cmd.Dir = dst
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go mod init cmd execution failed: %s", err)
	}

	//nolint:gosec
	cmd = exec.Command("go", "mod", "edit", fmt.Sprintf("-replace=github.com/stephenafamo/bob=%s", filepath.Dir(modPath)))
	cmd.Dir = dst
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go mod edit cmd execution failed: %s", err)
	}

	outputs := helpers.DefaultOutputs(dst, d.PackageName(), false, tpls)
	if err := gen.Run(context.Background(), &gen.State{Config: config, Outputs: outputs}, d); err != nil {
		t.Fatalf("Unable to execute State.Run: %s", err)
	}

	// From go1.16 dependencies are not auto downloaded
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dst
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go mod tidy cmd execution failed: %s", err)
	}

	cmd = exec.Command("go", "test", "-run", "xxxxxxx", "./...")
	cmd.Dir = dst
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go test cmd execution failed: %s", err)
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

		splits := strings.Split(string(bufLines[i]), ":")
		lineNum, err := strconv.Atoi(string(splits[1]))
		if err != nil {
			panic(fmt.Sprintf("Cant convert line number to int: %s", bufLines[i]))
		}

		eObj := errObj{
			fileName:   strings.TrimSpace(splits[0]),
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
		fmt.Printf("\n-----------------\n")
		fmt.Print(eObj.errMsg)

		filePath := filepath.Join(outFolder, eObj.fileName)
		fh, err := os.Open(filePath)
		if err != nil {
			panic(fmt.Sprintf("Cant open the file: %#v", filePath))
		}

		scanner := bufio.NewScanner(fh)
		for i := 1; scanner.Scan(); i++ {
			if i < (eObj.lineNumber - 3) {
				continue
			}

			leading := " "
			if i == eObj.lineNumber {
				leading = "•"
			}

			fmt.Printf("%s%03d| %s\n", leading, i, scanner.Bytes())

			if i > (eObj.lineNumber + 3) {
				break
			}
		}

		fmt.Printf("-----------------\n")

		fh.Close()
	}
}
