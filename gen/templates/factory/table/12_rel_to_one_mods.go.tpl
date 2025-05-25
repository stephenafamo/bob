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
    {{- $ftable := $.Aliases.Table .Foreign -}}
    {{- $relAlias := $tAlias.Relationship .Name -}}
    {
      {{range $.Tables.NeededBridgeRels . -}}
        {{$alias := $.Aliases.Table .Table -}}
        {{$alias.DownSingular}}{{.Position}} := o.f.New{{$alias.UpSingular}}(ctx)
      {{end}}
      related := o.f.New{{$ftable.UpSingular}}(ctx, {{$ftable.UpSingular}}Mods.WithParentsCascading())
      m.With{{$relAlias}}({{$.Tables.RelArgs $.Aliases .}} related).Apply(ctx, o)
    }
    {{end -}}
	})
}

{{range $.Relationships.Get $table.Key -}}
{{- if .IsToMany -}}{{continue}}{{end -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}

func (m {{$tAlias.DownSingular}}Mods) With{{$relAlias}}({{$.Tables.RelDependencies $.Aliases . "" "Template"}} rel *{{$ftable.UpSingular}}Template) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
		o.r.{{$relAlias}} = &{{$tAlias.DownSingular}}R{{$relAlias}}R{
			o: rel,
			{{$.Tables.RelDependenciesTypSet $.Aliases .}}
		}
	})
}

func (m {{$tAlias.DownSingular}}Mods) WithNew{{$relAlias}}(mods ...{{$ftable.UpSingular}}Mod) {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
		{{range $.Tables.NeededBridgeRels . -}}
			{{$alias := $.Aliases.Table .Table -}}
			{{$alias.DownSingular}}{{.Position}} := o.f.New{{$alias.UpSingular}}(ctx)
		{{end}}
	  related := o.f.New{{$ftable.UpSingular}}(ctx, mods...)

		m.With{{$relAlias}}({{$.Tables.RelArgs $.Aliases .}} related).Apply(ctx, o)
	})
}

func (m {{$tAlias.DownSingular}}Mods) Without{{$relAlias}}() {{$tAlias.UpSingular}}Mod {
	return {{$tAlias.UpSingular}}ModFunc(func (ctx context.Context, o *{{$tAlias.UpSingular}}Template) {
			o.r.{{$relAlias}} = nil
	})
}

{{end}}
