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
    {{- $colTyp := $.Types.GetOptional $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
		{{- if ignore $table.Key $orig_col_name $.TagIgnore}}
      {{$colAlias}} {{$colTyp.Name}} `db:"{{$table.DBTag $column}}" {{generateIgnoreTags $.Tags | trim}}`
		{{- else -}}
      {{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
			{{$colAlias}} {{$colTyp.Name}} `db:"{{$table.DBTag $column}}" {{generateTags $.Tags $tagName | trim}}`
		{{- end -}}		
	{{end -}}
}

func (s {{$tAlias.UpSingular}}Setter) SetColumns() []string {
  vals := make([]string, 0, {{len $table.NonGeneratedColumns}})
	{{range $column := $table.Columns -}}
    {{if $column.Generated}}{{continue}}{{end -}}
    {{$colAlias := $tAlias.Column $column.Name -}}
		if {{$.Types.IsOptionalValid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
			vals = append(vals, {{printf "%q" $column.Name}})
		}
	{{end -}}

	return vals
}

func (s {{$tAlias.UpSingular}}Setter) Overwrite(t *{{$tAlias.UpSingular}}) {
	{{- range $column := $table.Columns -}}
    {{if $column.Generated}}{{continue}}{{end -}}
    {{$colAlias := $tAlias.Column $column.Name -}}
    {{$colTyp := $.Types.GetOptional $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
		if {{$.Types.IsOptionalValid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
      t.{{$colAlias}} = {{$.Types.FromOptional $.CurrentPackage $.Importer $column.Type (cat "s." $colAlias) $column.Nullable $column.Nullable}}
		}
	{{end -}}
}

{{block "setter_insert_mod" . -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s *{{$tAlias.UpSingular}}Setter) Apply(q *dialect.InsertQuery) {
  {{if $table.Constraints.Primary -}}
    q.AppendHooks(func(ctx context.Context, exec bob.Executor) (context.Context, error) {
      return {{$tAlias.UpPlural}}.BeforeInsertHooks.RunHooks(ctx, exec, s)
    })
  {{end}}

	q.AppendValues(
  {{range $index, $column := $table.NonGeneratedColumns -}}
    genruntime.ValueOrDefault(func() (any, bool) {
        {{$colAlias := $tAlias.Column $column.Name -}}
        {{$colGetter := $.Types.FromOptional $.CurrentPackage $.Importer $column.Type (cat "s." $colAlias) $column.Nullable $column.Nullable -}}
        if {{$.Types.IsOptionalInvalid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
          return nil, false
        }
        return {{$colGetter}}, true
    }),
  {{- end}})
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

  {{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime" }}
	{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
	{{range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
    if {{$.Types.IsOptionalValid $.CurrentPackage $column.Type $column.Nullable (cat "s." $colAlias)}} {
      exprs = append(exprs, genruntime.SetField(prefix, {{quote $column.Name}}, s.{{$colAlias}}))
		}

	{{end -}}

  return exprs
}
{{- end}}


{{- end}}
