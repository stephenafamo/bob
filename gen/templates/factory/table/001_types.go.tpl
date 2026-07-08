{{$.Importer.Import "context"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}
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

type {{$tAlias.DownSingular}}Factory interface {
    {{$.FactoryDependencyMethods $table.Key}}
}

// {{$tAlias.UpSingular}}Template is an object representing the database table.
// all columns are optional and should be set by mods
type {{$tAlias.UpSingular}}Template struct {
    {{- range $column := $table.Columns -}}
        {{- $colAlias := $tAlias.Column $column.Name -}}
        {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
        {{$colAlias}} func() {{$colTyp}}
    {{end -}}

    {{block "factory_template/fields/additional" $}}{{end}}

    {{if $.Relationships.Get $table.Key -}}
        r {{$tAlias.DownSingular}}R
    {{- end}}
    f {{$tAlias.DownSingular}}Factory

    alreadyPersisted bool
    requireAll       bool
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
    o *{{$.FactoryTemplateType .Foreign}}
    {{$.FactoryRelDependenciesTyp .}}
}
{{end}}

func New{{$tAlias.UpSingular}}WithContext(ctx context.Context, f {{$tAlias.DownSingular}}Factory, baseMods {{$tAlias.UpSingular}}ModSlice, mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f}

	baseMods.Apply(ctx, o)
	{{$tAlias.UpSingular}}ModSlice(mods).Apply(ctx, o)

	return o
}

func FromExisting{{$tAlias.UpSingular}}(ctx context.Context, f {{$tAlias.DownSingular}}Factory, m *models.{{$tAlias.UpSingular}}) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f, alreadyPersisted: true}

  {{range $column := $table.Columns -}}
  {{$colAlias := $tAlias.Column $column.Name -}}
  {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
        o.{{$colAlias}} = func() {{$colTyp}} { return m.{{$colAlias}} }
  {{end}}

  {{range $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship .Name -}}
    {{if .IsToMany -}}
      if len(m.R.{{$relAlias}}) > 0 {
      {{$tAlias.UpSingular}}Mods.AddExisting{{$relAlias}}(m.R.{{$relAlias}}...).Apply(ctx, o)
      }
    {{- else -}}
      if m.R.{{$relAlias}} != nil {
      {{$tAlias.UpSingular}}Mods.WithExisting{{$relAlias}}(m.R.{{$relAlias}}).Apply(ctx, o)
      }
    {{- end}}
  {{end}}

  return o
}

func (o *{{$tAlias.UpSingular}}Template) AlreadyPersisted() bool {
	return o.alreadyPersisted
}

// Apply mods to the {{$tAlias.UpSingular}}Template
func (o *{{$tAlias.UpSingular}}Template) Apply(ctx context.Context, mods ...{{$tAlias.UpSingular}}Mod) {
  for _, mod := range mods {
        mod.Apply(ctx, o)
    }
}
