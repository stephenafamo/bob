package parser

import (
	"errors"
	"strings"
	"testing"
)

func TestIsReturningWithParseError(t *testing.T) {
	t.Parallel()

	err := errors.New("syntax error at or near \"WITH\"")

	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{
			name: "returning with clause and with syntax error",
			sql:  `UPDATE users SET name = $1 RETURNING WITH (OLD AS o, NEW AS n) o.*, n.*`,
			want: true,
		},
		{
			name: "query without returning with clause",
			sql:  `UPDATE users SET name = $1 RETURNING id`,
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isReturningWithParseError(tc.sql, err); got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got)
			}
		})
	}
}

func TestWrapParseErrorWithReturningWithHint(t *testing.T) {
	t.Parallel()

	baseErr := errors.New("syntax error at or near \"WITH\"")
	wrapped := wrapParseErrorWithReturningWithHint(
		"parse single",
		`DELETE FROM users WHERE id = $1 RETURNING WITH (OLD AS before) before.*`,
		baseErr,
	)

	msg := wrapped.Error()
	if !strings.Contains(msg, "RETURNING WITH (OLD|NEW AS ...)") {
		t.Fatalf("expected RETURNING WITH hint in error, got: %s", msg)
	}

	if !strings.Contains(msg, "parse single") {
		t.Fatalf("expected stage prefix in error, got: %s", msg)
	}
}
