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
	{{- $invRel := $.Relationships.GetInverse . -}}
	case "{{$relAlias}}":
		{{if .IsToMany -}}
			rels, ok := retrieved.({{$fAlias.UpSingular}}Slice)
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rels
			o.R.{{$.RelationLoadedName}}.{{$relAlias}} = true

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
			for _, rel := range rels {
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
					{{- else -}}
						rel.R.{{$invAlias}} =  o
						rel.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
					{{- end}}
				}
			}
			{{- end}}
			return nil
		{{else -}}
			rel, ok := retrieved.(*{{$fAlias.UpSingular}})
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rel
			o.R.{{$.RelationLoadedName}}.{{$relAlias}} = true

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
					{{- else -}}
						rel.R.{{$invAlias}} =  o
						rel.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
					{{- end}}
				}
			{{- end}}
			return nil
		{{end -}}

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
{{- $invRel := $.Relationships.GetInverse . -}}

// Load{{$relAlias}} loads the {{$tAlias.DownSingular}}'s {{$relAlias}} into the .R struct
func (o *{{$tAlias.UpSingular}}) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
  if o == nil {
	  return nil
	}

	// Reset the relationship
	o.R.{{$relAlias}} = nil
	o.R.{{$.RelationLoadedName}}.{{$relAlias}} = false

	{{if $rel.IsToMany -}}
	related, err := o.{{relQueryMethodName $tAlias $relAlias}}(mods...).All(ctx, exec)
	{{else -}}
	related, err := o.{{relQueryMethodName $tAlias $relAlias}}(mods...).One(ctx, exec)
	{{end -}}
	if err != nil {
		return err
	}

	{{if and (not $.NoBackReferencing) $invRel.Name -}}
	{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
	{{if $rel.IsToMany -}}
		for _, rel := range related {
			{{if $invRel.IsToMany -}}
				rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
			{{- else -}}
				rel.R.{{$invAlias}} =  o
				rel.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
			{{- end}}
		}
	{{else -}}
		{{if $invRel.IsToMany -}}
			related.R.{{$invAlias}} = append(related.R.{{$invAlias}}, o)
		{{else -}}
			related.R.{{$invAlias}} =  o
			related.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
		{{- end}}
	{{- end}}
	{{- end}}

	o.R.{{$relAlias}} = related
	o.R.{{$.RelationLoadedName}}.{{$relAlias}} = true
	return nil
}

{{$side := index $rel.Sides 0 -}}
{{$from := $.Aliases.Table $side.From -}}
{{$to := $.Aliases.Table $side.To -}}
{{$through := gt (len $rel.Sides) 1 -}}
// Load{{$relAlias}} loads the relationship for the whole slice with a single query.
func (os {{$tAlias.UpSingular}}Slice) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if len(os) == 0 { return nil }
	{{if $through -}}
	{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect)}}
	{{$.Importer.Import "github.com/stephenafamo/scan"}}
	// The child row does not contain the parent key. Select the key from the
	// first join table and scan it alongside the child, in the same row order.
	q := os.{{relQueryMethodName $tAlias $relAlias}}(append(mods,
		bob.ModFunc[*dialect.SelectQuery](func(q *dialect.SelectQuery) {
			if len(q.SelectList.Columns) == 0 { sm.Columns({{$fAlias.UpPlural}}.Columns).Apply(q) }
		}),
		{{range $i, $local := $side.FromColumns -}}
		sm.Columns({{$to.UpPlural}}.Columns.{{$to.Column (index $side.ToColumns $i)}}.As("related_{{$side.From}}.{{$from.Column $local}}")),
		{{end -}}
	)...)
	{{range $i, $local := $side.FromColumns -}}
		{{$col := $.Tables.GetColumn $side.To (index $side.ToColumns $i) -}}
		var keys{{$i}} []{{$.Types.GetNullable $.CurrentPackage $.Importer $col.Type $col.Nullable}}
	{{end -}}
	mapper := scan.Mod(q.Scanner, {{range $i, $local := $side.FromColumns -}}
		orm.CollectColumn("related_{{$side.From}}.{{$from.Column $local}}", &keys{{$i}}),
	{{end -}})
	children, err := bob.Allx[bob.SliceTransformer[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice]](ctx, exec, q, mapper)
	{{else -}}
	children, err := os.{{relQueryMethodName $tAlias $relAlias}}(mods...).All(ctx, exec)
	{{end -}}
	if err != nil { return err }

	for _, o := range os {
		if o != nil {
			o.R.{{$relAlias}} = nil
			o.R.{{$.RelationLoadedName}}.{{$relAlias}} = true
		}
	}
	{{if $side.FromWhere}}os = os.matching{{$relAlias}}Sources(){{end}}
	link := func(o *{{$tAlias.UpSingular}}, rel *{{$fAlias.UpSingular}}) {
		{{if not $rel.IsToMany -}}
		if o.R.{{$relAlias}} != nil { return } // keep the first match
		{{end -}}
		{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{$invAlias := $fAlias.Relationship $invRel.Name -}}
			{{if $invRel.IsToMany -}}
			rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
			{{else -}}
			rel.R.{{$invAlias}} = o
			rel.R.{{$.RelationLoadedName}}.{{$invAlias}} = true
			{{end -}}
		{{end -}}
		{{if $rel.IsToMany -}}
		o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rel)
		{{else -}}
		o.R.{{$relAlias}} = rel
		{{end -}}
	}
	{{if and (eq (len $side.FromColumns) 1) ($.Types.CanCompareWithEquals $.CurrentPackage ($.Tables.GetColumn $side.From (index $side.FromColumns 0)).Type) -}}
		{{$local := $.Tables.GetColumn $side.From (index $side.FromColumns 0) -}}
		{{$foreign := $.Tables.GetColumn $side.To (index $side.ToColumns 0) -}}
		{{$key := $.Types.Get $.CurrentPackage $.Importer $local.Type -}}
		{{$p := printf "o.%s" ($from.Column $local.Name) -}}
		{{$c := printf "rel.%s" ($to.Column $foreign.Name) -}}
		{{if $through}}{{$c = "keys0[i]"}}{{end -}}
		orm.StitchByKey(os, children,
			func(o *{{$tAlias.UpSingular}}) ({{$key}}, bool) {
				{{if $local.Nullable -}}
				if !({{$.Types.GetNullTypeValid $.CurrentPackage $local.Type $p}}) { var zero {{$key}}; return zero, false }
				{{end -}}
				return {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $local.Type $p $local.Nullable}}, true
			},
			func(i int, rel *{{$fAlias.UpSingular}}) ({{$key}}, bool) {
				{{if $foreign.Nullable -}}
				if !({{$.Types.GetNullTypeValid $.CurrentPackage $foreign.Type $c}}) { var zero {{$key}}; return zero, false }
				{{end -}}
				return {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $foreign.Type $c $foreign.Nullable}}, true
			}, link)
	{{else -}}
		// Composite keys and custom equality cannot assume a Go map key.
		for _, o := range os {
			if o == nil { continue }
			for {{if $through}}i{{else}}_{{end}}, rel := range children {
				{{range $i, $name := $side.FromColumns -}}
				{{$local := $.Tables.GetColumn $side.From $name -}}
				{{$foreign := $.Tables.GetColumn $side.To (index $side.ToColumns $i) -}}
				{{$p := printf "o.%s" ($from.Column $name) -}}
				{{$c := printf "rel.%s" ($to.Column $foreign.Name) -}}
				{{if $through}}{{$c = printf "keys%d[i]" $i}}{{end -}}
				{{$compare := $.Types.GetCompareExpr $.CurrentPackage $.Importer $local.Type $local.Nullable $foreign.Nullable -}}
				if !({{replace "AAA" $p $compare | replace "BBB" $c}}) { continue }
				{{end -}}
				link(o, rel)
				{{if not $rel.IsToMany}}break{{end}}
			}
		}
	{{end -}}
	return nil
}
{{end -}}
{{end -}}
