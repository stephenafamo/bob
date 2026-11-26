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
  {{$relAlias}} func(...bob.Mod[dialect.SelectEditor]) orm.Loader[Q]
  {{end -}}
}

func build{{$tAlias.UpSingular}}ThenLoader[Q orm.Loadable]() {{$tAlias.DownSingular}}ThenLoader[Q] {
  {{range $rel := $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship $rel.Name -}}
    type {{$relAlias}}LoadInterface interface{
      Load{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[dialect.SelectEditor]) error
    }
  {{end}}

  return {{$tAlias.DownSingular}}ThenLoader[Q]{
    {{range $rel := $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship $rel.Name -}}
    {{$fAlias := $.Aliases.Table $rel.Foreign -}}
    {{$relAlias}}: thenLoadBuilder[Q](
      "{{$relAlias}}",
      func(ctx context.Context, exec bob.Executor, retrieved {{$relAlias}}LoadInterface, mods ...bob.Mod[dialect.SelectEditor]) error {
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
func (o *{{$tAlias.UpSingular}}) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[dialect.SelectEditor]) error {
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
{{$from := $.Aliases.Table $side.From -}}
{{$path := printf "%sPathFor%s" $tAlias.DownSingular $relAlias -}}
{{$comparable := true -}}
// Load{{$relAlias}} loads the relationship for the whole slice with a single query.
func (os {{$tAlias.UpSingular}}Slice) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[dialect.SelectEditor]) error {
	if len(os) == 0 { return nil }
	parents := os{{if $side.FromWhere}}.matching{{$relAlias}}Sources(){{end}}
	{{if $side.FromColumns -}}
	{{$.Importer.Import "github.com/stephenafamo/bob/orm/genruntime"}}
	{{$.Importer.Import "github.com/stephenafamo/scan"}}
	type parentKey struct {
		{{range $i, $name := $side.FromColumns -}}
		{{$col := $table.GetColumn $name -}}
		{{if not ($.Types.CanCompareWithEquals $.CurrentPackage $col.Type)}}{{$comparable = false}}{{end -}}
		K{{$i}} {{$.Types.Get $.CurrentPackage $.Importer $col.Type}}
		{{end -}}
	}
	q := parents.{{relQueryMethodName $tAlias $relAlias}}(mods...).WithMods(bob.ModFunc[dialect.SelectEditor](func(s dialect.SelectEditor) {
		s.AppendPreloadSelect(
			{{range $i, $name := $side.ToColumns -}}
			{{$.Dialect}}.Quote({{$path}}[0].Table.Alias, {{quote $name}}).As("related_.key{{$i}}"),
			{{end -}}
		)
	}))
	keys := make(map[*{{$fAlias.UpSingular}}]parentKey)
	mapper := scan.Mod(q.Scanner, genruntime.CollectRelationshipKey(keys, func(k *parentKey) []any {
		return []any{ {{range $i, $_ := $side.FromColumns}}&k.K{{$i}},{{end}} }
	}))
	children, err := q.WithScanner(mapper).All(ctx, exec)
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
	{{if $comparable -}}
	genruntime.StitchByKey(parents, children,
		func(o *{{$tAlias.UpSingular}}) (parentKey, bool) {
			var key parentKey
			{{range $i, $name := $side.FromColumns -}}
			{{$col := $table.GetColumn $name -}}
			{{$field := printf "o.%s" ($from.Column $name) -}}
			{{if $col.Nullable}}if !({{$.Types.GetNullTypeValid $.CurrentPackage $col.Type $field}}) { return key, false }{{end}}
			key.K{{$i}} = {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $col.Type $field $col.Nullable}}
			{{end -}}
			return key, true
		},
		func(_ int, child *{{$fAlias.UpSingular}}) (parentKey, bool) { key, ok := keys[child]; return key, ok },
		assignment.Link)
	{{else -}}
	for _, o := range parents {
		if o == nil { continue }
		for _, child := range children {
			key, ok := keys[child]
			if !ok { continue }
			{{range $i, $name := $side.FromColumns -}}
			{{$col := $table.GetColumn $name -}}
			{{$compare := $.Types.GetCompareExpr $.CurrentPackage $.Importer $col.Type $col.Nullable false -}}
			if !({{replace "AAA" (printf "o.%s" ($from.Column $name)) $compare | replace "BBB" (printf "key.K%d" $i)}}) { continue }
			{{end -}}
			assignment.Link(o, child)
			{{if not $rel.IsToMany}}break{{end}}
		}
	}
	{{end -}}
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
