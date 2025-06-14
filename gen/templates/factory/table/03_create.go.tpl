{{if .Table.Constraints.Primary}}
{{$.Importer.Import "testing"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key}}

func ensureCreatable{{$tAlias.UpSingular}}(m *models.{{$tAlias.UpSingular}}Setter) {
  {{range $column := $table.Columns -}}
    {{- if $column.Default}}{{continue}}{{end -}}
    {{- if $column.Nullable}}{{continue}}{{end -}}
    {{- if $column.Generated}}{{continue}}{{end -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{- $colGetter := $.Types.ToOptional $.CurrentPackage $.Importer $column.Type "val" $column.Nullable $column.Nullable -}}
    {{- $typDef :=  $.Types.Index $column.Type -}}
    {{- $colTyp := or $typDef.AliasOf $column.Type -}}
    if !({{$.Types.IsOptionalValid $.CurrentPackage $column.Type $column.Nullable (cat "m." $colAlias)}}) {
      val := random_{{normalizeType $column.Type}}(nil, {{$column.LimitsString}})
      m.{{$colAlias}} = {{$colGetter}}
    }
  {{end -}}
}

// insertOptRels creates and inserts any optional the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. 
// any required relationship should have already exist on the model
func (o *{{$tAlias.UpSingular}}Template) insertOptRels(ctx context.Context, exec bob.Executor, m *models.{{$tAlias.UpSingular}}) (error) {
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

    is{{$relAlias}}Done, _ := {{$tAlias.DownSingular}}Rel{{$relAlias}}Ctx.Value(ctx)
		if !is{{$relAlias}}Done && o.r.{{$relAlias}} != nil {
        ctx = {{$tAlias.DownSingular}}Rel{{$relAlias}}Ctx.WithValue(ctx, true);
		{{- if .IsToMany -}}
				for _, r := range o.r.{{$relAlias}} {
          {{- range $.Tables.NeededBridgeRels . -}}
						{{$alias := $.Aliases.Table .Table -}}
            {{if not .Many}}
              {{$alias.DownSingular}}{{.Position}}, err := r.{{$alias.DownSingular}}.Create(ctx, exec)
            {{else}}
              {{$alias.DownSingular}}{{.Position}}, err := r.{{$alias.DownSingular}}.CreateMany(ctx, exec, r.number)
            {{end}}
						if err != nil {
							return err
						}
					{{end -}}

					rel{{$index}}, err := r.o.CreateMany(ctx, exec, r.number)
					if err != nil {
						return err
					}

					err = m.Attach{{$relAlias}}(ctx, exec, {{$.Tables.RelArgs $.Aliases $rel}} rel{{$index}}...)
					if err != nil {
						return err
					}
				}
		{{- else -}}
      {{- range $.Tables.NeededBridgeRels . -}}
				{{$alias := $.Aliases.Table .Table -}}
        {{if not .Many}}
          {{$alias.DownSingular}}{{.Position}}, err := r.{{$alias.DownSingular}}.Create(ctx, exec)
        {{else}}
          {{$alias.DownSingular}}{{.Position}}, err := r.{{$alias.DownSingular}}.CreateMany(ctx, exec, r.number)
        {{end}}
				if err != nil {
					return err
				}
			{{end -}}

			var rel{{$index}} *models.{{$ftable.UpSingular}}
			rel{{$index}}, err = o.r.{{$relAlias}}.o.Create(ctx, exec)
			if err != nil {
				return err
			}
			err = m.Attach{{$relAlias}}(ctx, exec, {{$.Tables.RelArgs $.Aliases $rel}} rel{{$index}})
			if err != nil {
				return err
			}
		{{end}}
		}

	{{end}}{{end}}

	return err
}


// Create builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
func (o *{{$tAlias.UpSingular}}Template) Create(ctx context.Context, exec bob.Executor) (*models.{{$tAlias.UpSingular}}, error) {
	var err error
	opt := o.BuildSetter()
	ensureCreatable{{$tAlias.UpSingular}}(opt)

	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- if not ($table.RelIsRequired $rel)}}{{continue}}{{end -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		if o.r.{{$relAlias}} == nil {
      {{$tAlias.UpSingular}}Mods.WithNew{{$relAlias}}().Apply(ctx, o)
		}

    rel{{$index}}, err := o.r.{{$relAlias}}.o.Create(ctx, exec)
    if err != nil {
      return nil, err
    }

		{{range $rel.ValuedSides -}}
			{{- if ne .TableName $table.Key}}{{continue}}{{end -}}
			{{range .Mapped}}
				{{- if ne .ExternalTable $rel.Foreign}}{{continue}}{{end -}}
				{{- $fromColA := index $tAlias.Columns .Column -}}
				{{- $relIndex := printf "rel%d" $index -}}
				opt.{{$fromColA}} = {{$.Tables.ColumnAssigner $.CurrentPackage $.Importer $.Types $.Aliases $.Table.Key $rel.Foreign .Column .ExternalColumn $relIndex true}}
			{{end}}
		{{- end}}
	{{end}}

	m, err := models.{{$tAlias.UpPlural}}.Insert(opt).One(ctx, exec)
	if err != nil {
	  return nil, err
	}

	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- if not ($table.RelIsRequired $rel) -}}{{continue}}{{end -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		m.R.{{$relAlias}} = rel{{$index}}
	{{end}}

  if err := o.insertOptRels(ctx, exec, m); err != nil {
    return nil, err
  }
	return m, err
}

// MustCreate builds a {{$tAlias.DownSingular}} and inserts it into the database
// Relations objects are also inserted and placed in the .R field
// panics if an error occurs
func (o *{{$tAlias.UpSingular}}Template) MustCreate(ctx context.Context, exec bob.Executor) *models.{{$tAlias.UpSingular}} {
  m, err := o.Create(ctx, exec)
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
  m, err := o.Create(ctx, exec)
  if err != nil {
    tb.Fatal(err)
    return nil
  }
	return m
}


// CreateMany builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
func (o {{$tAlias.UpSingular}}Template) CreateMany(ctx context.Context, exec bob.Executor, number int) (models.{{$tAlias.UpSingular}}Slice, error) {
	var err error
	m := make(models.{{$tAlias.UpSingular}}Slice, number)

	for i := range m {
	  m[i], err = o.Create(ctx, exec)
		if err != nil {
			return nil, err
		}
	}

	return m, nil
}

// MustCreateMany builds multiple {{$tAlias.DownPlural}} and inserts them into the database
// Relations objects are also inserted and placed in the .R field
// panics if an error occurs
func (o {{$tAlias.UpSingular}}Template) MustCreateMany(ctx context.Context, exec bob.Executor, number int) models.{{$tAlias.UpSingular}}Slice {
  m, err := o.CreateMany(ctx, exec, number)
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
  m, err := o.CreateMany(ctx, exec, number)
  if err != nil {
    tb.Fatal(err)
    return nil
  }
	return m
}
{{end}}
