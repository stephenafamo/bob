{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}

var {{$tAlias.DownSingular}}ScanMapper = genruntime.ModelMapper(func(name string) func(*{{$tAlias.UpSingular}}) any {
	switch name {
	{{range $column := $table.Columns -}}
	case {{quote $column.Name}}:
		return func(o *{{$tAlias.UpSingular}}) any { return &o.{{$tAlias.Column $column.Name}} }
	{{end -}}
	default: return nil
	}
})
