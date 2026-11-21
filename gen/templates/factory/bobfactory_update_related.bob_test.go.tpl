{{- /* Tables that can carry this test: a primary key (Update and BuildSetter are
     both gated on it), at least one relationship, and at least one column that
     can be assigned in an UPDATE. */ -}}
{{- $anyTable := false -}}
{{- range $table := .Tables -}}
  {{- if not $table.Constraints.Primary}}{{continue}}{{end -}}
  {{- if not ($.Relationships.Get $table.Key)}}{{continue}}{{end -}}
  {{- $pkSet := dict -}}
  {{- range $pkCol := $table.Constraints.Primary.Columns -}}
    {{- $_ := set $pkSet $pkCol true -}}
  {{- end -}}
  {{- range $column := $table.Columns -}}
    {{- if $column.Generated}}{{continue}}{{end -}}
    {{- if $column.AutoIncr}}{{continue}}{{end -}}
    {{- if hasKey $pkSet $column.Name}}{{continue}}{{end -}}
    {{- $anyTable = true -}}
  {{- end -}}
{{- end -}}
{{- if $anyTable -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

{{range $table := .Tables}}
{{- if not $table.Constraints.Primary}}{{continue}}{{end -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- if not $rels}}{{continue}}{{end -}}
{{- $tAlias := $.Aliases.Table $table.Key -}}
{{- $pkSet := dict -}}
{{- range $pkCol := $table.Constraints.Primary.Columns -}}
  {{- $_ := set $pkSet $pkCol true -}}
{{- end -}}
{{- $updCol := "" -}}
{{- range $column := $table.Columns -}}
  {{- if $updCol}}{{continue}}{{end -}}
  {{- if $column.Generated}}{{continue}}{{end -}}
  {{- if $column.AutoIncr}}{{continue}}{{end -}}
  {{- if hasKey $pkSet $column.Name}}{{continue}}{{end -}}
  {{- $updCol = $column.Name -}}
{{- end -}}
{{- if not $updCol}}{{continue}}{{end -}}
{{- $updColAlias := $tAlias.Column $updCol -}}
{{- $rel := index $rels 0 -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $ftable := $.Aliases.Table $rel.Foreign }}

// Test{{$tAlias.UpSingular}}UpdateKeepsLoadedRelationships checks that
// (*models.{{$tAlias.UpSingular}}).Update only overwrites the column fields,
// leaving .R (and any other non-column field, such as the counts .C added by
// the counts plugin) untouched.
func Test{{$tAlias.UpSingular}}UpdateKeepsLoadedRelationships(t *testing.T) {
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

  f := New()

  obj, err := f.New{{$tAlias.UpSingular}}WithContext(ctx).Create(ctx, tx)
  if err != nil {
    t.Fatalf("Error creating {{$tAlias.UpSingular}}: %v", err)
  }

  // A known instance in the relationship cache. It does not have to exist in the
  // database: the test only checks that Update does not throw the cache away.
  loaded := &models.{{$ftable.UpSingular}}{}
  {{if $rel.IsToMany -}}
  obj.R.{{$relAlias}} = models.{{$ftable.UpSingular}}Slice{loaded}
  {{- else -}}
  obj.R.{{$relAlias}} = loaded
  {{- end}}
  obj.R.{{$.RelationLoadedName}}.{{$relAlias}} = true

  // Update one column to the value it already has, so the statement is valid for
  // every schema: no unique, foreign key or NOT NULL violation is possible.
  setter := f.New{{$tAlias.UpSingular}}WithContext(ctx,
    {{$tAlias.UpSingular}}Mods.{{$updColAlias}}(obj.{{$updColAlias}}),
  ).BuildSetter()

  if err := obj.Update(ctx, tx, setter); err != nil {
    t.Fatalf("Error updating {{$tAlias.UpSingular}}: %v", err)
  }

  {{if $rel.IsToMany -}}
  if len(obj.R.{{$relAlias}}) != 1 || obj.R.{{$relAlias}}[0] != loaded {
    t.Fatalf("Update did not keep R.{{$relAlias}}, got %#v", obj.R.{{$relAlias}})
  }
  {{- else -}}
  if obj.R.{{$relAlias}} != loaded {
    t.Fatalf("Update did not keep R.{{$relAlias}}, got %#v", obj.R.{{$relAlias}})
  }
  {{- end}}
  if !obj.R.{{$.RelationLoadedName}}.{{$relAlias}} {
    t.Fatal("Update did not keep R.{{$.RelationLoadedName}}.{{$relAlias}}")
  }
}
{{end}}
{{- end -}}
