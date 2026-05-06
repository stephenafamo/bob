package sqlite_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/sqlite"
	"github.com/stephenafamo/bob/dialect/sqlite/sm"
)

func TestSelectQuerySetLimitIfUnset(t *testing.T) {
	ctx := context.Background()

	t.Run("injects limit when unset", func(t *testing.T) {
		q := sqlite.Select(sm.Columns("id"), sm.From("users"))
		var l bob.Limiter = q
		l.SetLimitIfUnset(1)

		sql, _, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if !strings.Contains(sql, "LIMIT 1") {
			t.Fatalf("expected LIMIT 1 in SQL, got: %s", sql)
		}
	})

	t.Run("preserves existing limit", func(t *testing.T) {
		q := sqlite.Select(sm.Columns("id"), sm.From("users"), sm.Limit(5))
		var l bob.Limiter = q
		l.SetLimitIfUnset(1)

		sql, _, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if !strings.Contains(sql, "LIMIT 5") {
			t.Fatalf("expected LIMIT 5 (preserved), got: %s", sql)
		}
		if strings.Contains(sql, "LIMIT 1") {
			t.Fatalf("did not expect LIMIT 1, got: %s", sql)
		}
	})
}
