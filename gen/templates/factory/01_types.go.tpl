{{$.Importer.Import "models" $.ModelsPackage}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table .Table.Name -}}

type {{$tAlias.UpSingular}}Mod interface {
	Apply(*{{$tAlias.UpSingular}}Template)
}

type {{$tAlias.UpSingular}}ModFunc func(*{{$tAlias.UpSingular}}Template)

func (f {{$tAlias.UpSingular}}ModFunc) Apply(n *{{$tAlias.UpSingular}}Template) {
	f(n)
}

type {{$tAlias.UpSingular}}Mods []{{$tAlias.UpSingular}}Mod

func (mods {{$tAlias.UpSingular}}Mods) Apply(n *{{$tAlias.UpSingular}}Template) {
	for _, f := range mods {
		 f.Apply(n)
	}
}

// {{$tAlias.UpSingular}}TemplateSlice is an alias for a slice of pointers to {{$tAlias.UpSingular}}.
// This should almost always be used instead of []{{$tAlias.UpSingular}}Template.
type {{$tAlias.UpSingular}}TemplateSlice []*{{$tAlias.UpSingular}}Template

// {{$tAlias.UpSingular}}Template is an object representing the database table.
// all columns are optional and should be set by mods
type {{$tAlias.UpSingular}}Template struct {
	{{- range $column := .Table.Columns -}}
		{{- $.Importer.ImportList $column.Imports -}}
		{{- $colAlias := $tAlias.Column $column.Name -}}
		{{- $colTyp := "" -}}
		{{- if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			{{- $colTyp = printf "omitnull.Val[%s]" $column.Type -}}
		{{- else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			{{- $colTyp = printf "omit.Val[%s]" $column.Type -}}
		{{- end -}}
		{{$colAlias}} {{$colTyp}}
	{{end -}}

	{{if .Table.Relationships}}
		r {{$tAlias.DownSingular}}R
	{{end -}}
}

{{if .Table.Relationships -}}
{{$.Importer.Import "github.com/aarondl/opt/omit" -}}
type {{$tAlias.DownSingular}}R struct {
	{{range .Table.Relationships -}}
		{{- $ftable := $.Aliases.Table .Foreign -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		{{- $relTyp := printf "*%sTemplate" $ftable.UpSingular -}}
		{{- if .IsToMany -}}
			{{$relTyp = printf "%sTemplateSlice" $ftable.UpSingular}}
		{{- end -}}
		{{- if  .NeededColumns -}}
			{{$relTyp = printf "*%s%sR" $tAlias.DownSingular $relAlias}}
		{{- end -}}
		{{- if  and .IsToMany .NeededColumns -}}
			{{$relTyp = printf "[]*%s%sR" $tAlias.DownSingular $relAlias}}
		{{- end -}}

		{{$relAlias}} {{$relTyp}}
	{{end -}}
}
{{- end}}

{{range .Table.Relationships}}{{if .NeededColumns -}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
{{- $relTyp := printf "*%sTemplate" $ftable.UpSingular -}}
{{- if .IsToMany -}}
	{{$relTyp = printf "%sTemplateSlice" $ftable.UpSingular}}
{{- end -}}
type {{$tAlias.DownSingular}}{{$relAlias}}R {{relDependenciesTyp $.Aliases . $relTyp}}
{{end}}{{end}}

// Apply mods to the {{$tAlias.UpSingular}}Template
func (o *{{$tAlias.UpSingular}}Template) Apply(mods ...{{$tAlias.UpSingular}}Mod) {
  for _, mod := range mods {
		mod.Apply(o)
	}
}

// toModel returns an *models.{{$tAlias.UpSingular}}
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) toModel() (*models.{{$tAlias.UpSingular}}) {
	m := &models.{{$tAlias.UpSingular}}{}

	{{range $column := .Table.Columns -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !o.{{$colAlias}}.IsUnset() {
			{{if $column.Nullable -}}
			m.{{$colAlias}} = o.{{$colAlias}}.MustGetNull()
			{{else -}}
			m.{{$colAlias}} = o.{{$colAlias}}.MustGet()
			{{end -}}
		}
	{{end}}

	return m
}

// toModel returns an models.{{$tAlias.UpSingular}}Slice
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}TemplateSlice) toModel() (models.{{$tAlias.UpSingular}}Slice) {
	m := make(models.{{$tAlias.UpSingular}}Slice, len(o))

	for i, o := range o {
	  m[i] = o.toModel()
	}

	return m
}

// setModelRelationships creates and sets the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. Nothing is inserted into the db
func (o {{$tAlias.UpSingular}}Template) setModelRelationships(m *models.{{$tAlias.UpSingular}}) {
	{{- range $index, $rel := .Table.Relationships -}}
		{{- $relAlias := $tAlias.Relationship .Name -}}
		{{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
		{{- $ftable := $.Aliases.Table $rel.Foreign -}}
		{{- $invAlias := "" -}}
    {{- if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias = $ftable.Relationship $invRel.Name}}
		{{- end -}}

		{{if not .IsToMany -}}
			{{- if not .NeededColumns}}
				rel{{$index}} := o.r.{{$relAlias}}.toModel()
			{{- else}}
				rel{{$index}} := o.r.{{$relAlias}}.o.toModel()
			{{- end}}
			{{- if and (not $.NoBackReferencing) $invRel.Name}}
				{{- if not $invRel.IsToMany}}
					rel{{$index}}.R.{{$invAlias}} = m
				{{- else}}
					rel{{$index}}.R.{{$invAlias}} = models.{{$tAlias.UpSingular}}Slice{m}
				{{- end}}
			{{- end}}
		{{else}}
			rel{{$index}} := models.{{$ftable.UpSingular}}Slice{}
			for _, r := range o.r.{{$relAlias}} {
				{{- if .NeededColumns}} for _, r := range r.o { {{- end}}
				relM := r.toModel()
				{{- if and (not $.NoBackReferencing) $invRel.Name}}
					{{- if not $invRel.IsToMany}}
						relM.R.{{$invAlias}} = m
					{{- else}}
						relM.R.{{$invAlias}} = models.{{$tAlias.UpSingular}}Slice{m}
					{{- end}}
					rel{{$index}} = append(rel{{$index}}, relM)
				{{- end}}
				{{- if .NeededColumns}} } {{- end}}
			}
		{{end -}}
		m.R.{{$relAlias}} = rel{{$index}}
	{{end -}}
}

