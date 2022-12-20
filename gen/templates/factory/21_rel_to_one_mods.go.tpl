{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range .Table.Relationships -}}
{{- if .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}(rel *{{$ftable.UpSingular}}Template) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
    {{setDeps $.Importer $.Tables $.Aliases . false false false}}

		o.r.{{$relAlias}} = omit.From(rel)

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      {{if $invRel.IsToMany -}}
        o.r.{{$relAlias}}.MustGet().r.{{$invAlias}} = omit.From({{$tAlias.UpSingular}}TemplateSlice{o})
      {{else -}}
        o.r.{{$relAlias}}.MustGet().r.{{$invAlias}} = omit.From(o)
      {{- end}}
    {{- end}}

		return nil
	})
}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}Func(f func() (*{{$ftable.UpSingular}}Template, error)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
		related, err := f()
		if err != nil {
			return err
		}

		return m.With{{$relAlias}}(related).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) WithNew{{$relAlias}}(f *Factory, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
	  if f == nil {
		  f = defaultFactory
		}

	  related, err := f.Get{{$ftable.UpSingular}}Template(mods...)
		if err != nil {
			return err
		}

		return m.With{{$relAlias}}(related).Apply(o)
	})
}

{{end}}
