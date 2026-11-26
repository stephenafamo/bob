{{$.Importer.Import "github.com/jaswdr/faker/v2"}}


var defaultFaker = faker.New()

{{$doneTypes := dict }}
{{$nullable := dict}}
{{- range $table := .Tables}}
{{- $tAlias := $.Aliases.Table $table.Key}}
  {{range $column := $table.Columns -}}
    {{if and $column.Nullable (not (hasKey $nullable $column.Type)) -}}
      {{$_ := set $nullable $column.Type true -}}
      func randomNullable_{{normalizeType $column.Type}}(f *faker.Faker, notNull bool, limits ...string) {{$.Types.GetNullable $.CurrentPackage $.Importer $column.Type true}} {
        if f == nil { f = &defaultFaker }
        val := random_{{normalizeType $column.Type}}(f, limits...)
        return {{$.Tables.ColumnSetter $.CurrentPackage $.Importer $.Types $table.Key $column.Name "val" "(notNull || f.Bool())"}}
      }
    {{end -}}
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
    {{- $typ := $.Types.Get $.CurrentPackage $.Importer $colTyp -}}
    {{- $.Importer.ImportList $typDef.RandomExprImports -}}
    func random_{{normalizeType $colTyp}}(f *faker.Faker, limits ...string) {{$typ}} {
      if f == nil {
        f = &defaultFaker
      }

      {{replace "BASETYPE" $typ $typDef.RandomExpr}}
    }
{{end -}}
