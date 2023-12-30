{{$table := .Table}}
{{ $tAlias := .Aliases.Table $table.Key -}}

{{range $.Relationships.Get $table.Key -}}
{{- if .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}({{relDependencies $.Tables $.Aliases . "" "Template"}} rel *{{$ftable.UpSingular}}Template) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = &{{$tAlias.DownSingular}}R{{$relAlias}}R{
			o: rel,
			{{relDependenciesTypSet $.Tables $.Aliases .}}
		}
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithNew{{$relAlias}}(mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{range neededBridgeRels $.Tables $.Aliases . -}}
			{{$alias := $.Aliases.Table .Table -}}
			{{$alias.DownSingular}}{{.Position}} := o.f.New{{$alias.UpSingular}}()
		{{end}}
	  related := o.f.New{{$ftable.UpSingular}}(mods...)

		m.With{{$relAlias}}({{relArgs $.Tables $.Aliases .}} related).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) Without{{$relAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
			o.r.{{$relAlias}} = nil
	})
}

{{end}}
