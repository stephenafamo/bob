package psql_test

import (
	"context"
	"strings"
	"testing"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/dialect/psql"
	"github.com/stephenafamo/bob/dialect/psql/sm"
)

func TestSelectQuerySetLimitIfUnset(t *testing.T) {
	ctx := context.Background()

	t.Run("injects limit when unset", func(t *testing.T) {
		q := psql.Select(sm.Columns("id"), sm.From("users"))
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

	t.Run("preserves existing LIMIT", func(t *testing.T) {
		q := psql.Select(sm.Columns("id"), sm.From("users"), sm.Limit(5))
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

	t.Run("preserves existing FETCH", func(t *testing.T) {
		q := psql.Select(sm.Columns("id"), sm.From("users"), sm.Fetch(5, false))
		var l bob.Limiter = q
		l.SetLimitIfUnset(1)

		sql, _, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		if strings.Contains(sql, "LIMIT") {
			t.Fatalf("did not expect LIMIT (FETCH already set), got: %s", sql)
		}
	})

	t.Run("union sets CombinedLimit", func(t *testing.T) {
		inner := psql.Select(sm.Columns("id"), sm.From("orders"))
		q := psql.Select(sm.Columns("id"), sm.From("users"), sm.Union(inner))
		var l bob.Limiter = q
		l.SetLimitIfUnset(1)

		sql, _, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("build: %v", err)
		}
		// LIMIT must appear after the UNION, not inside the first SELECT's parens.
		idx := strings.LastIndex(sql, "LIMIT 1")
		unionIdx := strings.Index(sql, "UNION")
		if idx == -1 {
			t.Fatalf("expected LIMIT 1, got: %s", sql)
		}
		if unionIdx == -1 || idx < unionIdx {
			t.Fatalf("expected LIMIT 1 after UNION, got: %s", sql)
		}
	})
}
