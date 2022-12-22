{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range .Table.Relationships -}}
{{- if not .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- $type := printf "...*%sTemplate" $ftable.UpSingular -}}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
    {{with setFactoryDeps $.Importer $.Tables $.Aliases . true -}}
			for _, rel := range related {
        {{.}}
			}
    {{end -}}

		{{if .NeededColumns -}}
			o.r.{{$relAlias}} = []*{{$tAlias.DownSingular}}{{$relAlias}}R{ {
				o: {{$ftable.UpSingular}}TemplateSlice(related),
				{{relDependenciesTypSet $.Aliases .}}
			}}
		{{else -}}
			o.r.{{$relAlias}} = related
		{{- end}}
	})
}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ {{$ftable.UpSingular}}TemplateSlice)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{relArgs $.Aliases .}} related := f()
		m.With{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) WithNew{{$relAlias}}(f *Factory, number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := f.New{{$alias.UpSingular}}()
		{{- end}}

		related := f.New{{$ftable.UpPlural}}(number, mods...)
		m.With{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) Add{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
    {{with setFactoryDeps $.Importer $.Tables $.Aliases . true -}}
			for _, rel := range related {
        {{.}}
			}
    {{end -}}

		{{if .NeededColumns -}}
			o.r.{{$relAlias}} = append(o.r.{{$relAlias}}, &{{$tAlias.DownSingular}}{{$relAlias}}R{
				o: {{$ftable.UpSingular}}TemplateSlice(related),
				{{relDependenciesTypSet $.Aliases .}}
			})
		{{else -}}
			o.r.{{$relAlias}} = append(o.r.{{$relAlias}}, related...)
		{{- end}}
	})
}

func (m {{$tAlias.UpSingular}}) Add{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ {{$ftable.UpSingular}}TemplateSlice)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{relArgs $.Aliases .}} related := f()
		m.Add{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) AddNew{{$relAlias}}(f *Factory, number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := f.New{{$alias.UpSingular}}()
		{{- end}}

		related := f.New{{$ftable.UpPlural}}(number, mods...)
		m.Add{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

{{end}}
