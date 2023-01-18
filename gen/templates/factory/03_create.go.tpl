{{if .Table.PKey}}
{{$.Importer.Import "models" $.ModelsPackage}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Key}}

// insertOptRels creates and inserts any optional the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. 
// any required relationship should have already exist on the model
func (o *{{$tAlias.UpSingular}}Template) insertOptRels(ctx context.Context, exec bob.Executor, m *models.{{$tAlias.UpSingular}}) (context.Context,error) {
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
		{{- if .IsToMany -}}
				for _, r := range o.r.{{$relAlias}} {
					{{- range .NeededColumns -}}
						{{$alias := $.Aliases.Table . -}}
						var {{$alias.DownSingular}} *models.{{$alias.UpSingular}}
						ctx, {{$alias.DownSingular}}, err = r.{{$alias.DownSingular}}.create(ctx, exec)
						if err != nil {
							return ctx, err
						}
					{{end -}}

					var rel{{$index}} models.{{$ftable.UpSingular}}Slice
					ctx, rel{{$index}}, err = r.o.createMany(ctx, exec, r.number)
					if err != nil {
						return ctx, err
					}

					err = m.Attach{{$relAlias}}(ctx, exec, {{relArgs $.Aliases $rel}} rel{{$index}}...)
					if err != nil {
						return ctx, err
					}
				}
		{{- else -}}
			{{- range .NeededColumns -}}
				{{$alias := $.Aliases.Table . -}}
				var {{$alias.DownSingular}} *models.{{$alias.UpSingular}}
				ctx, {{$alias.DownSingular}}, err = o.r.{{$relAlias}}.{{$alias.DownSingular}}.create(ctx, exec)
				if err != nil {
					return ctx, err
				}
			{{end -}}

			var rel{{$index}} *models.{{$ftable.UpSingular}}
			ctx, rel{{$index}}, err = o.r.{{$relAlias}}.o.create(ctx, exec)
			if err != nil {
				return ctx, err
			}
			err = m.Attach{{$relAlias}}(ctx, exec, {{relArgs $.Aliases $rel}} rel{{$index}})
			if err != nil {
				return ctx, err
			}
		{{end -}}
		}

	{{end}}

	return ctx, err
}


// Create builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
func (o *{{$tAlias.UpSingular}}Template) Create(ctx context.Context, exec bob.Executor) (*models.{{$tAlias.UpSingular}}, error) {
  _, m, err := o.create(ctx, exec)
	return m, err
}


// create builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
// this returns a context that includes the newly inserted model
func (o *{{$tAlias.UpSingular}}Template) create(ctx context.Context, exec bob.Executor) (context.Context, *models.{{$tAlias.UpSingular}}, error) {
	var err error
	opt := o.BuildSetter()

	{{range $index, $rel := $table.Relationships -}}
		{{- if not (relIsRequired $table $rel)}}{{continue}}{{end -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		var rel{{$index}} *models.{{$ftable.UpSingular}}
		if o.r.{{$relAlias}} == nil {
			var ok bool
			rel{{$index}}, ok = {{$ftable.DownSingular}}Ctx.Value(ctx)
			if !ok {
				{{$tAlias.UpSingular}}Mods.WithNew{{$relAlias}}().Apply(o)
			}
		}
		if o.r.{{$relAlias}} != nil {
			ctx, rel{{$index}}, err = o.r.{{$relAlias}}.o.create(ctx, exec)
			if err != nil {
				return ctx, nil, err
			}
		}
		{{range $rel.ValuedSides -}}
			{{- if ne .TableName $table.Key}}{{continue}}{{end -}}
			{{range .Mapped}}
				{{- if ne .ExternalTable $rel.Foreign}}{{continue}}{{end -}}
				{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
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

  ctx, err = o.insertOptRels(ctx, exec, m)
	return ctx, m, err
}


// CreateMany builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
func (o {{$tAlias.UpSingular}}Template) CreateMany(ctx context.Context, exec bob.Executor, number int) (models.{{$tAlias.UpSingular}}Slice, error) {
  _, m, err := o.createMany(ctx, exec, number)
	return m, err
}


// createMany builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
// this returns a context that includes the newly inserted models
func (o {{$tAlias.UpSingular}}Template) createMany(ctx context.Context, exec bob.Executor, number int) (context.Context, models.{{$tAlias.UpSingular}}Slice, error) {
	var err error
	m := make(models.{{$tAlias.UpSingular}}Slice, number)

	for i := range m {
	  ctx, m[i], err = o.create(ctx, exec)
		if err != nil {
			return ctx, nil, err
		}
	}

	return ctx, m, nil
}

{{end}}
