{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

{{if or $table.Constraints.Primary ($.Relationships.Get $table.Key) -}}
// {{$tAlias.UpSingular}}Setter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type {{$tAlias.UpSingular}}Setter struct {
	{{- range $column := $table.NonGeneratedColumns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{- $orig_col_name := $column.Name -}}
    {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
    {{- $colTyp = printf "*%s" $colTyp -}}
		{{- if ignore $table.Key $orig_col_name $.TagIgnore}}
      {{$colAlias}} {{$colTyp}} `db:"{{$table.DBTag $column}}" {{generateIgnoreTags $.Tags | trim}}`
		{{- else -}}
      {{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
			{{$colAlias}} {{$colTyp}} `db:"{{$table.DBTag $column}}" {{generateTags $.Tags $tagName | trim}}`
		{{- end -}}		
	{{end -}}
}

func (s {{$tAlias.UpSingular}}Setter) SetColumns() []string {
  vals := make([]string, 0, {{len $table.NonGeneratedColumns}})
	{{range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if s.{{$colAlias}} != nil {
			vals = append(vals, {{printf "%q" $column.Name}})
		}

	{{end -}}

	return vals
}

func (s {{$tAlias.UpSingular}}Setter) Overwrite(t *{{$tAlias.UpSingular}}) {
	{{- range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if s.{{$colAlias}} != nil {
      t.{{$colAlias}} = *s.{{$colAlias}}
		}
	{{end -}}
}

{{block "setter_insert_mod" . -}}
{{$.Importer.Import "io"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s *{{$tAlias.UpSingular}}Setter) Apply(q *dialect.InsertQuery) {
  {{if $table.Constraints.Primary -}}
    q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
      return {{$tAlias.UpPlural}}.BeforeInsertHooks.RunHooks(ctx, exec, s)
    })
  {{end}}

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error){
    vals := make([]bob.Expression, {{len $table.NonGeneratedColumns}})
    {{range $index, $column := $table.NonGeneratedColumns -}}
      {{$colAlias := $tAlias.Column $column.Name -}}
      if s.{{$colAlias}} != nil {
        vals[{{$index}}] = {{$.Dialect}}.Arg(*s.{{$colAlias}})
      } else {
        vals[{{$index}}] = {{$.Dialect}}.Raw("DEFAULT")
      }

    {{end -}}

    return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
  }))
}
{{- end}}

{{block "setter_update_mod" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) UpdateMod() bob.Mod[*dialect.UpdateQuery] {
  return um.Set(s.Expressions()...)
}
{{- end}}

{{block "setter_expressions" . -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) Expressions(prefix ...string) []bob.Expression {
  exprs := make([]bob.Expression, 0, {{len $table.NonGeneratedColumns}})

  {{$.Importer.Import "github.com/stephenafamo/bob/expr" }}
	{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
	{{range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if s.{{$colAlias}} != nil {
      exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
        {{$.Dialect}}.Quote(append(prefix, "{{$column.Name}}")...), 
        {{$.Dialect}}.Arg(s.{{$colAlias}}),
      }})
		}

	{{end -}}

  return exprs
}
{{- end}}


{{- end}}
