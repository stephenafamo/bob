package helpers

import (
	"testing"
)

func TestSplitStatements(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{
			name:  "single statement",
			input: "CREATE TABLE foo (id int);",
			want:  []string{"CREATE TABLE foo (id int);"},
		},
		{
			name:  "multiple statements",
			input: "CREATE TABLE foo (id int);\nCREATE TABLE bar (id int);",
			want:  []string{"CREATE TABLE foo (id int);", "CREATE TABLE bar (id int);"},
		},
		{
			name:  "dollar-quoted block",
			input: "DO $$ BEGIN RAISE NOTICE 'hello;world'; END $$;\nSELECT 1;",
			want:  []string{"DO $$ BEGIN RAISE NOTICE 'hello;world'; END $$;", "SELECT 1;"},
		},
		{
			name:  "tagged dollar-quote",
			input: "DO $fn$ BEGIN PERFORM 1; END $fn$;\nSELECT 2;",
			want:  []string{"DO $fn$ BEGIN PERFORM 1; END $fn$;", "SELECT 2;"},
		},
		{
			name:  "single-quoted string with semicolon",
			input: "SELECT 'a;b';\nSELECT 1;",
			want:  []string{"SELECT 'a;b';", "SELECT 1;"},
		},
		{
			name:  "escaped single quote",
			input: "SELECT 'it''s';\nSELECT 1;",
			want:  []string{"SELECT 'it''s';", "SELECT 1;"},
		},
		{
			name:  "line comment with semicolon",
			input: "-- this is a comment; not a separator\nSELECT 1;",
			want:  []string{"-- this is a comment; not a separator\nSELECT 1;"},
		},
		{
			name:  "block comment with semicolon",
			input: "/* comment; here */\nSELECT 1;",
			want:  []string{"/* comment; here */\nSELECT 1;"},
		},
		{
			name: "real migration with CONCURRENTLY and DO block",
			input: `-- atlas:txmode none

CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_foo ON bar (id);

REFRESH MATERIALIZED VIEW CONCURRENTLY bar;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_available_extensions
        WHERE name = 'pg_cron'
    ) THEN
    UPDATE cron.job
    SET active = FALSE
    WHERE jobname = 'update_metrics';
END IF;
END
$$;
`,
			want: []string{
				"-- atlas:txmode none\n\nCREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_foo ON bar (id);",
				"REFRESH MATERIALIZED VIEW CONCURRENTLY bar;",
				`DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_available_extensions
        WHERE name = 'pg_cron'
    ) THEN
    UPDATE cron.job
    SET active = FALSE
    WHERE jobname = 'update_metrics';
END IF;
END
$$;`,
			},
		},
		{
			name:  "trailing content without semicolon",
			input: "SELECT 1;\nSELECT 2",
			want:  []string{"SELECT 1;", "SELECT 2"},
		},
		{
			name:  "empty input",
			input: "",
			want:  nil,
		},
		{
			name:  "whitespace only",
			input: "   \n\n  ",
			want:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitStatements(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d statements, want %d\ngot:  %q\nwant: %q", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("statement %d:\ngot:  %q\nwant: %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
