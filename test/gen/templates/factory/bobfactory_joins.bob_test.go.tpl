{{- if and (has "videos" $.TableNames) (has "video_tags" $.TableNames) (has "tags" $.TableNames) -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

// Building the same query again must give the same SQL, also when it joins
// through another table, so that drivers can reuse its prepared statement
func TestJoinThroughTableIsStable(t *testing.T) {
	build := func() string {
		q := models.Videos.Query(models.SelectJoins{{if eq $.Dialect "sqlite"}}(){{end}}.Videos.InnerJoin.Tags)
		sql, _, err := bob.Build(context.Background(), q)
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return sql
	}

	if first, second := build(), build(); first != second {
		t.Fatalf("SQL differs between builds:\n%s\n%s", first, second)
	}
}
{{- end}}
