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
{{$hasNested := $query.HasNestedReturns}}

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
{{if not (list "" "slice"| has $query.Config.ResultTransformer)}}
  {{$queryResultTransformer = $.Types.Get $.CurrentPackage $.Importer $query.Config.ResultTransformer}}
{{end}}

{{$queryType := (lower $query.Type.String | titleCase)}}
{{$dialectType := printf "*dialect.%sQuery" $queryType}}
{{$colParams :=  printf "%s, %s, %s" $queryResultTypeOne $queryResultTypeAll $queryResultTransformer }}
{{if eq (len $query.Columns) 1}}
  {{$col := index $query.Columns 0}}
  {{$colType := $col.Type $.CurrentPackage $.Importer $.Types}}
  {{$colParams =  printf "%s, %s" $colType (or $query.Config.ResultTypeAll (printf "[]%s" $colType)) }}
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
          {{if $query.Config.ResultTransformer -}}
          Scanner: scan.StructMapper[{{$queryResultTypeOne}}](),
          {{- else -}}
          Scanner: func(context.Context, []string) (func(*scan.Row) (any, error), func(any) ({{$queryResultTypeOne}}, error)) {
            return func(row *scan.Row) (any, error) {
                var t {{$queryResultTypeOne}}
                {{range $colIndex, $col := $query.Columns.WithNames -}}
                  row.ScheduleScanByIndex({{$colIndex}}, &t.{{titleCase $col.Name}})
                {{end -}}
                return &t, nil
              }, func(v any) ({{$queryResultTypeOne}}, error) {
                return *(v.(*{{$queryResultTypeOne}})), nil
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

{{if $query.Columns}}
  {{if not $query.Config.ResultTransformer}}
    type {{trimPrefix "*" $queryResultTypeOne}} = struct {
      {{range $col := $query.Columns.WithNames -}}
        {{$col.Name}} {{$col.Type $.CurrentPackage $.Importer $.Types}} `db:"{{$col.DBName}}"`
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

      for _, row := range scanned {
          {{- $nested.Transform $.CurrentPackage $.Importer $.Types $query.Columns.WithNames false $typeName "final" "index"}}
      }

      return final, nil
    }

    type {{$queryResultTypeAll}} = []{{$queryResultTypeOne}}_

    {{range $nested.Types $.CurrentPackage $.Importer $.Types $typeName}} 
    {{.}}
    {{end}}

  {{else if (list "" "slice"| has $query.Config.ResultTransformer)}}
    type {{$lowerName}}Transformer = {{printf "bob.SliceTransformer[%s, %s]" $queryResultTypeOne $queryResultTypeAll}}
  {{end}}
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
