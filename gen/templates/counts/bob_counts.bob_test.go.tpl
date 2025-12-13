{{- $hasAnyToMany := false -}}
{{- range $table := .Tables -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- range $rel := $rels -}}
	{{- if $rel.IsToMany -}}{{- $hasAnyToMany = true -}}{{- end -}}
{{- end -}}
{{- end -}}

{{- if $hasAnyToMany -}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "context"}}

{{range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- $hasToMany := false -}}
{{- range $rel := $rels -}}
	{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
{{- end -}}
{{- if $hasToMany}}
// Test that {{$tAlias.UpSingular}} has a C field with count pointers for to-many relationships
func Test{{$tAlias.UpSingular}}CountStruct(t *testing.T) {
	var m {{$tAlias.UpSingular}}
	_ = m.C // Verify C field exists

	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name}}
	// Verify {{$relAlias}} count field exists and is *int64
	var _ *int64 = m.C.{{$relAlias}}
	{{end}}
}

// Test that {{$tAlias.UpSingular}} has LoadCount methods for to-many relationships
func Test{{$tAlias.UpSingular}}LoadCountMethods(t *testing.T) {
	var m *{{$tAlias.UpSingular}}
	var ms {{$tAlias.UpSingular}}Slice
	ctx := context.Background()

	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name}}
	// Verify LoadCount{{$relAlias}} method exists on single model
	_ = m.LoadCount{{$relAlias}}(ctx, nil)

	// Verify LoadCount{{$relAlias}} method exists on slice
	_ = ms.LoadCount{{$relAlias}}(ctx, nil)
	{{end}}
}

// Test that ThenLoadCount has {{$tAlias.UpSingular}} with methods for to-many relationships
func TestThenLoadCount{{$tAlias.UpSingular}}(t *testing.T) {
	_ = ThenLoadCount.{{$tAlias.UpSingular}}

	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name}}
	// Verify {{$relAlias}} loader exists
	_ = ThenLoadCount.{{$tAlias.UpSingular}}.{{$relAlias}}
	{{end}}
}

// Test that PreloadCount has {{$tAlias.UpSingular}} with methods for to-many relationships
func TestPreloadCount{{$tAlias.UpSingular}}(t *testing.T) {
	_ = PreloadCount.{{$tAlias.UpSingular}}

	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name}}
	// Verify {{$relAlias}} preloader exists and returns a Preloader
	_ = PreloadCount.{{$tAlias.UpSingular}}.{{$relAlias}}()
	{{end}}
}

// Test that {{$tAlias.UpSingular}} has PreloadCount method
func Test{{$tAlias.UpSingular}}PreloadCountMethod(t *testing.T) {
	var m *{{$tAlias.UpSingular}}
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name}}
	_ = m.PreloadCount("{{$relAlias}}", 0)
	{{end}}
}
{{end}}
{{end}}
{{end}}
