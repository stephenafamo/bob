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

type {{$tAlias.UpSingular}}ModSlice []{{$tAlias.UpSingular}}Mod

func (mods {{$tAlias.UpSingular}}ModSlice) Apply(n *{{$tAlias.UpSingular}}Template) {
    for _, f := range mods {
         f.Apply(n)
    }
}

// {{$tAlias.UpSingular}}Template is an object representing the database table.
// all columns are optional and should be set by mods
type {{$tAlias.UpSingular}}Template struct {
    {{- range $column := .Table.Columns -}}
        {{- $.Importer.ImportList $column.Imports -}}
        {{- $colAlias := $tAlias.Column $column.Name -}}
        {{- $colTyp := $column.Type -}}
        {{- if $column.Nullable -}}
            {{- $.Importer.Import "github.com/aarondl/opt/null" -}}
            {{- $colTyp = printf "null.Val[%s]" $column.Type -}}
        {{- end -}}
        {{$colAlias}} func() {{$colTyp}}
    {{end -}}

    {{if .Table.Relationships -}}
        r {{$tAlias.DownSingular}}R
    {{- end}}
    f *factory
}

{{if .Table.Relationships -}}
type {{$tAlias.DownSingular}}R struct {
    {{range .Table.Relationships -}}
        {{- $ftable := $.Aliases.Table .Foreign -}}
        {{- $relAlias := $tAlias.Relationship .Name -}}
        {{- $relTyp := printf "*%s%sR" $tAlias.DownSingular $relAlias -}}
        {{- if .IsToMany -}}
            {{$relTyp = printf "[]*%s%sR" $tAlias.DownSingular $relAlias}}
        {{- end -}}

        {{$relAlias}} {{$relTyp}}
    {{end -}}
}
{{- end}}

{{range .Table.Relationships}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
type {{$tAlias.DownSingular}}{{$relAlias}}R struct{
    {{- if .IsToMany}}
        number int
    {{- end}}
    o *{{$ftable.UpSingular}}Template
    {{relDependenciesTyp $.Aliases .}}
}
{{end}}

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
        if o.{{$colAlias}} != nil {
            m.{{$colAlias}} = o.{{$colAlias}}()
        }
    {{end}}

    return m
}

// toModels returns an models.{{$tAlias.UpSingular}}Slice
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) toModels(number int) (models.{{$tAlias.UpSingular}}Slice) {
    m := make(models.{{$tAlias.UpSingular}}Slice, number)

    for i := range m {
      m[i] = o.toModel()
    }

    return m
}

// setModelRelationships creates and sets the relationships on *models.{{$tAlias.UpSingular}}
// according to the relationships in the template. Nothing is inserted into the db
func (t {{$tAlias.UpSingular}}Template) setModelRelationships(o *models.{{$tAlias.UpSingular}}) {
    {{- range $index, $rel := .Table.Relationships -}}
        {{- $relAlias := $tAlias.Relationship .Name -}}
        {{- $invRel := $table.GetRelationshipInverse $.Tables . -}}
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
                  for _, rel := range related {
                    {{- setFactoryDeps $.Importer $.Tables $.Aliases . false}}
                    {{- if and (not $.NoBackReferencing) $invRel.Name}}
                        {{- if not $invRel.IsToMany}}
                            rel.R.{{$invAlias}} = o
                        {{- else}}
                            rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
                        {{- end}}
                    {{- end}}
                  }
                  rel = append(rel, related...)
                }
            {{- end}}
            o.R.{{$relAlias}} = rel
        }

    {{end -}}
}

