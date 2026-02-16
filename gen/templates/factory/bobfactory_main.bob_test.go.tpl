{{- $.Importer.Import "context" -}}
{{- $.Importer.Import "errors" -}}
{{- $.Importer.Import "testing" -}}
{{- $.Importer.Import "models" (index $.OutputPackages "models") -}}

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

{{- $hasRequiredCols := false -}}
{{- range $column := $table.Columns -}}
  {{- if $column.Default}}{{continue}}{{end -}}
  {{- if $column.Nullable}}{{continue}}{{end -}}
  {{- if $column.Generated}}{{continue}}{{end -}}
  {{- $hasRequiredCols = true -}}
{{- end}}

{{if $hasRequiredCols}}
func TestRequireAll{{$tAlias.UpSingular}}(t *testing.T) {
  var setter models.{{$tAlias.UpSingular}}Setter
  err := ensureCreatable{{$tAlias.UpSingular}}(&setter, true)

  var missingErr *MissingRequiredFieldsError
  if !errors.As(err, &missingErr) {
    t.Fatalf("Expected MissingRequiredFieldsError, got: %v", err)
  }

  if missingErr.TableName != "{{$tAlias.UpSingular}}" {
    t.Errorf("Expected table name %q, got %q", "{{$tAlias.UpSingular}}", missingErr.TableName)
  }

  expectedMissing := map[string]struct{}{
    {{- range $column := $table.Columns -}}
      {{- if $column.Default}}{{continue}}{{end -}}
      {{- if $column.Nullable}}{{continue}}{{end -}}
      {{- if $column.Generated}}{{continue}}{{end}}
      "{{$tAlias.Column $column.Name}}": {},
    {{- end}}
  }

  if len(missingErr.Missing) != len(expectedMissing) {
    t.Fatalf("Expected %d missing fields, got %d: %v", len(expectedMissing), len(missingErr.Missing), missingErr.Missing)
  }

  for _, field := range missingErr.Missing {
    if _, ok := expectedMissing[field]; !ok {
      t.Errorf("Unexpected missing field: %s", field)
    }
  }
}
{{end}}

{{end}}
