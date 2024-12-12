{{$.Importer.Import "testing"}}


{{$doneTypes := dict }}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
    {{- $colTyp := $column.Type -}}
    {{- if hasKey $doneTypes $colTyp}}{{continue}}{{end -}}
    {{- $_ :=  set $doneTypes $colTyp nil -}}
    {{- $typInfo :=  index $.Types $column.Type -}}
    {{- if eq $colTyp "bool"}}{{continue}}{{end -}}
    {{- if $typInfo.NoRandomizationTest}}{{continue}}{{end -}}
      func TestRandom_{{normalizeType $colTyp}}(t *testing.T) {
        t.Parallel()

        val1 := random_{{normalizeType $colTyp}}(nil)
        val2 := random_{{normalizeType $colTyp}}(nil)

        {{with $typInfo.CompareExpr -}}
          {{- $.Importer.ImportList $typInfo.CompareExprImports -}}
          if {{replace "AAA" "val1" . | replace "BBB" "val2"}}
        {{- else -}}
          if val1 == val2
        {{- end -}}
        {
          t.Fatalf("random_{{normalizeType $colTyp}}() returned the same value twice: %v", val1)
        }
      }

  {{end -}}
{{- end}}
