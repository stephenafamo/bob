{{$.Importer.Import "github.com/jaswdr/faker/v2"}}


var defaultFaker = faker.New()

{{$doneTypes := dict }}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
    {{- if hasKey $doneTypes $column.Type}}{{continue}}{{end -}}
    {{- $_ := set $doneTypes $column.Type nil -}}
    {{- $typDef := index $.Types $column.Type -}}
    {{range $depTyp := $typDef.DependsOn}}
      {{- $_ := set $doneTypes $depTyp nil -}}
    {{end}}
  {{end -}}
{{- end}}


{{range $colTyp := keys $doneTypes | sortAlpha -}}
    {{- $typDef := index $.Types $colTyp -}}
    {{- $typ := $.Types.Get $.CurrentPackage $.Importer $colTyp -}}
    {{- if not $typDef.RandomExpr -}}{{continue}}{{/*
      Ensures that compilation fails.
      Users of custom types can decide to use a non-random expression
      but this would be a conscious decision.
    */}}{{- end -}}
    {{- $.Importer.ImportList $typDef.RandomExprImports -}}
    func random_{{normalizeType $colTyp}}(f *faker.Faker, limits ...string) {{$typ}} {
      if f == nil {
        f = &defaultFaker
      }

      {{replace "TYPE" $typ $typDef.RandomExpr}}
    }
{{end -}}
