{{- $.Importer.Import "context" -}}
{{- $.Importer.Import "testing" -}}

{{range $table := .Tables}}{{if not $table.Constraints.Primary}}{{continue}}{{end}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func TestCreate{{$tAlias.UpSingular}}(t *testing.T) {
  if testDB == nil {
    t.Skip("skipping test, no DSN provided")
  }

  ctxTx, cancel := context.WithCancel(context.Background())
  defer cancel()

  tx, err := testDB.BeginTx(ctxTx, nil)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  New().New{{$tAlias.UpSingular}}().CreateOrFail(ctxTx, t, tx)
}

{{end}}
