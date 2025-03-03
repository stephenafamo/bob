{{$.Importer.Import "github.com/jaswdr/faker/v2"}}


var defaultFaker = faker.New()

{{$doneTypes := dict }}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
      {{- $colTyp := $column.Type -}}
      {{- if hasKey $doneTypes $column.Type}}{{continue}}{{end -}}
      {{- $_ := set $doneTypes $column.Type nil -}}
      {{range $depTyp := (index $.Types $column.Type).DependsOn}}
        {{- $_ := set $doneTypes $depTyp nil -}}
      {{end}}
  {{end -}}
{{- end}}


{{range $colTyp := keys $doneTypes | sortAlpha -}}
    {{- $typDef := index $.Types $colTyp -}}
    {{- if not $typDef.RandomExpr -}}{{continue}}{{/*
      Ensures that compilation fails.
      Users of custom types can decide to use a non-random expression
      but this would be a conscious decision.
    */}}{{- end -}}
    {{- $.Importer.ImportList $typDef.Imports -}}
    {{- $.Importer.ImportList $typDef.RandomExprImports -}}
    {{- if $typDef.InGeneratedPackage -}}
      {{$.Importer.Import "models" $.ModelsPackage}}
    {{- end -}}
    func random_{{normalizeType $colTyp}}(f *faker.Faker) {{getType $colTyp $typDef}} {
      if f == nil {
        f = &defaultFaker
      }

      {{$typDef.RandomExpr}}
    }
{{end -}}
