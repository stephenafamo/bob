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
    {{- $.Importer.ImportList $typInfo.Imports -}}

      func TestRandom_{{$colTyp | replace "." "_" | replace "[" "_" | replace "]" "_"}}(t *testing.T) {
        t.Parallel()

        seen := make([]{{$colTyp}}, 10)
        for i := 0; i < 10; i++ {
          seen[i] = random[{{$colTyp}}](nil)
          for j := 0; j < i; j++ {
            {{- with $typInfo.CompareExpr -}}
              {{- $.Importer.ImportList $typInfo.CompareExprImports -}}
              if {{replace "AAA" "seen[i]" . | replace "BBB" "seen[j]"}}
            {{- else -}}
              if seen[i] == seen[j]
            {{- end -}}
            {
              t.Fatalf("random[{{$colTyp}}]() returned the same value twice: %v", seen[i])
            }
          }
        }
      }

  {{end -}}
{{- end}}
