package testgen

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
	"sync"
	"testing"

	"github.com/nsf/jsondiff"
	"github.com/stephenafamo/bob/gen"
	helpers "github.com/stephenafamo/bob/gen/bobgen-helpers"
	"github.com/stephenafamo/bob/gen/drivers"
)

const module = "github.com/stephenafamo/bob/orm/bob-gen-test"

var rgxHasSpaces = regexp.MustCompile(`^\s+`)

type driverWrapper[T, C, I any] struct {
	drivers.Interface[T, C, I]
	info            *drivers.DBInfo[T, C, I]
	infoErr         error
	overwriteGolden bool
	goldenFile      string
	goldenFileMod   func([]byte) []byte
	once            sync.Once
}

func (d *driverWrapper[T, C, I]) Assemble(context.Context) (*drivers.DBInfo[T, C, I], error) {
	d.once.Do(func() {
		d.info, d.infoErr = d.Interface.Assemble(context.Background())
	})

	return d.info, d.infoErr
}

func (d *driverWrapper[T, C, I]) TestAssemble(t *testing.T) {
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

	if d.goldenFileMod != nil {
		want = d.goldenFileMod(want)
	}

	opts := jsondiff.DefaultConsoleOptions()
	opts.SkipMatches = true
	_, s := jsondiff.Compare(want, got, &opts)
	if s != "" {
		t.Fatal(s)
	}
}

type DriverTestConfig[T, C, I any] struct {
	Root            string
	Templates       *helpers.Templates
	OverwriteGolden bool
	GoldenFile      string
	GoldenFileMod   func([]byte) []byte
	GetDriver       func() drivers.Interface[T, C, I]
}

type AssembleTestConfig[T, C, I any] struct {
	Templates       *helpers.Templates
	OverwriteGolden bool
	GoldenFile      string
	GoldenFileMod   func([]byte) []byte
	GetDriver       func() drivers.Interface[T, C, I]
}

func TestAssemble[T, C, I any](t *testing.T, config AssembleTestConfig[T, C, I]) {
	t.Helper()

	d := &driverWrapper[T, C, I]{
		Interface:       config.GetDriver(),
		overwriteGolden: config.OverwriteGolden,
		goldenFile:      config.GoldenFile,
		goldenFileMod:   config.GoldenFileMod,
	}

	t.Run("assemble", func(t *testing.T) {
		d.TestAssemble(t)
	})
}

func TestDriver[T, C, I any](t *testing.T, config DriverTestConfig[T, C, I]) {
	t.Helper()

	var aliases drivers.Aliases

	d := &driverWrapper[T, C, I]{
		Interface:       config.GetDriver(),
		overwriteGolden: config.OverwriteGolden,
		goldenFile:      config.GoldenFile,
		goldenFileMod:   config.GoldenFileMod,
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

	defaultFolder := filepath.Join(config.Root, "default")
	aliasesFolder := filepath.Join(config.Root, "aliases")

	t.Run("generate", func(t *testing.T) {
		err := os.Mkdir(defaultFolder, os.ModePerm)
		if err != nil {
			t.Fatalf("unable to create default folder: %s", err)
		}

		testDriver(
			t, defaultFolder, config.Templates,
			gen.Config[C]{}, d, goModFilePath,
			&aliasPlugin[T, C, I]{},
			queryPathPlugin[T, C, I]{
				outputPath:   defaultFolder,
				trimPrefixes: []string{defaultFolder, aliasesFolder},
			},
			templatePlugin[C]{},
		)
	})

	t.Run("generate with aliases", func(t *testing.T) {
		err = os.Mkdir(aliasesFolder, os.ModePerm)
		if err != nil {
			t.Fatalf("unable to create aliases folder: %s", err)
		}

		testDriver(
			t, aliasesFolder, config.Templates,
			gen.Config[C]{Aliases: aliases}, d, goModFilePath,
			&aliasPlugin[T, C, I]{},
			queryPathPlugin[T, C, I]{
				outputPath:   aliasesFolder,
				trimPrefixes: []string{defaultFolder, aliasesFolder},
			},
			templatePlugin[C]{},
		)
	})
}

func testDriver[T, C, I any](t *testing.T, dst string, tpls *helpers.Templates, config gen.Config[C], d drivers.Interface[T, C, I], modPath string, plugins ...gen.Plugin) {
	t.Helper()
	buf := &bytes.Buffer{}

	cmd := exec.Command("go", "mod", "init", module)
	cmd.Dir = dst
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go mod init cmd execution failed: %s", err)
	}

	//nolint:gosec
	cmd = exec.Command("go", "mod", "edit", fmt.Sprintf("-replace=github.com/stephenafamo/bob=%s", filepath.Dir(modPath)))
	cmd.Dir = dst
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go mod edit cmd execution failed: %s", err)
	}

	outputs := helpers.DefaultOutputs(dst, "models", false, tpls)

	if err := gen.Run(
		context.Background(),
		&gen.State[C]{Config: config, Outputs: outputs},
		d, plugins...,
	); err != nil {
		t.Fatalf("Unable to execute State.Run: %s", err)
	}

	// From go1.16 dependencies are not auto downloaded
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Dir = dst
	cmd.Stdout = buf
	cmd.Stderr = buf

	if err := cmd.Run(); err != nil {
		outputCompileErrors(buf, dst)
		fmt.Println()
		t.Fatalf("go mod tidy cmd execution failed: %s", err)
	}

	cmd = exec.Command("go", "test", "-v", "./...")
	cmd.Dir = dst
	cmd.Stdout = buf
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
