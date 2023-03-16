{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Key -}}

{{range .Table.Relationships -}}
{{- if not .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $type := printf "*%sTemplate" $ftable.UpSingular -}}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}(number int, {{relDependencies $.Aliases . "" "Template"}} related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = []*{{$tAlias.DownSingular}}R{{$relAlias}}R{ {
			number: number,
			o: related,
			{{relDependenciesTypSet $.Aliases .}}
		}}
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithNew{{$relAlias}}(number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := o.f.New{{$alias.UpSingular}}()
		{{- end}}

		related := o.f.New{{$ftable.UpSingular}}(mods...)
		m.With{{$relAlias}}(number, {{relArgs $.Aliases .}} related).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) Add{{$relAlias}}(number int, {{relDependencies $.Aliases . "" "Template"}} related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = append(o.r.{{$relAlias}}, &{{$tAlias.DownSingular}}R{{$relAlias}}R{
			number: number,
			o: related,
			{{relDependenciesTypSet $.Aliases .}}
		})
	})
}

func (m {{$tAlias.DownSingular}}Mods) AddNew{{$relAlias}}(number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := o.f.New{{$alias.UpSingular}}()
		{{- end}}

		related := o.f.New{{$ftable.UpSingular}}(mods...)
		m.Add{{$relAlias}}(number, {{relArgs $.Aliases .}} related).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) Without{{$relAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
			o.r.{{$relAlias}} = nil
	})
}

{{end}}
