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
    if {{$.Types.IsOptionalInvalid $.CurrentPackage $column.Type $column.Nullable (cat "m." $colAlias)}} {
      val := random_{{normalizeType $column.Type}}(nil, {{$column.LimitsString}})
      m.{{$colAlias}} = {{$colGetter}}
    }
  {{end -}}
}

{{range $rel := $.Relationships.Get $table.Key -}}
{{with and (not ($.Tables.RelIsView $rel)) ($.Tables.NeededBridgeRels $rel) ($.Relationships.FactoryBridgeParents $rel) -}}
{{$bridge := $.Aliases.Table .Table -}}
{{$localRel := $bridge.Relationship .Local -}}
{{$foreignRel := $bridge.Relationship .Foreign -}}
func (r *{{$tAlias.DownSingular}}R{{$tAlias.Relationship $rel.Name}}R) createBridge(ctx context.Context, exec bob.Executor, local *models.{{$tAlias.UpSingular}}, foreign *models.{{($.Aliases.Table $rel.Foreign).UpSingular}}) error {
	// Bind the endpoints on a copy so the configured bridge template can be reused.
	bridge := *r.{{$bridge.DownSingular}}
	bridge.r.{{$localRel}} = &{{$bridge.DownSingular}}R{{$localRel}}R{boundParent: local}
	bridge.r.{{$foreignRel}} = &{{$bridge.DownSingular}}R{{$foreignRel}}R{boundParent: foreign}
	_, err := bridge.Create(ctx, exec)
	return err
}
{{end -}}
{{end -}}

// insertOptRels creates and inserts any optional the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. 
// any required relationship should have already exist on the model
func (o *{{$tAlias.UpSingular}}Template) insertOptRels(ctx context.Context, exec bob.Executor, m *models.{{$tAlias.UpSingular}}) (error) {
	var err error

	{{range $index, $rel := $.Relationships.Get $table.Key -}}{{if not ($.Tables.RelIsView $rel) -}}
		{{- if or ($table.RelIsRequired $rel) ($.Tables.RelIsOptionalFKParent $rel)}}{{continue}}{{end -}}
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
          if r.o.alreadyPersisted {
            m.R.{{$relAlias}} = append(m.R.{{$relAlias}}, r.o.Build())
          } else {
          {{- $bridges := $.Tables.NeededBridgeRels $rel -}}
          {{- if and $bridges ($.Relationships.FactoryBridgeParents $rel) }}
            for range r.number {
              related, err := r.o.Create(ctx, exec)
              if err != nil { return err }
              if err := r.createBridge(ctx, exec, m, related); err != nil { return err }
              m.R.{{$relAlias}} = append(m.R.{{$relAlias}}, related)
              {{- if $invAlias }}
                {{- if $invRel.IsToMany }}
                related.R.{{$invAlias}} = append(related.R.{{$invAlias}}, m)
                {{- else }}
                related.R.{{$invAlias}} = m
                related.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
                {{- end }}
              {{- end }}
            }
          {{- else }}
            {{- range $bridges -}}
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
          {{- end }}
					}
				}
		{{- else -}}
      if o.r.{{$relAlias}}.o.alreadyPersisted {
        m.R.{{$relAlias}} = o.r.{{$relAlias}}.o.Build()
        m.R.{{$.RelationLoadedName}}.{{$relAlias}} = true
      } else {
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

	// Retrieve ancestor models from context to avoid duplicate parent creation.
	// Parents are keyed by "parent_table:child_table:child_rel_name".
	// This works regardless of NoBackReferencing since it only uses child-side metadata.
	mInCreation, _ := modelsInCreationCtx.Value(ctx)

	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- if not ($table.RelIsRequired $rel)}}{{continue}}{{end -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}

		var rel{{$index}} *models.{{$ftable.UpSingular}}

		if o.r.{{$relAlias}} == nil {
      if parentModel, found := mInCreation["{{$rel.Foreign}}:{{$table.Key}}:{{$rel.Name}}"]; found {
        if pModel, ok := parentModel.(*models.{{$ftable.UpSingular}}); ok {
          rel{{$index}} = pModel
        }
      }
		{{if and (eq (len $rel.Sides) 1) (not $rel.IsToMany) -}}
		} else {
			rel{{$index}} = o.r.{{$relAlias}}.boundParent
		{{end -}}
		}

		if rel{{$index}} == nil {
			if o.r.{{$relAlias}} == nil {
				{{$tAlias.UpSingular}}Mods.WithNew{{$relAlias}}().Apply(ctx, o)
			}

			if o.r.{{$relAlias}}.o.alreadyPersisted {
				rel{{$index}} = o.r.{{$relAlias}}.o.Build()
			} else {
				rel{{$index}}, err = o.r.{{$relAlias}}.o.Create(ctx, exec)
				if err != nil {
					return nil, err
				}
			}
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

	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- if not ($.Tables.RelIsOptionalFKParent $rel)}}{{continue}}{{end -}}
		{{- $side := index $rel.Sides 0 -}}
		{{- $ftable := $.Aliases.Table $rel.Foreign -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		var rel{{$index}} *models.{{$ftable.UpSingular}}
		if done, _ := {{$tAlias.DownSingular}}Rel{{$relAlias}}Ctx.Value(ctx); !done && o.r.{{$relAlias}} != nil {
			parent := o.r.{{$relAlias}}.o
			if parent.alreadyPersisted {
				rel{{$index}} = parent.Build()
			} else {
				relCtx := {{$tAlias.DownSingular}}Rel{{$relAlias}}Ctx.WithValue(ctx, true)
				rel{{$index}}, err = parent.Create(relCtx, exec)
				if err != nil {
					return nil, err
				}
			}
			{{range $i, $col := $side.FromColumns -}}
			opt.{{$tAlias.Column $col}} = {{$.Tables.ColumnAssigner $.CurrentPackage $.Importer $.Types $.Aliases $table.Key $rel.Foreign $col (index $side.ToColumns $i) (printf "rel%d" $index) true}}
			{{end -}}
		}
	{{end}}

	m, err := models.{{$tAlias.UpPlural}}.Insert(opt).One(ctx, exec)
	if err != nil {
	  return nil, err
	}

  // Store this model in context for child creates.
  // Key format: "parent_table:child_table:child_rel_name" where child_rel_name is the FK name.
  // We store an entry for every relationship pointing TO this table (where this table is the parent).
  // This works regardless of NoBackReferencing since keys use child-side metadata.
  newMInCreation := make(map[string]any, len(mInCreation)+1)
  for k, v := range mInCreation {
    newMInCreation[k] = v
  }
  {{range $childTable := $.Tables -}}
    {{- range $childRel := $.Relationships.Get $childTable.Key -}}
      {{- if $childRel.IsToMany -}}{{continue}}{{end -}}
      {{- if ne $childRel.Foreign $table.Key -}}{{continue}}{{end -}}
      newMInCreation["{{$table.Key}}:{{$childTable.Key}}:{{$childRel.Name}}"] = m
    {{end -}}
  {{end}}
  ctx = modelsInCreationCtx.WithValue(ctx, newMInCreation)

	{{range $index, $rel := $.Relationships.Get $table.Key -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		{{if $table.RelIsRequired $rel -}}
		m.R.{{$relAlias}} = rel{{$index}}
		m.R.{{$.RelationLoadedName}}.{{$relAlias}} = true
		{{else if $.Tables.RelIsOptionalFKParent $rel -}}
		if rel{{$index}} != nil {
			m.R.{{$relAlias}} = rel{{$index}}
			m.R.{{$.RelationLoadedName}}.{{$relAlias}} = true
			{{ $invRel := $.Relationships.GetInverse $rel -}}
			{{if and (not $.NoBackReferencing) $invRel.Name -}}
				{{- $invAlias := ($.Aliases.Table $rel.Foreign).Relationship $invRel.Name -}}
				{{if $invRel.IsToMany -}}
				rel{{$index}}.R.{{$invAlias}} = append(rel{{$index}}.R.{{$invAlias}}, m)
				{{else -}}
				rel{{$index}}.R.{{$invAlias}} = m
				rel{{$index}}.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
				{{end -}}
			{{end -}}
		}
		{{end -}}
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
