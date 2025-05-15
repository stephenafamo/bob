{{if .QueryFile.Queries}}

{{$.Importer.Import "io"}}
{{$.Importer.Import "iter"}}
{{$.Importer.Import "_" "embed"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

//go:embed {{.QueryFile.BaseName}}.bob.sql
var formattedQueries_{{.QueryFile.BaseName}} string

{{range $queryIndex, $query := $.QueryFile.Queries}}
{{if $query.Args}}
  {{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{end}}

{{$upperName := titleCase $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$flatArgs := $query.ArgsByPosition}}
{{$queryRowName := $query.Config.RowName}}
{{if not $query.Config.GenerateRow}}
  {{- $typDef :=  index $.Types $queryRowName -}}
	{{- $queryRowName = or $typDef.AliasOf $queryRowName -}}
	{{- $.Importer.ImportList $typDef.Imports -}}
{{end}}

{{$queryType := (lower $query.Type.String | titleCase)}}
{{$dialectType := printf "*dialect.%sQuery" $queryType}}
{{$colParams :=  printf "%s, %s" $queryRowName (or $query.Config.RowSliceName (printf "[]%s" $queryRowName)) }}
{{if eq (len $query.Columns) 1}}
  {{$col := index $query.Columns 0}}
  {{$colType := $col.Type $.Importer $.Types}}
  {{$colParams =  printf "%s, %s" $colType (or $query.Config.RowSliceName (printf "[]%s" $colType)) }}
{{end}}

var {{$lowerName}}SQL = formattedQueries_{{$.QueryFile.BaseName}}[{{$.QueryFile.QueryPosition $queryIndex (len $.Language.Disclaimer)}}]

{{$args := list }}
{{range $arg := $query.Args -}}
  {{if $arg.CanBeMultiple}}
    {{$.Importer.Import "github.com/stephenafamo/bob/expr"}}
  {{end}}

  {{ $argName := titleCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.Importer $.Types) }}

  {{if gt (len $arg.Children) 0}}
    {{ $argType = printf "%s_%s" $upperName $argName }}
    type {{$argType}} = {{$arg.TypeDef $.Importer $.Types}}
    {{if $arg.CanBeMultiple}}
      {{ $argType = printf "[]%s" $argType }}
    {{end}}
  {{end}}

  {{$args = append $args (printf "%s %s" $argName $argType) }}
{{end}}


{{if $query.Columns -}}
func {{$upperName}} ({{join ", " $args}}) orm.ModQuery[{{$dialectType}}, {{$colParams}}] {
{{- else -}}
func {{$upperName}} ({{join ", " $args}}) orm.ModExecQuery[{{$dialectType}}] {
{{end}}
  var expressionTypArgs {{$lowerName}}

  {{range $arg := $query.Args -}}
    expressionTypArgs.{{titleCase $arg.Col.Name}} = {{$arg.ToExpression $.Dialect $lowerName (titleCase $arg.Col.Name)}}
  {{end}}

{{if $query.Columns -}}
  return orm.ModQuery[{{$dialectType}}, {{$colParams}}]{
    Query: orm.Query[orm.ModExpression[{{$dialectType}}], {{$colParams}}]{
      ExecQuery: orm.ExecQuery[orm.ModExpression[{{$dialectType}}]]{
        BaseQuery: bob.BaseQuery[orm.ModExpression[{{$dialectType}}]]{
          Expression: expressionTypArgs,
          Dialect:    dialect.Dialect,
          QueryType:  bob.QueryType{{$queryType}},
        },
      },
      {{- $.Importer.Import "github.com/stephenafamo/scan" -}}
      {{if gt (len $query.Columns) 1 -}}
        Scanner: scan.StructMapper[{{$queryRowName}}](),
      {{- else -}}
        {{- $col := index $query.Columns 0 -}}
        Scanner: scan.ColumnMapper[{{$col.Type $.Importer $.Types}}]("{{$col.DBName}}"),
      {{- end}}
    },
  }
}
{{- else -}}
	return orm.ModExecQuery[{{$dialectType}}]{
		ExecQuery: orm.ExecQuery[orm.ModExpression[{{$dialectType}}]]{
			BaseQuery: bob.BaseQuery[orm.ModExpression[{{$dialectType}}]]{
        Expression: expressionTypArgs,
				Dialect:    dialect.Dialect,
        QueryType:  bob.QueryType{{$queryType}},
			},
		},
	}
}
{{- end}} 

{{if and $query.Columns $query.Config.GenerateRow}}
type {{$queryRowName}} struct {
  {{range $col := $query.Columns -}}
    {{titleCase $col.Name}} {{$col.Type $.Importer $.Types}} `db:"{{$col.DBName}}"`
  {{end}}
}
{{end}}

type {{$lowerName}} struct {
  {{range $arg := $query.Args -}}
    {{titleCase $arg.Col.Name}} bob.Expression
  {{end}}
}

func (o {{$lowerName}}) args() iter.Seq[orm.ArgWithPosition]  {
  return func(yield func(arg orm.ArgWithPosition) bool) {
    {{range $flatArg := $flatArgs}}
      if !yield(orm.ArgWithPosition{
        Name: "{{camelCase $flatArg.Name}}",
        Start: {{$flatArg.Start}},
        Stop: {{$flatArg.Stop}},
        Expression: o.{{titleCase $flatArg.Name}},
      }) {
          return
      }
    {{end}}
  }
}

func (o {{$lowerName}}) raw(from, to int) string {
  return {{$lowerName}}SQL[from:to]
}

func (o {{$lowerName}}) expr(from, to int) bob.Expression {
  return orm.ArgsToExpression({{$lowerName}}SQL, from, to, o.args())
}

func (o {{$lowerName}}) Apply(q {{$dialectType}}) {
  {{$query.Mods.IncludeInTemplate $.Importer}}
}

func (o {{$lowerName}}) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return o.expr(0, len({{$lowerName}}SQL)).WriteSQL(ctx, w, d, start)
}

{{end}}


{{end}}
