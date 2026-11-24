{{if .QueryFile.Queries}}

{{$.Importer.Import "io"}}
{{$.Importer.Import "_" "embed"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

//go:embed {{.QueryFile.BaseName}}.bob.sql
var formattedQueries_{{.QueryFile.BaseName}} string

{{range $queryIndex, $query := $.QueryFile.Queries}}
{{$upperName := titleCase $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$flatArgs := $query.ArgsByPosition}}
{{$cols := $query.Columns.WithNames}}

{{$queryResultTypeOne := printf "%sRow" $upperName}}
{{if $query.Config.ResultTypeOne}}
  {{$queryResultTypeOne = $.Types.Get $.CurrentPackage $.Importer $query.Config.ResultTypeOne}}
{{end}}

{{$queryResultTypeAll := printf "[]%s" $queryResultTypeOne}}
{{if $query.Config.ResultTypeAll}}
  {{$queryResultTypeAll = $.Types.Get $.CurrentPackage $.Importer $query.Config.ResultTypeAll}}
{{else if and $query.HasNestedReturns (not $query.Config.ResultTransformer)}}
  {{$queryResultTypeAll = printf "All%s" $queryResultTypeOne}}
{{end}}

{{$queryResultTransformer := printf "%sTransformer" $lowerName}}
{{if not (list "" "slice" | has $query.Config.ResultTransformer)}}
  {{$queryResultTransformer = $.Types.Get $.CurrentPackage $.Importer $query.Config.ResultTransformer}}
{{end}}

{{$queryType := (lower $query.Type.String | titleCase)}}
{{$dialectType := printf "*dialect.%sQuery" $queryType}}
{{$colParams :=  printf "%s, %s, %s" $queryResultTypeOne $queryResultTypeAll $queryResultTransformer }}
{{if eq (len $query.Columns) 1}}
  {{$col := index $query.Columns 0}}
  {{$colType := $col.Type $.CurrentPackage $.Importer $.Types}}
  {{$colParams =  printf "%s, %s, %s" $colType (or $query.Config.ResultTypeAll (printf "[]%s" $colType)) $queryResultTransformer }}
{{end}}

var {{$lowerName}}SQL = formattedQueries_{{$.QueryFile.BaseName}}[{{$.QueryFile.QueryPosition $queryIndex (len $.Language.Disclaimer)}}]

{{$args := list }}
{{range $arg := $query.Args -}}
  {{ $argName := titleCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.CurrentPackage $.Importer $.Types) }}

  {{if gt (len $arg.Children) 0}}
    {{ $argType = printf "%s_%s" $upperName $argName }}
    type {{$argType}} = {{$arg.TypeDef $.CurrentPackage $.Importer $.Types true}}
    {{if $arg.CanBeMultiple}}
      {{ $argType = printf "[]%s" $argType }}
    {{end}}
  {{end}}

  {{$args = append $args (printf "%s %s" $argName $argType) }}
{{end}}


{{if $query.Columns -}}
type {{$upperName}}Query = orm.ModQuery[{{$dialectType}}, {{$lowerName}}, {{$colParams}}]
{{- else -}}
type {{$upperName}}Query = orm.ModExecQuery[{{$dialectType}}, {{$lowerName}}]
{{end}}

func {{$upperName}} ({{join ", " $args}}) *{{$upperName}}Query {
  a := {{$lowerName}}{
    {{range $arg := $query.Args -}}
      {{titleCase $arg.Col.Name}}: {{$arg.ToExpression $.Importer $.Dialect $lowerName (titleCase $arg.Col.Name)}},
    {{end -}}
  }
  return genruntime.{{if $query.Columns}}NewModQuery[{{$queryResultTransformer}}]{{else}}NewModExecQuery{{end}}(
    a, dialect.Dialect, bob.QueryType{{$queryType}}, {{$.Dialect}}.{{$queryType}},
    func(q {{$dialectType}}) {
      {{$mods := $query.Mods.IncludeInTemplate $.Importer -}}
      {{if contains "EXPR.subExpr(" $mods}}fragment := genruntime.BindSQL({{$lowerName}}SQL, a.args()...){{end}}
      {{if contains "EXPR.raw(" $mods}}raw := func(from, to int) string { return {{$lowerName}}SQL[from:to] }{{end}}
      {{$mods | replace "EXPR.subExpr(" "fragment(" | replace "EXPR.raw(" "raw(" | replace "EXPR" "a"}}
    },
    {{if $query.Columns}}scanMapper_{{$lowerName}},{{end}}
  )
}

{{if $query.Columns}}
  {{$.Importer.Import "github.com/stephenafamo/scan"}}
  var scanMapper_{{$lowerName}} = {{if eq (len $cols) 1 -}}
    {{$col := index $cols 0 -}}
    scan.ColumnMapper[{{$col.Type $.CurrentPackage $.Importer $.Types}}]({{quote $col.DBName}})
  {{else if $query.Config.ResultTransformer -}}
    scan.StructMapper[{{$queryResultTypeOne}}]()
  {{else -}}
    genruntime.PositionalMapper(func(row *scan.Row, t *{{$queryResultTypeOne}}) {
      {{range $i, $col := $cols -}}row.ScheduleScanByIndex({{$i}}, &t.{{$col.Name}})
      {{end -}}
    })
  {{end -}}

  {{if not $query.Config.ResultTransformer}}
    type {{trimPrefix "*" $queryResultTypeOne}} = struct {
      {{$tags := $.QueryResultTags.FieldTags $cols -}}
      {{range $i, $col := $cols -}}
        {{$col.Name}} {{$col.Type $.CurrentPackage $.Importer $.Types}} `db:"{{$col.DBName}}"{{with index $tags $i}} {{.}}{{end}}`
      {{end}}
    }

    {{if and (not $query.HasNestedReturns) ($query.Config.ResultTypeAll)}}
      type {{$queryResultTypeAll}} = []{{$queryResultTypeOne}}
    {{end}}
  {{end}}



  {{if and $query.HasNestedReturns (not $query.Config.ResultTransformer)}}
    {{$nested := $query.NestedColumns}}
    {{$typeName := printf "%s_" $queryResultTypeOne}}

    type {{$lowerName}}Transformer struct{}

    func ({{$lowerName}}Transformer) TransformScanned(scanned []{{$queryResultTypeOne}}) ({{$queryResultTypeAll}}, error) {
      final := make({{$queryResultTypeAll}}, 0, len(scanned))

      {{$nested.TransformRows $.CurrentPackage $.Importer $.Types $cols $typeName}}

      return final, nil
    }

    type {{$queryResultTypeAll}} = []{{$queryResultTypeOne}}_

    {{range $nested.TypesWithTags $.CurrentPackage $.Importer $.Types $typeName $.QueryResultTags}}
    {{.}}
    {{end}}

  {{else if (list "" "slice"| has $query.Config.ResultTransformer)}}
    {{if eq (len $query.Columns) 1}}
      {{$col := index $query.Columns 0}}
      {{$colType := $col.Type $.CurrentPackage $.Importer $.Types}}
      type {{$lowerName}}Transformer = {{printf "bob.SliceTransformer[%s, %s]" $colType (or $query.Config.ResultTypeAll (printf "[]%s" $colType))}}
    {{else}}
      type {{$lowerName}}Transformer = {{printf "bob.SliceTransformer[%s, %s]" $queryResultTypeOne $queryResultTypeAll}}
    {{end}}
  {{end}}
{{end}}

type {{$lowerName}} struct {
  {{range $arg := $query.Args -}}
    {{titleCase $arg.Col.Name}} bob.Expression
  {{end}}
}

func (o {{$lowerName}}) args() []orm.ArgWithPosition {
  return []orm.ArgWithPosition{ {{range $flatArg := $flatArgs}}
    {Name: {{quote (camelCase $flatArg.Name)}}, Start: {{$flatArg.Start}}, Stop: {{$flatArg.Stop}}, Expression: o.{{titleCase $flatArg.Name}}},{{end}} }
}

func (o {{$lowerName}}) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return genruntime.WriteSQL(ctx, w, d, start, {{$lowerName}}SQL, o.args())
}

{{end}}


{{end}}
