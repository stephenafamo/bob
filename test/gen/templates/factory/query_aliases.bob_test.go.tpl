{{- $hasVideos := has "videos" $.TableNames -}}
{{- $hasTags := has "tags" $.TableNames -}}
{{- $hasVideoTags := has "video_tags" $.TableNames -}}
{{- $hasUsers := has "users" $.TableNames -}}
{{- if and $hasVideos $hasTags $hasVideoTags $hasUsers }}
{{$.Importer.Import "context"}}
{{$.Importer.Import "regexp"}}
{{$.Importer.Import "slices"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

{{- /* SQLite assigns SelectJoins to the generic function; other dialects call it. */ -}}
{{- $joins := "models.SelectJoins" -}}
{{- if eq $.Dialect "sqlite" -}}
	{{- $joins = "models.SelectJoins()" -}}
{{- end}}

var factoryJoinAliasRE = regexp.MustCompile(`(?i)(?:LEFT|INNER|RIGHT)?\s*JOIN\s+\S+\s+AS\s+(?:"([^"]+)"|\x60([^\x60]+)\x60|([A-Za-z0-9_]+))`)

func factoryJoinAliases(sql string) []string {
	matches := factoryJoinAliasRE.FindAllStringSubmatch(sql, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		for _, g := range m[1:] {
			if g != "" {
				out = append(out, g)
				break
			}
		}
	}
	return out
}

func unquoteSQL(sql string) string {
	return strings.NewReplacer(`"`, "", "`", "").Replace(sql)
}

func assertJoinAliases(t *testing.T, sql string, want ...string) {
	t.Helper()
	if got := factoryJoinAliases(sql); !slices.Equal(got, want) {
		t.Fatalf("aliases = %v, want %v\n%s", got, want, sql)
	}
}

func runAll(t *testing.T, ctx context.Context, q interface {
	All(context.Context, bob.Executor) (models.VideoSlice, error)
}) {
	t.Helper()
	if testDB == nil {
		return
	}
	tx, err := testDB.Begin(ctx)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	defer tx.Rollback(ctx)
	if _, err := q.All(ctx, tx); err != nil {
		t.Fatalf("All: %v", err)
	}
}

func TestMultiHopJoinAliasStableSQL(t *testing.T) {
	ctx := context.Background()

	q1 := models.Videos.Query({{$joins}}.Videos.InnerJoin.Tags)
	q2 := models.Videos.Query({{$joins}}.Videos.InnerJoin.Tags)
	sql1, _, err := q1.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	sql2, _, err := q2.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if sql1 != sql2 {
		t.Fatalf("multi-hop join SQL not stable:\n%s\n---\n%s", sql1, sql2)
	}
	assertJoinAliases(t, sql1, "bob_q0", "tags")
	runAll(t, ctx, q1)
}

func TestMultiHopJoinDistinctFinalAliases(t *testing.T) {
	ctx := context.Background()

	q := models.Videos.Query(
		{{$joins}}.Videos.InnerJoin.Tags.AliasedAs("a"),
		{{$joins}}.Videos.InnerJoin.Tags.AliasedAs("b"),
	)
	sql, _, err := q.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	assertJoinAliases(t, sql, "bob_q0", "a", "bob_q1", "b")
}

func TestMultiHopJoinParentAliasedAs(t *testing.T) {
	ctx := context.Background()

	q := models.Videos.Query(
		{{$joins}}.Videos.AliasedAs("v").InnerJoin.Tags.AliasedAs("t"),
	)
	sql, _, err := q.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	assertJoinAliases(t, sql, "bob_q0", "t")
	if !strings.Contains(unquoteSQL(sql), "bob_q0.video_id = v.id") {
		t.Fatalf("first hop should join from parent alias v:\n%s", sql)
	}
}

func TestSingleHopJoinUnchanged(t *testing.T) {
	ctx := context.Background()

	q := models.Videos.Query({{$joins}}.Videos.InnerJoin.User)
	sql, _, err := q.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	assertJoinAliases(t, sql, "users")
}

func TestMixedJoinAndPreloadShareSequence(t *testing.T) {
	ctx := context.Background()

	q := models.Videos.Query(
		{{$joins}}.Videos.InnerJoin.Tags.AliasedAs("t"),
		models.Preload.Video.User(),
	)
	sql, _, err := q.Build(ctx)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	assertJoinAliases(t, sql, "bob_q0", "t", "bob_q1")
	runAll(t, ctx, q)
}
{{- end}}
