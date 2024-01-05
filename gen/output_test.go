package gen

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestWriteFile(t *testing.T) {
	// t.Parallel() cannot be used

	// set the function pointer back to its original value
	// after we modify it for the test
	saveTestHarnessWriteFile := testHarnessWriteFile
	defer func() {
		testHarnessWriteFile = saveTestHarnessWriteFile
	}()

	var output []byte
	testHarnessWriteFile = func(_ string, in []byte, _ os.FileMode) error {
		output = in
		return nil
	}

	buf := &bytes.Buffer{}
	writePackageName(buf, "pkg")
	fmt.Fprintf(buf, "func hello() {}\n\n\nfunc world() {\nreturn\n}\n\n\n\n")

	if err := writeFile("", "", buf, "v1"); err != nil {
		t.Error(err)
	}

	if string(output) != "package pkg\n\nfunc hello() {}\n\nfunc world() {\n\treturn\n}\n" {
		t.Errorf("Wrong output: %q", output)
	}
}

func TestFormatBuffer(t *testing.T) {
	t.Parallel()

	buf := &bytes.Buffer{}

	fmt.Fprintf(buf, "package pkg\n\nfunc() {a}\n")

	// Only test error case - happy case is taken care of by template test
	_, err := formatBuffer(buf, "v1")
	if err == nil {
		t.Error("want an error")
	}

	if txt := err.Error(); !strings.Contains(txt, ">>>> func() {a}") {
		t.Error("got:\n", txt)
	}
}

func TestOutputFilenameParts(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Filename string

		Normalized  string
		IsSingleton bool
		IsGo        bool
		UsePkg      bool
	}{
		{"00_struct.go.tpl", "struct.go", false, true, true},
		{"singleton/00_struct.go.tpl", "struct.go", true, true, true},
		{"notpkg/00_struct.go.tpl", "notpkg/struct.go", false, true, false},
		{"js/singleton/00_struct.js.tpl", "js/struct.js", true, false, false},
		{"js/00_struct.js.tpl", "js/struct.js", false, false, false},
	}

	for i, test := range tests {
		normalized, isSingleton, isGo, usePkg := outputFilenameParts(test.Filename)

		if normalized != test.Normalized {
			t.Errorf("%d) normalized wrong, want: %s, got: %s", i, test.Normalized, normalized)
		}
		if isSingleton != test.IsSingleton {
			t.Errorf("%d) isSingleton wrong, want: %t, got: %t", i, test.IsSingleton, isSingleton)
		}
		if isGo != test.IsGo {
			t.Errorf("%d) isGo wrong, want: %t, got: %t", i, test.IsGo, isGo)
		}
		if usePkg != test.UsePkg {
			t.Errorf("%d) usePkg wrong, want: %t, got: %t", i, test.UsePkg, usePkg)
		}
	}
}

func TestGetOutputFilename(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		SchemaName string
		TableName  string
		IsTest     bool
		IsGo       bool
		Expected   string
	}{
		"regular": {
			TableName: "hello",
			IsTest:    false,
			IsGo:      true,
			Expected:  "hello",
		},
		"with schema": {
			SchemaName: "schema",
			TableName:  "hello",
			IsTest:     false,
			IsGo:       true,
			Expected:   "hello.schema",
		},
		"begins with underscore": {
			TableName: "_hello",
			IsTest:    false,
			IsGo:      true,
			Expected:  "und_hello",
		},
		"ends with _test": {
			TableName: "hello_test",
			IsTest:    false,
			IsGo:      true,
			Expected:  "hello_test_model",
		},
		"ends with _js": {
			TableName: "hello_js",
			IsTest:    false,
			IsGo:      true,
			Expected:  "hello_js_model",
		},
		"ends with _windows": {
			TableName: "hello_windows",
			IsTest:    false,
			IsGo:      true,
			Expected:  "hello_windows_model",
		},
		"ends with _arm64": {
			TableName: "hello_arm64",
			IsTest:    false,
			IsGo:      true,
			Expected:  "hello_arm64_model",
		},
		"non-go ends with _arm64": {
			TableName: "hello_arm64",
			IsTest:    false,
			IsGo:      false,
			Expected:  "hello_arm64",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			notTest := getOutputFilename(tc.SchemaName, tc.TableName, false, tc.IsGo)
			if diff := cmp.Diff(tc.Expected, notTest); diff != "" {
				t.Fatalf(diff)
			}

			isTest := getOutputFilename(tc.SchemaName, tc.TableName, true, tc.IsGo)
			if diff := cmp.Diff(tc.Expected+"_test", isTest); diff != "" {
				t.Fatalf(diff)
			}
		})
	}
}
