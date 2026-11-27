package dialect

import (
	"strings"
	"testing"
)

func TestWriteQuoted(t *testing.T) {
	tests := map[string]string{
		"id":      `"id"`,
		`we"ird`:  `"we""ird"`,
		`"`:       `""""`,
		`a"; --b`: `"a""; --b"`,
	}

	for in, want := range tests {
		var b strings.Builder
		Dialect.WriteQuoted(&b, in)
		if got := b.String(); got != want {
			t.Errorf("WriteQuoted(%q) = %s, want %s", in, got, want)
		}
	}
}
