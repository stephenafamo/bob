{{ $tAlias := .Aliases.Table .Table.Name -}}

// {{$tAlias.UpSingular}} has methods that act as mods for the {{$tAlias.UpSingular}}Template
var {{$tAlias.UpSingular}}Mods {{$tAlias.DownSingular}}Mods
type {{$tAlias.DownSingular}}Mods struct {}

{{range $column := .Table.Columns}}
{{$colAlias := $tAlias.Column $column.Name -}}
{{- $colTyp := $column.Type -}}
{{- if $column.Nullable -}}
	{{- $.Importer.Import "github.com/aarondl/opt/null" -}}
	{{- $colTyp = printf "null.Val[%s]" $column.Type -}}
{{- end -}}

func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}(val {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = func() {{$colTyp}} { return val }
	})
}

func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}Func(f func() {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
			o.{{$colAlias}} = f
	})
}

func (m {{$tAlias.DownSingular}}Mods) Unset{{$colAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = nil
	})
}

{{end}}
