{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range .Table.Relationships -}}
{{- if .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} rel *{{$ftable.UpSingular}}Template) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
    {{setFactoryDeps $.Importer $.Tables $.Aliases . false}}

		{{if  .NeededColumns -}}
			o.r.{{$relAlias}} = &{{$tAlias.DownSingular}}{{$relAlias}}R{
				o: rel,
				{{relDependenciesTypSet $.Aliases .}}
			}
		{{else -}}
			o.r.{{$relAlias}} = rel
		{{- end}}
	})
}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ *{{$ftable.UpSingular}}Template)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		m.With{{$relAlias}}(f()).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithNew{{$relAlias}}(f *Factory, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := f.New{{$alias.UpSingular}}()
		{{- end}}
	  related := f.New{{$ftable.UpSingular}}(mods...)

		m.With{{$relAlias}}({{relArgs $.Aliases .}} related).Apply(o)
	})
}

{{end}}
