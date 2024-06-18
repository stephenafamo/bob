package bob

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

var reArgs = regexp.MustCompile(`\n\d+\:`)

func TestDebugExecutorDefaultWriter(t *testing.T) {
	d, ok := DebugToWriter(NoopExecutor{}, nil).(debugExecutor)
	if !ok {
		t.Fatal("DebugToWriter does not return an instance of debugExecutor")
	}

	writer, ok := d.printer.(writerPrinter)
	if !ok {
		t.Fatal("printer for debugExecutor is not a writerPrinter")
	}

	debugFile, ok := writer.Writer.(*os.File)
	if !ok {
		t.Fatal("writer for writerPrinter is not an *os.File")
	}

	if debugFile != os.Stdout {
		t.Fatal("writer for debugExecutor is not os.Stdout")
	}
}

func TestDebugExecutorDefaultPrinter(t *testing.T) {
	d, ok := DebugToPrinter(NoopExecutor{}, nil).(debugExecutor)
	if !ok {
		t.Fatal("DebugExecutor does not return an instance of debugExecutor")
	}

	writer, ok := d.printer.(writerPrinter)
	if !ok {
		t.Fatal("printer for debugExecutor is not a writerPrinter")
	}

	debugFile, ok := writer.Writer.(*os.File)
	if !ok {
		t.Fatal("writer for writerPrinter is not an *os.File")
	}

	if debugFile != os.Stdout {
		t.Fatal("writer for debugExecutor is not os.Stdout")
	}
}

func TestDebugExecutor(t *testing.T) {
	t.Run("QueryContext", func(t *testing.T) {
		testDebugExecutor(t, func(exec Executor, s string, a ...any) error {
			_, err := exec.QueryContext(context.Background(), s, a...)
			if err != nil {
				t.Fatal("error running QueryContext")
			}

			return err
		})
	})

	t.Run("ExecContext", func(t *testing.T) {
		testDebugExecutor(t, func(exec Executor, s string, a ...any) error {
			_, err := exec.ExecContext(context.Background(), s, a...)
			if err != nil {
				t.Fatal("error running QueryContext")
			}

			return err
		})
	})
}

func testDebugExecutor(t *testing.T, f func(Executor, string, ...any) error) {
	t.Helper()

	dest := &bytes.Buffer{}
	exec := DebugToWriter(NoopExecutor{}, dest)

	sql := "A QUERY"
	args := []any{"arg1", "arg2", "3rd arg"}

	err := f(exec, sql, args...)
	if err != nil {
		t.Fatal(err)
	}

	debugsql, debugArgsStr, found := strings.Cut(dest.String(), "\n0:")
	if !found {
		t.Fatalf("arg delimiter not found in\n%s", dest.String())
	}

	if strings.TrimSpace(debugsql) != sql {
		t.Fatalf("wrong debug sql.\nExpected: %s\nGot: %s", sql, strings.TrimSpace(debugsql))
	}

	var debugArgs []string //nolint:prealloc
	for _, s := range reArgs.Split("\n0:"+debugArgsStr, -1) {
		s := strings.TrimSpace(s)
		if s == "" {
			continue
		}

		debugArgs = append(debugArgs, s)
	}

	if len(debugArgs) != len(args) {
		t.Fatalf("wrong length of debug args.\nExpected: %d\nGot: %d\n\n%s", len(args), len(debugArgs), debugArgs)
	}

	for i := range args {
		argStr := strings.TrimSpace(fmt.Sprint(args[i]))
		debugStr := strings.TrimSpace(debugArgs[i])
		if "string: "+argStr != debugStr {
			t.Fatalf("wrong debug arg %d.\nExpected: %s\nGot: %s", i, argStr, debugStr)
		}
	}
}
