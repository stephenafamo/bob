{{ $tAlias := .Aliases.Table .Table.Key -}}

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
		o.{{$colAlias}} = func(*faker.Faker) {{$colTyp}} { return val }
	})
}

func (m {{$tAlias.DownSingular}}Mods) {{$colAlias}}Func(f func(*faker.Faker) {{$colTyp}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
			o.{{$colAlias}} = f
	})
}

func (m {{$tAlias.DownSingular}}Mods) Unset{{$colAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = nil
	})
}

func (m {{$tAlias.DownSingular}}Mods) Random{{$colAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.{{$colAlias}} = func(f *faker.Faker) {{$colTyp}} {
			{{if $column.Nullable -}}
				return RandomNull[{{$column.Type}}](f)
			{{- else -}}
				return Random[{{$column.Type}}](f)
			{{- end}}
		}
	})
}

{{end}}
