{{- $hasAnyRel := false -}}
{{- range $table := .Tables -}}
{{- if $.Relationships.Get $table.Key -}}{{- $hasAnyRel = true -}}{{- end -}}
{{- end -}}
{{- if $hasAnyRel -}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}

{{range $table := .Tables -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- if $rels -}}
{{- $tAlias := $.Aliases.Table $table.Key}}
// Test{{$tAlias.UpSingular}}HasRelationsEmitExists verifies that every generated
// Has{Rel} helper produces a correlated EXISTS subquery (semi-join) rather than
// an INNER JOIN, so the parent rows are never multiplied.
func Test{{$tAlias.UpSingular}}HasRelationsEmitExists(t *testing.T) {
	ctx := context.Background()
	{{range $rel := $rels -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name}}
	t.Run("{{$relAlias}}", func(t *testing.T) {
		q := {{$.Dialect}}.Select(
			sm.From({{$tAlias.UpPlural}}.NameExpr()),
			SelectWhere.{{$tAlias.UpPlural}}.R.Has{{$relAlias}}(),
		)
		sql, _, err := bob.Build(ctx, q)
		if err != nil {
			t.Fatalf("Has{{$relAlias}}: build error: %v", err)
		}
		if !strings.Contains(sql, "EXISTS") {
			t.Errorf("Has{{$relAlias}}: expected EXISTS in query, got: %s", sql)
		}
	})
	{{end -}}
}
{{end -}}
{{end -}}
{{- end -}}
