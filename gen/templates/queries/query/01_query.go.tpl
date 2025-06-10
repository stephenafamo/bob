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
{{$upperName := titleCase $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$flatArgs := $query.ArgsByPosition}}
{{$queryRowName := $query.Config.RowName}}
{{if not $query.Config.GenerateRow}}
  {{- $queryRowName = $.Types.Get $.CurrentPackage $.Importer $queryRowName -}}
{{end}}

{{$queryType := (lower $query.Type.String | titleCase)}}
{{$dialectType := printf "*dialect.%sQuery" $queryType}}
{{$colParams :=  printf "%s, %s" $queryRowName (or $query.Config.RowSliceName (printf "[]%s" $queryRowName)) }}
{{if eq (len $query.Columns) 1}}
  {{$col := index $query.Columns 0}}
  {{$colType := $col.Type $.CurrentPackage $.Importer $.Types}}
  {{$colParams =  printf "%s, %s" $colType (or $query.Config.RowSliceName (printf "[]%s" $colType)) }}
{{end}}

var {{$lowerName}}SQL = formattedQueries_{{$.QueryFile.BaseName}}[{{$.QueryFile.QueryPosition $queryIndex (len $.Language.Disclaimer)}}]

{{$args := list }}
{{range $arg := $query.Args -}}
  {{if $arg.CanBeMultiple}}
    {{$.Importer.Import "github.com/stephenafamo/bob/expr"}}
  {{end}}

  {{ $argName := titleCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.CurrentPackage $.Importer $.Types) }}

  {{if gt (len $arg.Children) 0}}
    {{ $argType = printf "%s_%s" $upperName $argName }}
    type {{$argType}} = {{$arg.TypeDef $.CurrentPackage $.Importer $.Types}}
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
  var expressionTypArgs {{$lowerName}}

  {{range $arg := $query.Args -}}
    expressionTypArgs.{{titleCase $arg.Col.Name}} = {{$arg.ToExpression $.Importer $.Dialect $lowerName (titleCase $arg.Col.Name)}}
  {{end}}

  {{if $query.Columns -}}
    {{$.Importer.Import "github.com/stephenafamo/scan"}}
    return &{{$upperName}}Query{
      Query: orm.Query[{{$lowerName}}, {{$colParams}}]{
        ExecQuery: orm.ExecQuery[{{$lowerName}}]{
          BaseQuery: bob.BaseQuery[{{$lowerName}}]{
            Expression: expressionTypArgs,
            Dialect:    dialect.Dialect,
            QueryType:  bob.QueryType{{$queryType}},
          },
        },
        {{if gt (len $query.Columns) 1 -}}
          {{if not $query.Config.GenerateRow -}}
          Scanner: scan.StructMapper[{{$queryRowName}}](),
          {{- else -}}
          Scanner: func(context.Context, []string) (func(*scan.Row) (any, error), func(any) ({{$queryRowName}}, error)) {
            return func(row *scan.Row) (any, error) {
                var t {{$queryRowName}}
                {{range $colIndex, $col := $query.Columns -}}
                  row.ScheduleScanByIndex({{$colIndex}}, &t.{{titleCase $col.Name}})
                {{end -}}
                return &t, nil
              }, func(v any) ({{$queryRowName}}, error) {
                return *(v.(*{{$queryRowName}})), nil
              }
          },
          {{- end}}
        {{- else -}}
          {{- $col := index $query.Columns 0 -}}
          Scanner: scan.ColumnMapper[{{$col.Type $.CurrentPackage $.Importer $.Types}}]("{{$col.DBName}}"),
        {{- end}}
      },
      Mod: bob.ModFunc[{{$dialectType}}](func(q {{$dialectType}}) {
          {{replace "EXPR" "expressionTypArgs" ($query.Mods.IncludeInTemplate $.Importer)}}
      }),
    }
  {{- else -}}
    return &{{$upperName}}Query{
      ExecQuery: orm.ExecQuery[{{$lowerName}}]{
        BaseQuery: bob.BaseQuery[{{$lowerName}}]{
          Expression: expressionTypArgs,
          Dialect:    dialect.Dialect,
          QueryType:  bob.QueryType{{$queryType}},
        },
      },
      Mod: bob.ModFunc[{{$dialectType}}](func(q {{$dialectType}}) {
          {{replace "EXPR" "expressionTypArgs" ($query.Mods.IncludeInTemplate $.Importer)}}
      }),
    }
  {{- end}} 
}

{{if and $query.Columns $query.Config.GenerateRow}}
type {{trimPrefix "*" $queryRowName}} struct {
  {{range $col := $query.Columns -}}
    {{titleCase $col.Name}} {{$col.Type $.CurrentPackage $.Importer $.Types}} `db:"{{$col.DBName}}"`
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

func (o {{$lowerName}}) subExpr(from, to int) bob.Expression {
  return orm.ArgsToExpression({{$lowerName}}SQL, from, to, o.args())
}

func (o {{$lowerName}}) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return o.subExpr(0, len({{$lowerName}}SQL)).WriteSQL(ctx, w, d, start)
}

{{end}}


{{end}}
