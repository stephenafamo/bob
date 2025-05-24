{{- $.Importer.Import "context" -}}
{{- $.Importer.Import "testing" -}}

{{range $table := .Tables}}{{if not $table.Constraints.Primary}}{{continue}}{{end}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func TestCreate{{$tAlias.UpSingular}}(t *testing.T) {
  if testDB == nil {
    t.Skip("skipping test, no DSN provided")
  }

  ctx, cancel := context.WithCancel(context.Background())
  t.Cleanup(cancel)

  tx, err := testDB.BeginTx(ctx, nil)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  New().New{{$tAlias.UpSingular}}(ctx).CreateOrFail(ctx, t, tx)
}

{{end}}
