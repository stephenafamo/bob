{{if $.Relationships.Get .Table.Key -}}
{{$.Importer.Import "fmt" -}}
{{$.Importer.Import "context" -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm" -}}

{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}


func (o *{{$tAlias.UpSingular}}) Preload(name string, retrieved any) error {
	if o == nil {
		return nil
	}

	switch name {
	{{range $.Relationships.Get $table.Key -}}
	{{- $fAlias := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	case "{{$relAlias}}":
		related, ok := retrieved.({{if .IsToMany}}{{$fAlias.UpSingular}}Slice{{else}}*{{$fAlias.UpSingular}}{{end}})
		if !ok { return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name) }
		{{$tAlias.UpSingular}}RelAssignments.{{$relAlias}}().Replace(o, related)
		return nil

	{{end -}}
	default:
		return fmt.Errorf("{{$tAlias.DownSingular}} has no relationship %q", name)
	}
}

type {{$tAlias.DownSingular}}Preloader struct {
  {{range $rel := $.Relationships.Get $table.Key -}}
  {{- if $rel.IsToMany -}}{{continue}}{{- end -}}
  {{- $relAlias := $tAlias.Relationship $rel.Name -}}
  {{$relAlias}} func(...{{$.Dialect}}.PreloadOption) {{$.Dialect}}.Preloader
  {{end -}}
}

func build{{$tAlias.UpSingular}}Preloader() {{$tAlias.DownSingular}}Preloader {
  return {{$tAlias.DownSingular}}Preloader{
    {{range $rel := $.Relationships.Get $table.Key -}}
    {{- if $rel.IsToMany -}}{{continue}}{{- end -}}
    {{- $relAlias := $tAlias.Relationship $rel.Name -}}
    {{- $fAlias := $.Aliases.Table $rel.Foreign -}}
    {{$relAlias}}: func(opts ...{{$.Dialect}}.PreloadOption) {{$.Dialect}}.Preloader {
      return {{$.Dialect}}.Preload[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice]({{$tAlias.DownSingular}}Rel{{$relAlias}}, {{$fAlias.UpPlural}}.Columns.Names(), {{$fAlias.DownSingular}}ScanMapperNullable, opts...)
    },
    {{end -}}
  }
}


type {{$tAlias.DownSingular}}ThenLoader[Q orm.Loadable] struct {
  {{range $rel := $.Relationships.Get $table.Key -}}
  {{- $relAlias := $tAlias.Relationship $rel.Name -}}
  {{$relAlias}} func(...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q]
  {{end -}}
}

func build{{$tAlias.UpSingular}}ThenLoader[Q orm.Loadable]() {{$tAlias.DownSingular}}ThenLoader[Q] {
  {{range $rel := $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship $rel.Name -}}
    type {{$relAlias}}LoadInterface interface{
      Load{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[*dialect.SelectQuery]) error
    }
  {{end}}

  return {{$tAlias.DownSingular}}ThenLoader[Q]{
    {{range $rel := $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship $rel.Name -}}
    {{$fAlias := $.Aliases.Table $rel.Foreign -}}
    {{$relAlias}}: thenLoadBuilder[Q](
      "{{$relAlias}}",
      func(ctx context.Context, exec bob.Executor, retrieved {{$relAlias}}LoadInterface, mods ...bob.Mod[*dialect.SelectQuery]) error {
        return retrieved.Load{{$relAlias}}(ctx, exec, mods...)
      },
    ),
    {{end}}
  }
}



{{range $rel := $.Relationships.Get $table.Key -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}

// Load{{$relAlias}} loads the {{$tAlias.DownSingular}}'s {{$relAlias}} into the .R struct
func (o *{{$tAlias.UpSingular}}) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
  if o == nil {
	  return nil
	}

	assignment := {{$tAlias.UpSingular}}RelAssignments.{{$relAlias}}()
	assignment.Reset(o, false)

	{{if $rel.IsToMany -}}
	related, err := o.{{relQueryMethodName $tAlias $relAlias}}(mods...).All(ctx, exec)
	{{else -}}
	related, err := o.{{relQueryMethodName $tAlias $relAlias}}(mods...).One(ctx, exec)
	{{end -}}
	if err != nil {
		return err
	}

	assignment.Replace(o, related)
	return nil
}

{{$side := index $rel.Sides 0 -}}
// Load{{$relAlias}} loads the relationship for the whole slice with a single query.
func (os {{$tAlias.UpSingular}}Slice) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if len(os) == 0 { return nil }
	parents := os{{if $side.FromWhere}}.matching{{$relAlias}}Sources(){{end}}
	{{if $side.FromColumns -}}
	{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
	parents, keys, owners := parents.parentsFor{{$relAlias}}()
	q := {{$tAlias.DownSingular}}QueryFor{{$relAlias}}(keys, mods...)
	q.Expression.AppendPreloadSelect(owners.As("related_.parents"))
	indexes := make(genruntime.ParentIndexes[{{$fAlias.UpSingular}}])
	q.Scanner = indexes.Mapper(q.Scanner, "related_.parents")
	children, err := q.All(ctx, exec)
	{{else -}}
	children, err := os.{{relQueryMethodName $tAlias $relAlias}}(mods...).All(ctx, exec)
	{{end -}}
	if err != nil { return err }

	assignment := {{$tAlias.UpSingular}}RelAssignments.{{$relAlias}}()
	for _, o := range os {
		if o != nil {
			assignment.Reset(o, true)
		}
	}
	{{if $side.FromColumns -}}
	for _, child := range children {
		for _, j := range indexes[child] { assignment.Link(parents[j], child) }
	}
	{{else -}}
		// With no join keys, each eligible parent gets the same result.
		for _, o := range parents {
			if o == nil { continue }
			for _, rel := range children {
				assignment.Link(o, rel)
				{{if not $rel.IsToMany}}break{{end}}
			}
		}
	{{end -}}
	return nil
}
{{end -}}
{{end -}}
