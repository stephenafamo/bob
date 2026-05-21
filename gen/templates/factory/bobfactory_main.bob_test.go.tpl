{{- $needsModels := false -}}
{{- range $table := .Tables -}}
  {{- range $rel := $.Relationships.Get $table.Key -}}
    {{- $bridgeRels := $.Tables.NeededBridgeRels $rel -}}
    {{- if and $rel.IsToMany (ne $rel.Foreign $table.Key) (not $bridgeRels) (eq (len $rel.Sides) 1) -}}
      {{- $needsModels = true -}}
    {{- end -}}
  {{- end -}}
{{- end -}}
{{- $.Importer.Import "context" -}}
{{- $.Importer.Import "testing" -}}
{{- if $needsModels -}}
{{- $.Importer.Import "models" (index $.OutputPackages "models") -}}
{{- end -}}

{{range $table := .Tables}}{{if not $table.Constraints.Primary}}{{continue}}{{end}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func TestCreate{{$tAlias.UpSingular}}(t *testing.T) {
  if testDB == nil {
    t.Skip("skipping test, no DSN provided")
  }

  ctx, cancel := context.WithCancel(t.Context())
  t.Cleanup(cancel)

  tx, err := testDB.Begin(ctx)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  defer func() {
    if err := tx.Rollback(ctx); err != nil {
      t.Fatalf("Error rolling back transaction: %v", err)
    }
  }()

  if _, err := New().New{{$tAlias.UpSingular}}WithContext(ctx).Create(ctx, tx); err != nil {
    t.Fatalf("Error creating {{$tAlias.UpSingular}}: %v", err)
  }
}

{{range $rel := $.Relationships.Get $table.Key -}}
{{- if not .IsToMany -}}{{continue}}{{end -}}
{{- if eq .Foreign $table.Key -}}{{continue}}{{end -}}
{{- if $.Tables.NeededBridgeRels . -}}{{continue}}{{end -}}
{{- if gt (len .Sides) 1 -}}{{continue}}{{end -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
func TestCreate{{$tAlias.UpSingular}}With{{$relAlias}}DoesNotDuplicateParent(t *testing.T) {
  if testDB == nil {
    t.Skip("skipping test, no DSN provided")
  }

  ctx, cancel := context.WithCancel(t.Context())
  t.Cleanup(cancel)

  tx, err := testDB.Begin(ctx)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  defer func() {
    if err := tx.Rollback(ctx); err != nil {
      t.Fatalf("Error rolling back transaction: %v", err)
    }
  }()

  before, err := models.{{$tAlias.UpPlural}}.Query().Count(ctx, tx)
  if err != nil {
    t.Fatalf("Error counting {{$tAlias.UpPlural}}: %v", err)
  }

  if _, err := New().New{{$tAlias.UpSingular}}WithContext(ctx, {{$tAlias.UpSingular}}Mods.WithNew{{$relAlias}}(2)).Create(ctx, tx); err != nil {
    t.Fatalf("Error creating {{$tAlias.UpSingular}} with {{$relAlias}}: %v", err)
  }

  after, err := models.{{$tAlias.UpPlural}}.Query().Count(ctx, tx)
  if err != nil {
    t.Fatalf("Error counting {{$tAlias.UpPlural}}: %v", err)
  }

  if got := after - before; got != 1 {
    t.Fatalf("Expected {{$tAlias.UpPlural}} to increase by 1, got %d", got)
  }
}

{{end}}

{{end}}
