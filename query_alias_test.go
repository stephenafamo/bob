package bob

import "testing"

func TestNextQueryAliasSequential(t *testing.T) {
	t.Parallel()

	var l Load
	got := []string{l.NextQueryAlias(), l.NextQueryAlias(), l.NextQueryAlias()}
	want := []string{"bob_q0", "bob_q1", "bob_q2"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("alias[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestNextQueryAliasIndependentLoads(t *testing.T) {
	t.Parallel()

	var a, b Load
	if a.NextQueryAlias() != "bob_q0" {
		t.Fatalf("first Load did not start at bob_q0")
	}
	if b.NextQueryAlias() != "bob_q0" {
		t.Fatalf("second Load did not start at bob_q0")
	}
}

func TestNextQueryAliasBase36Boundaries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		counter uint64
		want    string
	}{
		{35, "bob_qz"},
		{36, "bob_q10"},
	}
	for _, tt := range tests {
		l := Load{queryAliasCounter: tt.counter}
		if got := l.NextQueryAlias(); got != tt.want {
			t.Fatalf("counter %d: alias = %q, want %q", tt.counter, got, tt.want)
		}
	}
}
