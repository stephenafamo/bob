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

  tx, err := testDB.Begin(ctx)
  if err != nil {
    t.Fatalf("Error starting transaction: %v", err)
  }

  defer func() {
    if err := tx.Rollback(ctx); err != nil {
      t.Fatalf("Error rolling back transaction: %v", err)
    }
  }()

  if _, err := New().New{{$tAlias.UpSingular}}(ctx).Create(ctx, tx); err != nil {
    t.Fatalf("Error creating {{$tAlias.UpSingular}}: %v", err)
  }
}

{{end}}
