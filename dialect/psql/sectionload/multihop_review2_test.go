package sectionload

// Tests for the public function CarriedKeyName. CarriedKeyPrefix is covered
// indirectly by the existing "_sl_k0" tests (SQL string checks), but those
// would not notice CarriedKeyName(i) itself returning a wrong name.
// TestCarriedKeyName verifies it directly.

import (
	"strings"
	"testing"
)

// TestCarriedKeyName pins CarriedKeyName's contract: for every i, it
// returns CarriedKeyPrefix followed by the decimal digits of i, with no
// separator — the exact "_sl_k0", "_sl_k1", ... sequence Edge.Carried's
// doc comment (parts.go) promises, and the same value the generated
// template (01_edges.go.tpl) now calls this function to produce instead
// of duplicating the literal.
func TestCarriedKeyName(t *testing.T) {
	tests := map[string]struct {
		i    int
		want string
	}{
		"zero":                     {0, "_sl_k0"},
		"single digit":             {1, "_sl_k1"},
		"multi-digit (two digits)": {10, "_sl_k10"},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := CarriedKeyName(tc.i)
			if got != tc.want {
				t.Errorf("CarriedKeyName(%d) = %q, want %q", tc.i, got, tc.want)
			}
			// CarriedKeyName must always be built from CarriedKeyPrefix (the
			// single source of truth parentexpr.go's reservedNameRe also
			// derives from) — a hardcoded "_sl_k" duplicate inside
			// CarriedKeyName that happened to match today's constant value
			// would still pass the exact-string checks above but would
			// silently desync the moment CarriedKeyPrefix changed.
			if !strings.HasPrefix(got, CarriedKeyPrefix) {
				t.Errorf("CarriedKeyName(%d) = %q, want prefix %q (CarriedKeyPrefix)", tc.i, got, CarriedKeyPrefix)
			}
		})
	}
}
