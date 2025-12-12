{{- $table := .Table -}}
{{- $tAlias := .Aliases.Table $table.Key -}}
{{- $rels := $.Relationships.Get $table.Key -}}
{{- $hasToMany := false -}}
{{- range $rel := $rels -}}
	{{- if $rel.IsToMany -}}{{- $hasToMany = true -}}{{- end -}}
{{- end -}}

{{if $hasToMany -}}
{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}

// {{$tAlias.DownSingular}}C is where relationship counts are stored.
type {{$tAlias.DownSingular}}C struct {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name -}}
	{{$relAlias}} *int64 {{if $.Tags}}`{{generateTags $.Tags $relAlias | trim}}`{{end}}
	{{end -}}
}

type {{$tAlias.DownSingular}}CountThenLoader[Q orm.Loadable] struct {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{- $relAlias := $tAlias.Relationship $rel.Name -}}
	{{$relAlias}} func(...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q]
	{{end -}}
}

func build{{$tAlias.UpSingular}}CountThenLoader[Q orm.Loadable]() {{$tAlias.DownSingular}}CountThenLoader[Q] {
	{{range $rel := $rels -}}
	{{- if not $rel.IsToMany}}{{continue}}{{end -}}
	{{$relAlias := $tAlias.Relationship $rel.Name -}}
	type {{$relAlias}}CountInterface interface {
		LoadCount{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[*dialect.SelectQuery]) error
	}
	{{end}}

	return {{$tAlias.DownSingular}}CountThenLoader[Q]{
		{{range $rel := $rels -}}
		{{- if not $rel.IsToMany}}{{continue}}{{end -}}
		{{$relAlias := $tAlias.Relationship $rel.Name -}}
		{{$relAlias}}: countThenLoadBuilder[Q](
			"{{$relAlias}}",
			func(ctx context.Context, exec bob.Executor, retrieved {{$relAlias}}CountInterface, mods ...bob.Mod[*dialect.SelectQuery]) error {
				return retrieved.LoadCount{{$relAlias}}(ctx, exec, mods...)
			},
		),
		{{end}}
	}
}

{{range $rel := $rels -}}
{{- if not $rel.IsToMany}}{{continue}}{{end -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}

// LoadCount{{$relAlias}} loads the count of {{$relAlias}} into the C struct
func (o *{{$tAlias.UpSingular}}) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if o == nil {
		return nil
	}

	count, err := o.{{relQueryMethodName $tAlias $relAlias}}(mods...).Count(ctx, exec)
	if err != nil {
		return err
	}

	o.C.{{$relAlias}} = &count
	return nil
}

// LoadCount{{$relAlias}} loads the count of {{$relAlias}} for a slice
func (os {{$tAlias.UpSingular}}Slice) LoadCount{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	if len(os) == 0 {
		return nil
	}

	for _, o := range os {
		if err := o.LoadCount{{$relAlias}}(ctx, exec, mods...); err != nil {
			return err
		}
	}

	return nil
}

{{end -}}
{{end -}}
