package bob

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestHooks(t *testing.T) {
	type skipKey struct{}
	var H Hooks[*string, skipKey]

	// Test Adding Hooks
	for i := 0; i < 5; i++ {
		initial := len(H.hooks)
		f := func(ctx context.Context, _ Executor, s *string) (context.Context, error) {
			*s = *s + fmt.Sprintf("%d", initial+1)
			return ctx, nil
		}
		H.AppendHooks(f)
		if len(H.hooks) != initial+1 {
			t.Fatalf("Did not add hook number %d", i+1)
		}
	}

	s := ""
	if _, err := H.RunHooks(context.Background(), nil, &s); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("12345", s); diff != "" {
		t.Fatal(diff)
	}

	// test skipping hooks
	s = ""
	if _, err := H.RunHooks(context.WithValue(context.Background(), skipKey{}, true), nil, &s); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff("", s); diff != "" {
		t.Fatal(diff)
	}
}
