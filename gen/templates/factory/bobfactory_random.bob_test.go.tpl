{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}

// Set the testDB to enable tests that use the database
var testDB bob.Transactor

{{$doneTypes := dict }}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
    {{- if hasKey $doneTypes $column.Type}}{{continue}}{{end -}}
    {{- $_ := set $doneTypes $column.Type nil -}}
    {{- $typDef := $.Types.Index $column.Type -}}
    {{range $depTyp := $typDef.DependsOn}}
      {{- $_ := set $doneTypes $depTyp nil -}}
    {{end}}
  {{end -}}
{{- end}}


{{range $colTyp := keys $doneTypes | sortAlpha -}}
    {{- $typDef := $.Types.Index $colTyp -}}
    {{- if not $typDef.RandomExpr -}}{{continue}}{{/*
      Ensures that compilation fails.
      Users of custom types can decide to use a non-random expression
      but this would be a conscious decision.
    */}}{{- end -}}
    {{- if $typDef.NoRandomizationTest}}{{continue}}{{end -}}
      func TestRandom_{{normalizeType $colTyp}}(t *testing.T) {
        t.Parallel()

        val1 := random_{{normalizeType $colTyp}}(nil)
        val2 := random_{{normalizeType $colTyp}}(nil)

        {{with $typDef.CompareExpr -}}
          {{- $.Importer.ImportList $typDef.CompareExprImports -}}
          if {{replace "AAA" "val1" . | replace "BBB" "val2"}}
        {{- else -}}
          if val1 == val2
        {{- end -}}
        {
          t.Fatalf("random_{{normalizeType $colTyp}}() returned the same value twice: %v", val1)
        }
      }

{{end -}}
