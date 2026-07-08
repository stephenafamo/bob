{{$table := .Table}}
{{ $tAlias := .Aliases.Table $table.Key -}}

func (m {{$tAlias.DownSingular}}Mods) WithParentsCascading() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
    if isDone, _ := {{$tAlias.DownSingular}}WithParentsCascadingCtx.Value(ctx); isDone {
      return
    }
    ctx = {{$tAlias.DownSingular}}WithParentsCascadingCtx.WithValue(ctx, true)
    {{range $.Relationships.Get $table.Key -}}
    {{- if .IsToMany -}}{{continue}}{{end -}}
    {{- $rel := . -}}
    {{- $ftable := $.Aliases.Table .Foreign -}}
    {{- $relAlias := $tAlias.Relationship .Name -}}
    {{ if not ($table.RelIsRequired .) }}
    if false{{range $side := $rel.ValuedSides -}}
      {{- if ne $side.TableName $table.Key}}{{continue}}{{end -}}
      {{- range $mapping := $side.Mapped -}}
        {{- if ne $mapping.ExternalTable $rel.Foreign}}{{continue}}{{end -}}
        || o.{{index $tAlias.Columns $mapping.Column}} != nil
      {{- end -}}
    {{- end}} {
    {{- end}}
    {
      {{range $.AllTables.NeededBridgeRels . -}}
        {{$alias := $.Aliases.Table .Table -}}
        {{$alias.DownSingular}}{{.Position}} := o.f.New{{$alias.UpSingular}}WithContext(ctx)
      {{end}}
      related := o.f.New{{$ftable.UpSingular}}WithContext(ctx, {{$.FactoryModsVar .Foreign}}.WithParentsCascading())
      m.With{{$relAlias}}({{$.AllTables.RelArgs $.Aliases .}} related).Apply(ctx, o)
    }
    {{ if not ($table.RelIsRequired .)}}
    }
    {{ end }}
    {{end -}}
	})
}

{{range $.Relationships.Get $table.Key -}}
{{- if .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}({{$.FactoryRelDependencies .}} rel *{{$.FactoryTemplateType .Foreign}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = &{{$tAlias.DownSingular}}R{{$relAlias}}R{
			o: rel,
			{{$.AllTables.RelDependenciesTypSet $.Aliases .}}
		}
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithNew{{$relAlias}}(mods ...{{$.FactoryModType .Foreign}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
		{{range $.AllTables.NeededBridgeRels . -}}
			{{$alias := $.Aliases.Table .Table -}}
			{{$alias.DownSingular}}{{.Position}} := o.f.New{{$alias.UpSingular}}WithContext(ctx)
		{{end}}
	  related := o.f.New{{$ftable.UpSingular}}WithContext(ctx, mods...)

		m.With{{$relAlias}}({{$.AllTables.RelArgs $.Aliases .}} related).Apply(ctx, o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithExisting{{$relAlias}}(em *models.{{$ftable.UpSingular}}) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = &{{$tAlias.DownSingular}}R{{$relAlias}}R{
			o: o.f.FromExisting{{$ftable.UpSingular}}(ctx, em),
		}
	})
}

func (m {{$tAlias.DownSingular}}Mods) Without{{$relAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
			o.r.{{$relAlias}} = nil
	})
}

{{end}}
