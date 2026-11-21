{{- /* Tables that can carry this test: a primary key and at least one
     relationship. Unlike the single-model update test, no updatable column is
     needed, since ReloadAll only re-reads. */ -}}
{{- $anyTable := false -}}
{{- range $table := .Tables -}}
  {{- if not $table.Constraints.Primary}}{{continue}}{{end -}}
  {{- if not ($.Relationships.Get $table.Key)}}{{continue}}{{end -}}
  {{- $anyTable = true -}}
{{- end -}}
{{- if $anyTable -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

// zeroOf returns the zero value of the given value's type. It is used to
// break a column's value in memory (without touching the database) so the
// test can check that ReloadAll writes it back.
func zeroOf[T any](T) T { var z T; return z }

{{range $table := .Tables}}
{{- if not $table.Constraints.Primary}}{{continue}}{{end -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- if not $rels}}{{continue}}{{end -}}
{{- $tAlias := $.Aliases.Table $table.Key -}}
{{- $rel := index $rels 0 -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
{{- $ftable := $.Aliases.Table $rel.Foreign }}
{{- /* Pick one non-PK column so the test can also verify that ReloadAll
     really overwrites the column fields, not just that it leaves .R alone.
     Tables with no non-PK column (rare) simply skip that extra check. */ -}}
{{- $pkSet := dict -}}
{{- range $pkCol := $table.Constraints.Primary.Columns -}}
  {{- $_ := set $pkSet $pkCol true -}}
{{- end -}}
{{- $chkCol := "" -}}
{{- range $column := $table.Columns -}}
  {{- if $chkCol}}{{continue}}{{end -}}
  {{- if hasKey $pkSet $column.Name}}{{continue}}{{end -}}
  {{- $chkCol = $column.Name -}}
{{- end -}}
{{- $chkColAlias := "" -}}
{{- if $chkCol}}{{$chkColAlias = $tAlias.Column $chkCol}}{{$.Importer.Import "reflect"}}{{end -}}

// Test{{$tAlias.UpSingular}}ReloadAllKeepsLoadedRelationships checks that
// (models.{{$tAlias.UpSingular}}Slice).ReloadAll only overwrites the column
// fields of the models already in the slice, leaving .R (and any other
// non-column field, such as the counts .C added by the counts plugin)
// untouched, and keeping the slice's original pointers.
func Test{{$tAlias.UpSingular}}ReloadAllKeepsLoadedRelationships(t *testing.T) {
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
  // database: the test only checks that ReloadAll does not throw the cache away.
  loaded := &models.{{$ftable.UpSingular}}{}
  {{if $rel.IsToMany -}}
  obj.R.{{$relAlias}} = models.{{$ftable.UpSingular}}Slice{loaded}
  {{- else -}}
  obj.R.{{$relAlias}} = loaded
  {{- end}}
  obj.R.{{$.RelationLoadedName}}.{{$relAlias}} = true

  {{if $chkCol -}}
  // Break the column in memory (not in the database) so we can check below
  // that ReloadAll actually reads the column back from the database, and
  // does not just leave the slice's models as they already were.
  want{{$chkColAlias}} := obj.{{$chkColAlias}}
  obj.{{$chkColAlias}} = zeroOf(want{{$chkColAlias}})
  {{end -}}

  slice := models.{{$tAlias.UpSingular}}Slice{obj}

  if err := slice.ReloadAll(ctx, tx); err != nil {
    t.Fatalf("Error reloading {{$tAlias.UpSingular}}Slice: %v", err)
  }

  {{if $chkCol -}}
  if !reflect.DeepEqual(obj.{{$chkColAlias}}, want{{$chkColAlias}}) {
    t.Fatalf("ReloadAll did not copy {{$chkColAlias}} back: got %#v, want %#v",
      obj.{{$chkColAlias}}, want{{$chkColAlias}})
  }
  {{end -}}

  // The slice must keep the pointer it was given, not swap in the freshly
  // scanned model, otherwise callers holding obj would not see the reload.
  if slice[0] != obj {
    t.Fatal("ReloadAll replaced the pointer in the slice")
  }

  {{if $rel.IsToMany -}}
  if len(obj.R.{{$relAlias}}) != 1 || obj.R.{{$relAlias}}[0] != loaded {
    t.Fatalf("ReloadAll did not keep R.{{$relAlias}}, got %#v", obj.R.{{$relAlias}})
  }
  {{- else -}}
  if obj.R.{{$relAlias}} != loaded {
    t.Fatalf("ReloadAll did not keep R.{{$relAlias}}, got %#v", obj.R.{{$relAlias}})
  }
  {{- end}}
  if !obj.R.{{$.RelationLoadedName}}.{{$relAlias}} {
    t.Fatal("ReloadAll did not keep R.{{$.RelationLoadedName}}.{{$relAlias}}")
  }
}
{{end}}
{{- end -}}
