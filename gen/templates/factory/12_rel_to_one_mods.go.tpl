{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range .Table.Relationships -}}
{{- if .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} rel *{{$ftable.UpSingular}}Template) {{$tAlias.UpSingular}}Mod {
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

		{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $ftable.Relationship $invRel.Name -}}
			{{- if not .NeededColumns -}}
				{{- if $invRel.IsToMany -}}
					rel.r.{{$invAlias}} = append(rel.r.{{$invAlias}}, o)
				{{- else -}}
					rel.r.{{$invAlias}} = o
				{{- end -}}
			{{else -}}
				{{- if $invRel.IsToMany -}}
					rel.r.{{$invAlias}} =  append(rel.r.{{$invAlias}}, &{{$ftable.DownSingular}}{{$invAlias}}R{
					o: {{$tAlias.UpSingular}}TemplateSlice{o},
						{{relDependenciesTypSet $.Aliases .}}
					})
				{{- else -}}
					rel.r.{{$invAlias}} = &{{$ftable.DownSingular}}{{$invAlias}}R{
						o: o,
						{{relDependenciesTypSet $.Aliases .}}
					}
				{{- end -}}
			{{- end}}
		{{- end}}
	})
}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ *{{$ftable.UpSingular}}Template)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		m.With{{$relAlias}}(f()).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) WithNew{{$relAlias}}(f *Factory, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := f.Get{{$alias.UpSingular}}Template()
		{{- end}}
	  related := f.Get{{$ftable.UpSingular}}Template(mods...)

		m.With{{$relAlias}}({{relArgs $.Aliases .}} related).Apply(o)
	})
}

{{end}}
