{{$table := .Table}}
{{ $tAlias := .Aliases.Table .Table.Name -}}

{{range .Table.Relationships -}}
{{- if not .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
{{- $type := printf "%sTemplateSlice" $ftable.UpSingular -}}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}(related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
    for _, rel := range related {
      {{setDeps $.Importer $.Tables $.Aliases . false true false}}
    }

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      for _, rel := range related {
        {{if $invRel.IsToMany -}}
          rel.r.{{$invAlias}} = append(rel.r.{{$invAlias}}, omit.From(o))
        {{else -}}
          rel.r.{{$invAlias}} = omit.From(o)
        {{- end}}
      }
    {{- end}}

		o.r.{{$relAlias}} = omit.From(related)

		return nil
	})
}

func (m {{$tAlias.UpSingular}}) With{{$relAlias}}Func(f func() ({{$type}}, error)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
		related, err := f()
		if err != nil {
			return err
		}

		return m.With{{$relAlias}}(related).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) WithNew{{$relAlias}}(f *Factory, number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
	  if f == nil {
		  f = defaultFactory
		}

		related, err := f.Get{{$ftable.UpSingular}}TemplateSlice(number, mods...)
		if err != nil {
			return err
		}

		return m.With{{$relAlias}}(related).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) Add{{$relAlias}}(related {{$type}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
    for _, rel := range related {
      {{setDeps $.Importer $.Tables $.Aliases . false true false}}
    }

    {{if and (not $.NoBackReferencing) $invRel.Name -}}
    {{- $invAlias := $ftable.Relationship $invRel.Name -}}
      for _, rel := range related {
        {{if $invRel.IsToMany -}}
          rel.r.{{$invAlias}} = append(rel.r.{{$invAlias}}, omit.From(o))
        {{else -}}
          rel.r.{{$invAlias}} = omit.From(o)
        {{- end}}
      }
    {{- end}}

		o.r.{{$relAlias}} = omit.From(append(o.r.{{$relAlias}}.GetOrZero(), related...))

		return nil
	})
}

func (m {{$tAlias.UpSingular}}) Add{{$relAlias}}Func(f func() ({{$type}}, error)) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
		related, err := f()
		if err != nil {
			return err
		}

		return m.Add{{$relAlias}}(related).Apply(o)
	})
}

func (m {{$tAlias.UpSingular}}) AddNew{{$relAlias}}(f *Factory, number int, mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func(o *{{$tAlias.UpSingular}}Template) error {
	  if f == nil {
		  f = defaultFactory
		}

		related, err := f.Get{{$ftable.UpSingular}}TemplateSlice(number, mods...)
		if err != nil {
			return err
		}

		return m.Add{{$relAlias}}(related).Apply(o)
	})
}

{{end}}
