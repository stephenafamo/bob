{{define "model_and_query" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.Constraints.Primary -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Name}}", build{{$tAlias.UpSingular}}Columns({{quote $table.Key}}), {{$tAlias.DownSingular}}ScanMapper)
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Name}}", build{{$tAlias.UpSingular}}Columns({{quote $table.Key}}), {{$tAlias.DownSingular}}ScanMapper, {{$tAlias.DownSingular}}RowMapping, (*{{$tAlias.UpSingular}}Setter).Overwrite, {{$table.UniqueColPairs}})
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- end}}
{{- end}}


{{define "setter_update_mod" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) UpdateMod() bob.Mod[dialect.UpdateEditor] {
  return um.Set(s.Expressions("{{$table.Name}}")...)
}
{{- end}}

{{define "slice_update" -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (o {{$tAlias.UpSingular}}Slice) UpdateAll(ctx context.Context, exec bob.Executor, vals {{$tAlias.UpSingular}}Setter) error {
	_, err := {{$tAlias.UpPlural}}.Update(vals.UpdateMod(), o.UpdateMod()).Exec(ctx, exec)

  for i := range o {
    vals.Overwrite(o[i]) 
  }

  return err
}
{{- end}}

{{define "model/relationship/batch_key_args" -}}
{{/* Use a typed VALUES key only for matching SQL columns with their native
Go types. Otherwise compare the original parameter to the target column. */ -}}
{{$native := dict
	"char" "string" "varchar" "string" "tinyint(1)" "bool"
	"tinyint" "int8" "tinyint unsigned" "uint8"
	"smallint" "int16" "smallint unsigned" "uint16"
	"mediumint" "int16" "mediumint unsigned" "uint16"
	"int" "int32" "int unsigned" "uint32"
	"bigint" "int64" "bigint unsigned" "types.Uint64" -}}
[]mysqlrel.KeyBinding{
	{{range $i, $name := .Side.ToColumns -}}
	{{$from := $.Tables.GetColumn $.Side.From (index $.Side.FromColumns $i) -}}
	{{$to := $.Tables.GetColumn $.Side.To $name -}}
	{{$sqlType := lower $to.DBType -}}
	{{if ne $sqlType "tinyint(1)" -}}
		{{$sqlType = regexReplaceAll `\([0-9]+\)` $sqlType "" -}}
	{{end -}}
	{{if and (eq $from.DBType $to.DBType) (eq $from.Type $to.Type) (hasKey $native $sqlType) (eq $from.Type (get $native $sqlType)) -}}
		{{if eq $sqlType "char" "varchar"}}mysqlrel.Character{{else if hasSuffix " unsigned" $sqlType}}mysqlrel.Unsigned{{else}}mysqlrel.Signed{{end}},
	{{else -}}
		mysqlrel.Direct,
	{{end -}}
	{{end -}}
},
{{- end}}
