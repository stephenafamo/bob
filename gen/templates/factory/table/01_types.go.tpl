{{$.Importer.Import "context"}}
{{$.Importer.Import "models" $.ModelsPackage}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}

type {{$tAlias.UpSingular}}Mod interface {
    Apply(context.Context, *{{$tAlias.UpSingular}}Template)
}

type {{$tAlias.UpSingular}}ModFunc func(context.Context, *{{$tAlias.UpSingular}}Template)

func (f {{$tAlias.UpSingular}}ModFunc) Apply(ctx context.Context, n *{{$tAlias.UpSingular}}Template) {
    f(ctx, n)
}

type {{$tAlias.UpSingular}}ModSlice []{{$tAlias.UpSingular}}Mod

func (mods {{$tAlias.UpSingular}}ModSlice) Apply(ctx context.Context, n *{{$tAlias.UpSingular}}Template) {
    for _, f := range mods {
         f.Apply(ctx, n)
    }
}

// {{$tAlias.UpSingular}}Template is an object representing the database table.
// all columns are optional and should be set by mods
type {{$tAlias.UpSingular}}Template struct {
    {{- range $column := $table.Columns -}}
        {{- $typDef :=  index $.Types $column.Type -}}
        {{- $colTyp := getType $column.Type $typDef -}}
        {{- $.Importer.ImportList $typDef.Imports -}}
        {{- $colAlias := $tAlias.Column $column.Name -}}
        {{- if $column.Nullable -}}
            {{- $.Importer.Import "database/sql" -}}
            {{- $colTyp = printf "sql.Null[%s]" $colTyp -}}
        {{- end -}}
        {{$colAlias}} func() {{$colTyp}}
    {{end -}}

    {{block "factory_template/fields/additional" $}}{{end}}

    {{if $.Relationships.Get $table.Key -}}
        r {{$tAlias.DownSingular}}R
    {{- end}}
    f *Factory
}

{{if $.Relationships.Get $table.Key -}}
type {{$tAlias.DownSingular}}R struct {
    {{range $.Relationships.Get $table.Key -}}
        {{- $ftable := $.Aliases.Table .Foreign -}}
        {{- $relAlias := $tAlias.Relationship .Name -}}
        {{- $relTyp := printf "*%sR%sR" $tAlias.DownSingular $relAlias -}}
        {{- if .IsToMany -}}
            {{$relTyp = printf "[]*%sR%sR" $tAlias.DownSingular $relAlias}}
        {{- end -}}

        {{$relAlias}} {{$relTyp}}
    {{end -}}
}
{{- end}}

{{range $.Relationships.Get $table.Key}}
{{- $ftable := $.Aliases.Table .Foreign -}}
{{- $relAlias := $tAlias.Relationship .Name -}}
type {{$tAlias.DownSingular}}R{{$relAlias}}R struct{
    {{- if .IsToMany}}
        number int
    {{- end}}
    o *{{$ftable.UpSingular}}Template
    {{$.Tables.RelDependenciesTyp $.Aliases .}}
}
{{end}}

// Apply mods to the {{$tAlias.UpSingular}}Template
func (o *{{$tAlias.UpSingular}}Template) Apply(ctx context.Context, mods ...{{$tAlias.UpSingular}}Mod) {
  for _, mod := range mods {
        mod.Apply(ctx, o)
    }
}

// toModel returns an *models.{{$tAlias.UpSingular}}
// this does nothing with the relationship templates
func (o {{$tAlias.UpSingular}}Template) toModel() (*models.{{$tAlias.UpSingular}}) {
    m := &models.{{$tAlias.UpSingular}}{}

    {{range $column := $table.Columns -}}
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

