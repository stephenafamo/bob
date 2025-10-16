{{if $.Relationships.Get .Table.Key -}}
{{$.Importer.Import "fmt" -}}
{{$.Importer.Import "context" -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm" -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect) -}}

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

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
			for _, rel := range rels {
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
					{{- else -}}
						rel.R.{{$invAlias}} =  o
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

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
					{{- else -}}
						rel.R.{{$invAlias}} =  o
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
      return {{$.Dialect}}.Preload[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice]({{$.Dialect}}.PreloadRel{
          Name: "{{$relAlias}}",
          Sides:  []{{$.Dialect}}.PreloadSide{
            {{- $toTable := $table }}{{/* To be able to access the last one after the loop */}}
            {{range $side := $rel.Sides -}}
            {{- $from := $.Aliases.Table $side.From -}}
            {{- $to := $.Aliases.Table $side.To -}}
            {{- $fromTable := $.Tables.Get $side.From -}}
            {{- $toTable = $.Tables.Get $side.To -}}
            {
              From: {{$from.UpPlural}},
              To: {{$to.UpPlural}},
              {{if $side.FromColumns -}}
              FromColumns: []string{
                {{- range $name := $side.FromColumns -}}
                {{$name | quote}},
                {{- end -}}
              },
              {{- end}}
              {{if $side.ToColumns -}}
              ToColumns: []string{
                {{- range $name := $side.ToColumns -}}
                {{$name | quote}},
                {{- end -}}
              },
              {{end -}}
              {{if $side.FromWhere -}}
              FromWhere: []orm.RelWhere{
                {{range $where := $side.FromWhere -}}
                {
                  Column: {{quote $where.Column}},
                  SQLValue: {{quote $where.SQLValue}},
                  GoValue: {{quote $where.GoValue}},
                },
                {{end -}}
              },
              {{end -}}
              {{if $side.ToWhere -}}
              ToWhere: []orm.RelWhere{
                {{range $where := $side.ToWhere -}}
                {
                  Column: {{quote $where.Column}},
                  SQLValue: {{quote $where.SQLValue}},
                  GoValue: {{quote $where.GoValue}},
                },
                {{end -}}
              },
              {{end -}}
            },
            {{- end}}
          },
        }, {{$fAlias.UpPlural}}.Columns.Names(), opts...)
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
{{- $isToView := $.Tables.RelIsView $rel -}}
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
				rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
			{{- else -}}
				rel.R.{{$invAlias}} =  o
			{{- end}}
		}
	{{else -}}
		{{if $invRel.IsToMany -}}
			related.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
		{{else -}}
			related.R.{{$invAlias}} =  o
		{{- end}}
	{{- end}}
	{{- end}}

	o.R.{{$relAlias}} = related
	return nil
}

// Load{{$relAlias}} loads the {{$tAlias.DownSingular}}'s {{$relAlias}} into the .R struct
{{if le (len $rel.Sides) 1 -}}
func (os {{$tAlias.UpSingular}}Slice) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	{{- $side := (index $rel.Sides 0) -}}
	{{- $fromAlias := $.Aliases.Table $side.From -}}
	{{- $toAlias := $.Aliases.Table $side.To -}}
  if len(os) == 0 {
	  return nil
	}

	{{$fAlias.DownPlural}}, err := os.{{relQueryMethodName $tAlias $relAlias}}(mods...).All(ctx, exec)
	if err != nil {
		return err
	}

	{{if $rel.IsToMany -}}
		for _, o := range os {
			if o == nil {
				continue
			}

			o.R.{{$relAlias}} = nil
		}
	{{- end}}

	for _, o := range os {
		if o == nil {
			continue
		}

		for _, rel := range {{$fAlias.DownPlural}} {
			{{range $index, $local := $side.FromColumns -}}
        {{- $foreign := index $side.ToColumns $index -}}
        {{- $fromCol := $.Tables.GetColumn $side.From $local -}}
        {{- $toCol := $.Tables.GetColumn $side.To $foreign -}}

        {{- $fromColAlias := index $fromAlias.Columns $local -}}
        {{- $toColAlias := index $toAlias.Columns $foreign -}}


        {{if $fromCol.Nullable -}}
          if !{{$.Types.GetNullTypeValid $.CurrentPackage $fromCol.Type (cat "o." $fromColAlias)}} {
            continue
          }
        {{end}}

        {{if $toCol.Nullable -}}
          if !{{$.Types.GetNullTypeValid $.CurrentPackage $toCol.Type (cat "rel." $toColAlias)}} {
            continue
          }
        {{end}}



        {{- $fromColGet := (cat "o." ($fromAlias.Column $local)) -}}
        {{- $toColGet := (cat "rel." ($toAlias.Column $foreign)) -}}
        {{- with $.Types.GetCompareExpr $.CurrentPackage $.Importer $fromCol.Type $fromCol.Nullable $toCol.Nullable -}}
          if !({{replace "AAA" $fromColGet . | replace "BBB" $toColGet}}) {
            continue
          }
        {{- end}}
			{{end}}

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
			{{- end}}

			{{if $rel.IsToMany -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rel)
			{{else -}}
				o.R.{{$relAlias}} =  rel
				break
			{{end -}}
		}
	}

	return nil
}

{{else -}}
func (os {{$tAlias.UpSingular}}Slice) Load{{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) error {
	{{- $firstSide := (index $rel.Sides 0) -}}
	{{- $firstFrom := $.Aliases.Table $firstSide.From -}}
	{{- $firstTo := $.Aliases.Table $firstSide.To -}}
  if len(os) == 0 {
	  return nil
	}

  // since we are changing the columns, we need to check if the original columns were set or add the defaults
  sq := dialect.SelectQuery{}
  for _, mod := range mods {
   mod.Apply(&sq)
  }

	if len(sq.SelectList.Columns) == 0 {
		mods = append(mods, sm.Columns({{$fAlias.UpPlural}}.Columns))
	}

	q := os.{{relQueryMethodName $tAlias $relAlias}}(append(
		mods,
		{{range $index, $local := $firstSide.FromColumns -}}
			{{- $toCol := index $firstTo.Columns (index $firstSide.ToColumns $index) -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			sm.Columns({{$firstTo.UpPlural}}.Columns.{{$toCol}}.As("related_{{$firstSide.From}}.{{$fromCol}}")),
		{{- end}}
	)...)

  {{range $index, $local := $firstSide.FromColumns -}}
    {{- $fromColAlias := index $firstFrom.Columns $local -}}
    {{- $fromCol := $.Tables.GetColumn $firstSide.From $local -}}
    {{- $fromTyp := $.Types.Get $.CurrentPackage $.Importer $fromCol.Type -}}
    {{$fromColAlias}}Slice := []{{$fromTyp}}{}
  {{end}}

	{{$.Importer.Import "github.com/stephenafamo/scan" -}}
  mapper := scan.Mod(scan.StructMapper[*{{$fAlias.UpSingular}}](), func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any, any) error) {
    return func(row *scan.Row) (any, error) {
      {{range $index, $local := $firstSide.FromColumns -}}
        {{- $fromColAlias := index $firstFrom.Columns $local -}}
        {{- $fromCol := $.Tables.GetColumn $firstSide.From $local -}}
        {{- $fromTyp := $.Types.Get $.CurrentPackage $.Importer $fromCol.Type -}}
        {{$fromColAlias}}Slice = append({{$fromColAlias}}Slice, *new({{$fromTyp}}))
        row.ScheduleScanByName("related_{{$firstSide.From}}.{{$fromColAlias}}", &{{$fromColAlias}}Slice[len({{$fromColAlias}}Slice)-1])
      {{end}}

      return nil, nil
    },
    func(any, any) error {
      return nil
    }
  })

	{{$fAlias.DownPlural}}, err := bob.Allx[bob.SliceTransformer[*{{$fAlias.UpSingular}}, {{$fAlias.UpSingular}}Slice]](ctx, exec, q, mapper)
	if err != nil {
		return err
	}

	{{if $rel.IsToMany -}}
		for _, o := range os {
			o.R.{{$relAlias}} = nil
		}
	{{- end}}

	for _, o := range os {
		for i, rel := range {{$fAlias.DownPlural}} {
			{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
      {{- $foreign := index $firstSide.ToColumns $index -}}
      {{- $fromColDef := $.Tables.GetColumn $firstSide.From $local -}}
      {{- $toColDef := $.Tables.GetColumn $firstSide.To $foreign -}}

      {{- $fromColGet := (printf "o.%s" $fromCol) -}}
      {{- $toColGet := (printf "%sSlice[i]" $fromCol) -}}

			{{- $typInfo := $.Types.Index ($.Tables.GetColumn $firstSide.From $local).Type -}}
      {{- with $.Types.GetCompareExpr $.CurrentPackage $.Importer $fromColDef.Type $fromColDef.Nullable $toColDef.Nullable -}}
				if !({{replace "AAA" $fromColGet . | replace "BBB" $toColGet}}) {
					continue
				}
			{{- end}}
			{{end}}

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				{{if $invRel.IsToMany -}}
					rel.R.{{$invAlias}} = append(rel.R.{{$invAlias}}, o)
				{{else -}}
					rel.R.{{$invAlias}} =  o
				{{- end}}
			{{- end}}


			{{if $rel.IsToMany -}}
				o.R.{{$relAlias}} = append(o.R.{{$relAlias}}, rel)
			{{else -}}
				o.R.{{$relAlias}} =  rel
				break
			{{end -}}
		}
	}

	return nil
}

{{end -}}
{{end -}}
{{end -}}
