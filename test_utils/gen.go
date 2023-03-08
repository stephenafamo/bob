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

var rgxHasSpaces = regexp.MustCompile(`^\s+`)

type driverWrapper[T any] struct {
	drivers.Interface[T]
	info            *drivers.DBInfo[T]
	overwriteGolden bool
	goldenFile      string
}

func (d *driverWrapper[T]) Assemble(context.Context) (*drivers.DBInfo[T], error) {
	return d.info, nil
}

func (d *driverWrapper[T]) TestAssemble(t *testing.T) {
	t.Helper()

	var err error
	d.info, err = d.Interface.Assemble(context.Background())
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
		if err = os.WriteFile(d.goldenFile, got, 0o664); err != nil {
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
	Driver          drivers.Interface[T]
	OverwriteGolden bool
	GoldenFile      string
}

func TestDriver[T any](t *testing.T, config DriverTestConfig[T]) {
	t.Helper()

	d := &driverWrapper[T]{
		Interface:       config.Driver,
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

	outputs := helpers.DefaultOutputs(d.Destination(), d.PackageName(), false, config.Templates)
	t.Run("generate", func(t *testing.T) {
		state := &gen.State[T]{Outputs: outputs}
		testDriver[T](t, state, d, config.Root, false)
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

	t.Run("generate with aliases", func(t *testing.T) {
		state := &gen.State[T]{
			Config:  gen.Config{Aliases: aliases, Wipe: true},
			Outputs: outputs,
		}
		testDriver[T](t, state, d, config.Root, true)
	})
}

func testDriver[T any](t *testing.T, state *gen.State[T], d drivers.Interface[T], root string, skipInit bool) {
	t.Helper()

	module := "github.com/stephenafamo/bob/orm/bob-gen-test"
	buf := &bytes.Buffer{}

	cmd := exec.Command("go", "env", "GOMOD")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env GOMOD cmd execution failed: %s", err)
	}

	goModFilePath := strings.TrimSpace(string(output))

	if string(goModFilePath) == os.DevNull {
		t.Fatalf("go env GOMOD cmd execution failed: %s", "not in a go module")
	}

	if !skipInit {
		cmd = exec.Command("go", "mod", "init", module)
		cmd.Dir = root
		cmd.Stderr = buf

		if err = cmd.Run(); err != nil {
			outputCompileErrors(buf, root)
			fmt.Println()
			t.Fatalf("go mod init cmd execution failed: %s", err)
		}
	}

	//nolint:gosec
	cmd = exec.Command("go", "mod", "edit", fmt.Sprintf("-replace=github.com/stephenafamo/bob=%s", filepath.Dir(string(goModFilePath))))
	cmd.Dir = root
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		outputCompileErrors(buf, root)
		fmt.Println()
		t.Fatalf("go mod edit cmd execution failed: %s", err)
	}

	if err = state.Run(context.Background(), d); err != nil {
		t.Fatalf("Unable to execute State.Run: %s", err)
	}

	// From go1.16 dependencies are not auto downloaded
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = root
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		outputCompileErrors(buf, root)
		fmt.Println()
		t.Fatalf("go mod tidy cmd execution failed: %s", err)
	}

	cmd = exec.Command("go", "test", "-run", "xxxxxxx", "./...")
	cmd.Dir = root
	cmd.Stderr = buf

	if err = cmd.Run(); err != nil {
		outputCompileErrors(buf, root)
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
