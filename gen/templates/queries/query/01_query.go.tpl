{{if .QueryFile.Queries}}

{{$.Importer.Import "io"}}
{{$.Importer.Import "iter"}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}


{{range $query := $.QueryFile.Queries}}

{{$upperName := titleCase $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$flatArgs := $query.ArgsByPosition}}
{{$args := list }}
{{range $arg := $query.Args -}}
  {{if $arg.CanBeMultiple}}
    {{$.Importer.Import "github.com/stephenafamo/bob/expr"}}
  {{end}}
  {{ $argName := camelCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.Importer $.Types) }}
  {{$args = append $args (printf "%s %s" $argName $argType) }}
{{end}}

{{$queryRowName := $query.Config.RowName}}
{{if not $query.Config.GenerateRow}}
  {{- $typDef :=  index $.Types $queryRowName -}}
	{{- $queryRowName = or $typDef.AliasOf $queryRowName -}}
	{{- $.Importer.ImportList $typDef.Imports -}}
{{end}}

{{$queryType := (lower $query.Type.String | titleCase)}}
{{$dialectType := printf "*dialect.%sQuery" $queryType}}
{{$colParams :=  printf "%s, %s" $queryRowName (or $query.Config.RowSliceName (printf "[]%s" $queryRowName)) }}

const {{$lowerName}}SQL = `{{replace "`" "`+\"`\"+`" $query.SQL}}`

{{if $query.Columns -}}
func {{$upperName}} ({{join ", " $args}}) orm.ModQuery[{{$dialectType}}, {{$colParams}}] {
{{- else -}}
func {{$upperName}} ({{join ", " $args}}) orm.ModQuery[{{$dialectType}}] {
{{end}}
  var expressionTypArgs {{$lowerName}}

  {{range $arg := $query.Args -}}
    expressionTypArgs.{{camelCase $arg.Col.Name}} = {{$arg.ToExpression $.Dialect $lowerName (camelCase $arg.Col.Name)}}
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
    {{camelCase $arg.Col.Name}} bob.Expression
  {{end}}
}

func (o {{$lowerName}}) args() iter.Seq[orm.ArgWithPosition]  {
  return func(yield func(arg orm.ArgWithPosition) bool) {
    {{range $flatArg := $flatArgs}}
      if !yield(orm.ArgWithPosition{
        Name: "{{camelCase $flatArg.Name}}",
        Start: {{$flatArg.Start}},
        Stop: {{$flatArg.Stop}},
        Expression: o.{{camelCase $flatArg.Name}},
      }) {
          return
      }
    {{end}}
  }
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
