{{$.Importer.Import "testing"}}


{{$doneTypes := dict }}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
    {{- $colTyp := $column.Type -}}
    {{- if hasKey $doneTypes $colTyp}}{{continue}}{{end -}}
    {{- $.Importer.ImportList $column.Imports -}}
    {{- $_ :=  set $doneTypes $colTyp nil -}}

      func TestRandom_{{replace $colTyp "." "_"}}(t *testing.T) {
        t.Parallel()

        seen := make([]{{$colTyp}}, 10)
        for i := 0; i < 10; i++ {
          seen[i] = random[{{$colTyp}}](nil)
          for j := 0; j < i; j++ {
            if seen[i] == seen[j] {
              t.Fatalf("random[{{$colTyp}}]() returned the same value twice: %v", seen[i])
            }
          }
        }
      }

  {{end -}}
{{- end}}
