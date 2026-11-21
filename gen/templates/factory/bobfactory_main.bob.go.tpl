{{$.Importer.Import "context"}}
{{$.Importer.Import "models" (index $.OutputPackages "models") }}

type Factory struct {
    {{range $table := .Tables}}
    {{ $tAlias := $.Aliases.Table $table.Key -}}
		base{{$tAlias.UpSingular}}Mods {{$tAlias.UpSingular}}ModSlice
    {{- end}}
}

func New() *Factory {
  return &Factory{}
}


{{range $table := .Tables}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func (f *Factory) New{{$tAlias.UpSingular}}(mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	return f.New{{$tAlias.UpSingular}}WithContext(context.Background(), mods...)
}

func (f *Factory) New{{$tAlias.UpSingular}}WithContext(ctx context.Context, mods ...{{$tAlias.UpSingular}}Mod) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f}

  if f != nil {
    f.base{{$tAlias.UpSingular}}Mods.Apply(ctx, o)
  }

  {{$tAlias.UpSingular}}ModSlice(mods).Apply(ctx, o)

	return o
}

func (f *Factory) FromExisting{{$tAlias.UpSingular}}(ctx context.Context, m *models.{{$tAlias.UpSingular}}) *{{$tAlias.UpSingular}}Template {
  return f.fromExisting{{$tAlias.UpSingular}}(ctx, m)
}

func (f *Factory) fromExisting{{$tAlias.UpSingular}}(ctx context.Context, m *models.{{$tAlias.UpSingular}}) *{{$tAlias.UpSingular}}Template {
	o := &{{$tAlias.UpSingular}}Template{f: f, alreadyPersisted: true}

  {{range $column := $table.Columns -}}
  {{$colAlias := $tAlias.Column $column.Name -}}
  {{- $colTyp := $.Types.GetNullable $.CurrentPackage $.Importer $column.Type $column.Nullable -}}
        o.{{$colAlias}} = func() {{$colTyp}} { return m.{{$colAlias}} }
  {{end}}

  {{if $.Relationships.Get $table.Key -}}
  path, ok := factoryPathCtx.Value(ctx)
  if !ok {
    path = make(map[any]struct{})
    ctx = factoryPathCtx.WithValue(ctx, path)
  }

  if _, onPath := path[m]; onPath {
    return o
  }
  path[m] = struct{}{}
  defer delete(path, m)
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
  {{- end}}

  return o
}

{{end}}

{{range $table := .Tables}}
{{ $tAlias := $.Aliases.Table $table.Key -}}
func (f *Factory) ClearBase{{$tAlias.UpSingular}}Mods() {
    f.base{{$tAlias.UpSingular}}Mods = nil
}

func (f *Factory) AddBase{{$tAlias.UpSingular}}Mod(mods ...{{$tAlias.UpSingular}}Mod) {
f.base{{$tAlias.UpSingular}}Mods = append(f.base{{$tAlias.UpSingular}}Mods, mods...)
}

{{end}}
