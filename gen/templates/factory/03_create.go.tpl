{{if .Table.PKey}}
{{$.Importer.Import "models" $.ModelsPackage}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Name}}

// insert{{$tAlias.UpSingular}}Relationships creates and inserts the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. 
// one-relationships that already exist on the model are skipped
// many-relationships that already exist on the model are added to
func (f *Factory) insertOpt{{$tAlias.UpSingular}}Rels(ctx context.Context, exec bob.Executor, o *{{$tAlias.UpSingular}}Template, m *models.{{$tAlias.UpSingular}}) (context.Context,error) {
	var err error

	{{range $index, $rel := .Table.Relationships -}}
		{{- if (relIsRequired $table $rel)}}{{continue}}{{end -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
		{{- $ftable := $.Aliases.Table $rel.Foreign -}}
		{{- $invAlias := "" -}}
    {{- if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias = $ftable.Relationship $invRel.Name -}}
		{{- end -}}

		if o.r.{{$relAlias}} != nil {
		{{- if not .NeededColumns -}}
			{{if not .IsToMany}}
				var rel{{$index}} *models.{{$ftable.UpSingular}}
				ctx, rel{{$index}}, err = f.Create{{$ftable.UpSingular}}(ctx, exec, o.r.{{$relAlias}})
				if err != nil {
					return ctx, err
				}
				err = m.Attach{{$relAlias}}(ctx, exec, rel{{$index}})
				if err != nil {
					return ctx, err
				}
			{{else}}
				var rel{{$index}} models.{{$ftable.UpSingular}}Slice
				ctx, rel{{$index}}, err = f.Create{{$ftable.UpPlural}}(ctx, exec, o.r.{{$relAlias}}...)
				if err != nil {
					return ctx, err
				}
				err = m.Attach{{$relAlias}}(ctx, exec, rel{{$index}}...)
				if err != nil {
					return ctx, err
				}
			{{end}}
		{{- else -}}
			{{if not .IsToMany}}
				{{- range .NeededColumns -}}
					{{$alias := $.Aliases.Table . -}}
					var {{$alias.DownSingular}} *models.{{$alias.UpSingular}}
					ctx, {{$alias.DownSingular}}, err = f.Create{{$alias.UpSingular}}(ctx, exec, o.r.{{$relAlias}}.{{$alias.DownSingular}})
					if err != nil {
						return ctx, err
					}
				{{- end}}

				var rel{{$index}} *models.{{$ftable.UpSingular}}
				ctx, rel{{$index}}, err = f.Create{{$ftable.UpSingular}}(ctx, exec, o.r.{{$relAlias}}.o)
				if err != nil {
					return ctx, err
				}
				err = m.Attach{{$relAlias}}(ctx, exec, {{relArgs $.Aliases $rel}} rel{{$index}})
				if err != nil {
					return ctx, err
				}
			{{else}}
				for _, r := range o.r.{{$relAlias}} {
					{{- range .NeededColumns -}}
						{{$alias := $.Aliases.Table . -}}
						var {{$alias.DownSingular}} *models.{{$alias.UpSingular}}
						ctx, {{$alias.DownSingular}}, err = f.Create{{$alias.UpSingular}}(ctx, exec, r.{{$alias.DownSingular}})
						if err != nil {
							return ctx, err
						}
					{{- end}}

					var rel{{$index}} models.{{$ftable.UpSingular}}Slice
					ctx, rel{{$index}}, err = f.Create{{$ftable.UpPlural}}(ctx, exec, r.o...)
					if err != nil {
						return ctx, err
					}

					err = m.Attach{{$relAlias}}(ctx, exec, {{relArgs $.Aliases $rel}} rel{{$index}}...)
					if err != nil {
						return ctx, err
					}
				}
			{{end}}
		{{end -}}
		}

	{{end}}

	return ctx, err
}


// Create builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
func (f *Factory) Create{{$tAlias.UpSingular}}(ctx context.Context, exec bob.Executor, o *{{$tAlias.UpSingular}}Template) (context.Context, *models.{{$tAlias.UpSingular}}, error) {
	var err error
	opt := o.BuildOptional()

	{{range $index, $rel := $table.Relationships -}}
		{{- if not (relIsRequired $table $rel)}}{{continue}}{{end -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		var rel{{$index}} *models.{{$ftable.UpSingular}}
		if o.r.{{$relAlias}} == nil {
			var ok bool
			rel{{$index}}, ok = {{$ftable.DownSingular}}Ctx.Value(ctx)
			if !ok {
				{{$tAlias.UpSingular}}Mods.WithNew{{$relAlias}}(f).Apply(o)
			}
		}
		if o.r.{{$relAlias}} != nil {
			ctx, rel{{$index}}, err = f.Create{{$ftable.UpSingular}}(ctx, exec, o.r.{{$relAlias}})
			if err != nil {
				return ctx, nil, err
			}
		}
		{{range $rel.ValuedSides -}}
			{{- if ne .TableName $table.Name}}{{continue}}{{end -}}
			{{range .Mapped}}
				{{- if ne .ExternalTable $rel.Foreign}}{{continue}}{{end -}}
				{{- $fromColA := index $tAlias.Columns .Column -}}
				{{- $toColA := index $ftable.Columns .ExternalColumn -}}
				opt.{{$fromColA}} = omit.From(rel{{$index}}.{{$toColA}})
			{{end}}
		{{- end}}
	{{end}}

	m, err := models.{{$tAlias.UpPlural}}Table.Insert(ctx, exec, opt)
	if err != nil {
	  return ctx, nil, err
	}
	ctx = {{$tAlias.DownSingular}}Ctx.WithValue(ctx, m)


	{{range $index, $rel := $table.Relationships -}}
		{{- if not (relIsRequired $table $rel) -}}{{continue}}{{end -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		m.R.{{$relAlias}} = rel{{$index}}
	{{end}}

  ctx, err = f.insertOpt{{$tAlias.UpSingular}}Rels(ctx, exec, o, m)
	return ctx, m, err
}

// Create builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
func (f *Factory) Create{{$tAlias.UpPlural}}(ctx context.Context, exec bob.Executor, o ...*{{$tAlias.UpSingular}}Template) (context.Context, models.{{$tAlias.UpSingular}}Slice, error) {
	var err error
	m := make(models.{{$tAlias.UpSingular}}Slice, len(o))

	for i, o := range o {
	  ctx, m[i], err = f.Create{{$tAlias.UpSingular}}(ctx, exec, o)
		if err != nil {
			return ctx, nil, err
		}
	}

	return ctx, m, nil
}


{{end}}
