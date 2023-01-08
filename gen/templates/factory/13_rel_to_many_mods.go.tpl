{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Key -}}

{{range .Table.Relationships -}}
{{- if not .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $type := printf "*%sTemplate" $ftable.UpSingular -}}

func (m {{$tAlias.DownSingular}}Mods) Without{{$relAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
			o.r.{{$relAlias}} = nil
	})
}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} related {{$type}}, number int) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = []*{{$tAlias.DownSingular}}{{$relAlias}}R{ {
			number: number,
			o: related,
			{{relDependenciesTypSet $.Aliases .}}
		}}
	})
}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ {{$type}}), number int) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{relArgs $.Aliases .}} related := f()
		m.With{{$relAlias}}({{relArgs $.Aliases .}} related, number).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithNew{{$relAlias}}(number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
	  f := o.f
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := f.New{{$alias.UpSingular}}()
		{{- end}}

		related := f.New{{$ftable.UpSingular}}(mods...)
		m.With{{$relAlias}}({{relArgs $.Aliases .}} related, number).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) Add{{$relAlias}}({{relDependencies $.Aliases . "" "Template"}} related {{$type}}, number int) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = append(o.r.{{$relAlias}}, &{{$tAlias.DownSingular}}{{$relAlias}}R{
			number: number,
			o: related,
			{{relDependenciesTypSet $.Aliases .}}
		})
	})
}

func (m {{$tAlias.DownSingular}}Mods) Add{{$relAlias}}Func(f func() ({{relDependencies $.Aliases . "" "Template"}} _ *{{$ftable.UpSingular}}Template), number int) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
		{{relArgs $.Aliases .}} related := f()
		m.Add{{$relAlias}}({{relArgs $.Aliases .}} related, number).Apply(o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) AddNew{{$relAlias}}(number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) {
	  f := o.f
	  if f == nil {
		  f = defaultFactory
		}

		{{range .NeededColumns -}}
			{{$alias := $.Aliases.Table . -}}
			{{$alias.DownSingular}} := f.New{{$alias.UpSingular}}()
		{{- end}}

		related := f.New{{$ftable.UpSingular}}(mods...)
		m.Add{{$relAlias}}({{relArgs $.Aliases .}} related, number).Apply(o)
	})
}

{{end}}
