{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range $column := .Table.Columns}}
{{$colAlias := $tAlias.Column $column.Name -}}
{{- $colTyp := "" -}}
{{- if $column.Nullable -}}
	{{- $colTyp = printf "omitnull.Val[%s]" $column.Type -}}
{{- else -}}
	{{- $colTyp = printf "omit.Val[%s]" $column.Type -}}
{{- end -}}

func (m {{$tAlias.UpSingular}}) {{$colAlias}}(val {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = val
	})
}

func (m {{$tAlias.UpSingular}}) {{$colAlias}}Func(f func() ({{$colTyp}})) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = f()
	})
}

{{end}}
