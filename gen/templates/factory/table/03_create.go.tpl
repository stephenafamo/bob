{{if .Table.Constraints.Primary}}
{{$.Importer.Import "models" $.ModelsPackage}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/twitter-payments/bob"}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key}}

func ensureCreatable{{$tAlias.UpSingular}}(m *models.{{$tAlias.UpSingular}}Setter) {
	{{range $column := $table.Columns -}}
  {{- if $column.Default}}{{continue}}{{end -}}
  {{- if $column.Nullable}}{{continue}}{{end -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
  {{- $typDef :=  index $.Types $column.Type -}}
  {{- $colTyp := or $typDef.AliasOf $column.Type -}}
		if m.{{$colAlias}}.IsUnset() {
        m.{{$colAlias}} = omit.From(random_{{normalizeType $column.Type}}(nil))
    }
	{{end -}}
}

// insertOptRels creates and inserts any optional the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. 
// any required relationship should have already exist on the model
func (o *{{$tAlias.UpSingular}}Template) insertOptRels(ctx context.Context, exec bob.Executor, m *models.{{$tAlias.UpSingular}}) (context.Context,error) {
	var err error

	{{range $index, $rel := $.Relationships.Get $table.Key -}}{{if not ($.Tables.RelIsView $rel) -}}
		{{- if ($table.RelIsRequired $rel)}}{{continue}}{{end -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		{{- $invRel := $.Relationships.GetInverse . -}}
		{{- $ftable := $.Aliases.Table $rel.Foreign -}}
		{{- $invAlias := "" -}}
    {{- if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias = $ftable.Relationship $invRel.Name -}}
		{{- end -}}

		if o.r.{{$relAlias}} != nil {
		{{- if .IsToMany -}}
				for _, r := range o.r.{{$relAlias}} {
          {{- range $.Tables.NeededBridgeRels . -}}
						{{$alias := $.Aliases.Table .Table -}}
            {{if not .Many}}
              var {{$alias.DownSingular}}{{.Position}} *models.{{$alias.UpSingular}}
              ctx, {{$alias.DownSingular}}{{.Position}}, err = r.{{$alias.DownSingular}}.create(ctx, exec)
            {{else}}
              var {{$alias.DownSingular}}{{.Position}} models.{{$alias.UpSingular}}Slice
              ctx, {{$alias.DownSingular}}{{.Position}}, err = r.{{$alias.DownSingular}}.createMany(ctx, exec, r.number)
            {{end}}
						if err != nil {
							return ctx, err
						}
					{{end -}}

					var rel{{$index}} models.{{$ftable.UpSingular}}Slice
					ctx, rel{{$index}}, err = r.o.createMany(ctx, exec, r.number)
					if err != nil {
						return ctx, err
					}

					err = m.Attach{{$relAlias}}(ctx, exec, {{$.Tables.RelArgs $.Aliases $rel}} rel{{$index}}...)
					if err != nil {
						return ctx, err
					}
				}
		{{- else -}}
      {{- range $.Tables.NeededBridgeRels . -}}
				{{$alias := $.Aliases.Table .Table -}}
        {{if not .Many}}
          var {{$alias.DownSingular}}{{.Position}} *models.{{$alias.UpSingular}}
          ctx, {{$alias.DownSingular}}{{.Position}}, err = r.{{$alias.DownSingular}}.create(ctx, exec)
        {{else}}
          var {{$alias.DownSingular}}{{.Position}} models.{{$alias.UpSingular}}Slice
          ctx, {{$alias.DownSingular}}{{.Position}}, err = r.{{$alias.DownSingular}}.createMany(ctx, exec, r.number)
        {{end}}
				if err != nil {
					return ctx, err
				}
			{{end -}}

			var rel{{$index}} *models.{{$ftable.UpSingular}}
			ctx, rel{{$index}}, err = o.r.{{$relAlias}}.o.create(ctx, exec)
			if err != nil {
				return ctx, err
			}
			err = m.Attach{{$relAlias}}(ctx, exec, {{$.Tables.RelArgs $.Aliases $rel}} rel{{$index}})
			if err != nil {
				return ctx, err
			}
		{{end -}}
		}

	{{end}}{{end}}

	return ctx, err
}


// Create builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
func (o *{{$tAlias.UpSingular}}Template) Create(ctx context.Context, exec bob.Executor) (*models.{{$tAlias.UpSingular}}, error) {
  _, m, err := o.create(ctx, exec)
	return m, err
}

// MustCreate builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
// panics if an error occurs
func (o *{{$tAlias.UpSingular}}Template) MustCreate(ctx context.Context, exec bob.Executor) *models.{{$tAlias.UpSingular}} {
  _, m, err := o.create(ctx, exec)
  if err != nil {
    panic(err)
  }
	return m
}

// CreateOrFail builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
// It calls `tb.Fatal(err)` on the test/benchmark if an error occurs
func (o *{{$tAlias.UpSingular}}Template) CreateOrFail(ctx context.Context, tb testing.TB, exec bob.Executor) *models.{{$tAlias.UpSingular}} {
  tb.Helper()
  _, m, err := o.create(ctx, exec)
  if err != nil {
    tb.Fatal(err)
    return nil
  }
	return m
}



// create builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
// this returns a context that includes the newly inserted model
func (o *{{$tAlias.UpSingular}}Template) create(ctx context.Context, exec bob.Executor) (context.Context, *models.{{$tAlias.UpSingular}}, error) {
	var err error
	opt := o.BuildSetter()
	ensureCreatable{{$tAlias.UpSingular}}(opt)

	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- if not ($table.RelIsRequired $rel)}}{{continue}}{{end -}}
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

	m, err := models.{{$tAlias.UpPlural}}.Insert(opt).One(ctx, exec)
	if err != nil {
	  return ctx, nil, err
	}
	ctx = {{$tAlias.DownSingular}}Ctx.WithValue(ctx, m)


	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- if not ($table.RelIsRequired $rel) -}}{{continue}}{{end -}}
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

// MustCreateMany builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
// panics if an error occurs
func (o {{$tAlias.UpSingular}}Template) MustCreateMany(ctx context.Context, exec bob.Executor, number int) models.{{$tAlias.UpSingular}}Slice {
  _, m, err := o.createMany(ctx, exec, number)
  if err != nil {
    panic(err)
  }
	return m
}

// CreateManyOrFail builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
// It calls `tb.Fatal(err)` on the test/benchmark if an error occurs
func (o {{$tAlias.UpSingular}}Template) CreateManyOrFail(ctx context.Context, tb testing.TB, exec bob.Executor, number int) models.{{$tAlias.UpSingular}}Slice {
  tb.Helper()
  _, m, err := o.createMany(ctx, exec, number)
  if err != nil {
    tb.Fatal(err)
    return nil
  }
	return m
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
