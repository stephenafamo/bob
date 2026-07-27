package ctest

import (
	"testing"

	tc "github.com/testcontainers/testcontainers-go"
)

func Cleanup(t *testing.T, container tc.Container) {
	t.Helper()
	t.Cleanup(func() {
		if err := tc.TerminateContainer(container); err != nil {
			t.Logf("failed to terminate container: %v", err)
		}
	})
}
