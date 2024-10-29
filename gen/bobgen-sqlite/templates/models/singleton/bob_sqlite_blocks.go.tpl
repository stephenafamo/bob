{{- define "helpers/join_variables"}}
var (
	SelectJoins = getJoins[*dialect.SelectQuery]
	UpdateJoins = getJoins[*dialect.UpdateQuery]
)
{{end -}}

{{define "setter_insert_mod" -}}
{{$.Importer.Import "io"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s *{{$tAlias.UpSingular}}Setter) Apply(q *dialect.InsertQuery) {
  q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
    return {{$tAlias.UpPlural}}.BeforeInsertHooks.RunHooks(ctx, exec, s)
  })

  if len(q.Table.Columns) == 0 {
    q.Table.Columns = s.SetColumns()
  }

	q.AppendValues(bob.ExpressionFunc(func(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error){
    vals := make([]bob.Expression, 0, {{len $table.NonGeneratedColumns}})
    {{range $index, $column := $table.NonGeneratedColumns -}}
      {{$colAlias := $tAlias.Column $column.Name -}}
      if !s.{{$colAlias}}.IsUnset() {
        vals = append(vals, {{$.Dialect}}.Arg(s.{{$colAlias}}))
      }

    {{end -}}

    return bob.ExpressSlice(ctx, w, d, start, vals, "", ", ", "")
  }))
}
{{- end}}

{{define "unique_constraint_error_detection_method" -}}
func (e *errUniqueConstraint) Is(target error) bool {
	{{if not (eq $.DriverName "modernc.org/sqlite" "github.com/mattn/go-sqlite3")}}
	return false
	{{else}}
		{{$errType := ""}}
		{{$codeGetter := ""}}
		{{$.Importer.Import "strings"}}
		{{if eq $.DriverName "modernc.org/sqlite"}}
			{{$.Importer.Import "sqliteDriver" $.DriverName}}
			{{$errType = "*sqliteDriver.Error"}}
			{{$codeGetter = "Code()"}}
		{{else}}
			{{$.Importer.Import $.DriverName}}
			{{$errType = "sqlite3.Error"}}
			{{$codeGetter = "ExtendedCode"}}
		{{end}}
	err, ok := target.({{$errType}})
	if !ok {
		return false
	}
	return err.{{$codeGetter}} == 2067 && strings.Contains(err.Error(), e.s)
	{{end}}
}
{{end -}}
