{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range .Table.Relationships -}}
{{- if not .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- $type := printf "...*%sTemplate" $ftable.UpSingular -}}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
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


		{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $ftable.Relationship $invRel.Name -}}
			for _, rel := range related {
			{{- if not .NeededColumns -}}
				{{- if $invRel.IsToMany -}}
					rel.r.{{$invAlias}} = append(rel.r.{{$invAlias}}, o)
				{{- else -}}
					rel.r.{{$invAlias}} = o
				{{- end -}}
			{{- else -}}
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
			}
		{{- end}}


		return nil
	})
}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ {{$ftable.UpSingular}}TemplateSlice, _ error)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
		{{relArgs $.Aliases .}} related, err := f()
		if err != nil {
			return err
		}

		return m.With{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) WithNew{{$relAlias}}(f *Factory, number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}}, err := f.Get{{$alias.UpSingular}}Template()
			if err != nil {
			return err
			}
		{{- end}}

		related, err := f.Get{{$ftable.UpSingular}}TemplateSlice(number, mods...)
		if err != nil {
			return err
		}

		return m.With{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) Add{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
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

		{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $ftable.Relationship $invRel.Name -}}
			for _, rel := range related {
			{{- if not .NeededColumns -}}
				{{- if $invRel.IsToMany -}}
					rel.r.{{$invAlias}} = append(rel.r.{{$invAlias}}, o)
				{{- else -}}
					rel.r.{{$invAlias}} = o
				{{- end -}}
			{{- else -}}
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
			}
		{{- end}}

		return nil
	})
}

func (m {{$tAlias.UpSingular}}) Add{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ {{$ftable.UpSingular}}TemplateSlice, _ error)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
		{{relArgs $.Aliases .}} related, err := f()
		if err != nil {
			return err
		}

		return m.Add{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) AddNew{{$relAlias}}(f *Factory, number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}}, err := f.Get{{$alias.UpSingular}}Template()
			if err != nil {
			return err
			}
		{{- end}}

		related, err := f.Get{{$ftable.UpSingular}}TemplateSlice(number, mods...)
		if err != nil {
			return err
		}

		return m.Add{{$relAlias}}({{relArgs $.Aliases .}} related...).Apply(o)
	})
}

{{end}}
