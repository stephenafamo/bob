{{$.Importer.Import "models" $.ModelsPackage}}
{{ $table := .Table}}
{{ $tAlias := .Aliases.Table $table.Key -}}

// setModelRels creates and sets the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. Nothing is inserted into the db
func (t {{$tAlias.UpSingular}}Template) setModelRels(o *models.{{$tAlias.UpSingular}}) {
    {{- range $index, $rel := $.Relationships.Get $table.Key -}}
        {{- $relAlias := $tAlias.Relationship .Name -}}
        {{- $invRel := $.Relationships.GetInverse $.Tables . -}}
        {{- $ftable := $.Aliases.Table $rel.Foreign -}}
        {{- $invAlias := "" -}}
    {{- if and (not $.NoBackReferencing) $invRel.Name -}}
            {{- $invAlias = $ftable.Relationship $invRel.Name}}
        {{- end -}}

        if t.r.{{$relAlias}} != nil {
            {{- if not .IsToMany}}
                rel := t.r.{{$relAlias}}.o.toModel()
                {{- if and (not $.NoBackReferencing) $invRel.Name}}
                    {{- if not $invRel.IsToMany}}
                        rel.R.{{$invAlias}} = o
                    {{- else}}
                        rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
                    {{- end}}
                {{- end}}
                {{setFactoryDeps $.Importer $.Tables $.Aliases . false}}
            {{- else -}}
                rel := models.{{$ftable.UpSingular}}Slice{}
                for _, r := range t.r.{{$relAlias}} {
                  related := r.o.toModels(r.number)
                  {{- $setter := setFactoryDeps $.Importer $.Tables $.Aliases . false}}
                  {{- if or $setter (and (not $.NoBackReferencing) $invRel.Name) }}
                  for _, rel := range related {
                    {{$setter}}
                    {{- if and (not $.NoBackReferencing) $invRel.Name}}
                        {{- if not $invRel.IsToMany}}
                            rel.R.{{$invAlias}} = o
                        {{- else}}
                            rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
                        {{- end}}
                    {{- end}}
                  }
                  {{- end}}
                  rel = append(rel, related...)
                }
            {{- end}}
            o.R.{{$relAlias}} = rel
        }

    {{end -}}
}

{{if $table.Constraints.Primary -}}
// BuildSetter returns an *models.{{$tAlias.UpSingular}}Setter
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) BuildSetter() *models.{{$tAlias.UpSingular}}Setter {
	m := &models.{{$tAlias.UpSingular}}Setter{}

	{{range $column := $table.Columns -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if o.{{$colAlias}} != nil {
			{{if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			m.{{$colAlias}} = omitnull.FromNull(o.{{$colAlias}}())
			{{else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			m.{{$colAlias}} = omit.From(o.{{$colAlias}}())
			{{end -}}
		}
	{{end}}

	return m
}

// BuildManySetter returns an []*models.{{$tAlias.UpSingular}}Setter
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) BuildManySetter(number int) []*models.{{$tAlias.UpSingular}}Setter {
	m := make([]*models.{{$tAlias.UpSingular}}Setter, number)

	for i := range m {
	  m[i] = o.BuildSetter()
	}

	return m
}
{{- end}}

// Build returns an *models.{{$tAlias.UpSingular}}
// Related objects are also created and placed in the .R field
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}Template.Create
func (o {{$tAlias.UpSingular}}Template) Build() *models.{{$tAlias.UpSingular}} {
	m := o.toModel()
	o.setModelRels(m)

	return m
}

// BuildMany returns an models.{{$tAlias.UpSingular}}Slice
// Related objects are also created and placed in the .R field
// NOTE: Objects are not inserted into the database. Use {{$tAlias.UpSingular}}Template.CreateMany
func (o {{$tAlias.UpSingular}}Template) BuildMany(number int) models.{{$tAlias.UpSingular}}Slice {
	m := make(models.{{$tAlias.UpSingular}}Slice, number)

	for i := range m {
	  m[i] = o.Build()
	}

	return m
}
