package bob_test

import (
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	mysqldialect "github.com/stephenafamo/bob/dialect/mysql/dialect"
	psqldialect "github.com/stephenafamo/bob/dialect/psql/dialect"
	sqlitedialect "github.com/stephenafamo/bob/dialect/sqlite/dialect"
)

func TestWriteQuotedEscapesIdentifierQuotes(t *testing.T) {
	tests := []struct {
		name    string
		dialect bob.Dialect
		input   string
		want    string
	}{
		{name: "postgres", dialect: psqldialect.Dialect, input: `a"b`, want: `"a""b"`},
		{name: "mysql", dialect: mysqldialect.Dialect, input: "a`b", want: "`a``b`"},
		{name: "sqlite", dialect: sqlitedialect.Dialect, input: `a"b`, want: `"a""b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got strings.Builder
			tt.dialect.WriteQuoted(&got, tt.input)
			if got.String() != tt.want {
				t.Errorf("WriteQuoted(%q) = %q, want %q", tt.input, got.String(), tt.want)
			}
		})
	}
}
