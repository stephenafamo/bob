{{$.Importer.Import "fmt" -}}

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
			rels, ok := retrieved.({{$.SliceType .Foreign}})
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rels
			o.R.Loaded.{{$relAlias}} = true

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
			for _, rel := range rels {
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
					{{- else -}}
						rel.R.{{$invAlias}} =  o
						rel.R.Loaded.{{$invAlias}} = true
					{{- end}}
				}
			}
			{{- end}}
			return nil
		{{else -}}
			rel, ok := retrieved.(*{{$.ModelType .Foreign}})
			if !ok {
				return fmt.Errorf("{{$tAlias.DownSingular}} cannot load %T as %q", retrieved, name)
			}

			o.R.{{$relAlias}} = rel
			o.R.Loaded.{{$relAlias}} = true

			{{if and (not $.NoBackReferencing) $invRel.Name -}}
			{{- $invAlias := $fAlias.Relationship $invRel.Name -}}
				if rel != nil {
					{{if $invRel.IsToMany -}}
						rel.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
					{{- else -}}
						rel.R.{{$invAlias}} =  o
						rel.R.Loaded.{{$invAlias}} = true
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

{{if $.Relationships.Get .Table.Key -}}
{{$.Importer.Import "context" -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/orm" -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/sm" $.Dialect) -}}

type {{$tAlias.UpSingular}}Preloader struct {
  {{range $rel := $.Relationships.Get $table.Key -}}
  {{- if $rel.IsToMany -}}{{continue}}{{- end -}}
  {{- $relAlias := $tAlias.Relationship $rel.Name -}}
  {{$relAlias}} func(...{{$.Dialect}}.PreloadOption) {{$.Dialect}}.Preloader
  {{end -}}
}

func Build{{$tAlias.UpSingular}}Preloader() {{$tAlias.UpSingular}}Preloader {
  return {{$tAlias.UpSingular}}Preloader{
    {{range $rel := $.Relationships.Get $table.Key -}}
    {{- if $rel.IsToMany -}}{{continue}}{{- end -}}
    {{- $relAlias := $tAlias.Relationship $rel.Name -}}
    {{- $fAlias := $.Aliases.Table $rel.Foreign -}}
    {{$relAlias}}: func(opts ...{{$.Dialect}}.PreloadOption) {{$.Dialect}}.Preloader {
      return {{$.Dialect}}.Preload[*{{$.ModelType $rel.Foreign}}, {{$.SliceType $rel.Foreign}}]({{$.Dialect}}.PreloadRel{
          Name: "{{$relAlias}}",
          Sides:  []{{$.Dialect}}.PreloadSide{
            {{- $toTable := $table }}{{/* To be able to access the last one after the loop */}}
            {{range $side := $rel.Sides -}}
            {{- $from := $.Aliases.Table $side.From -}}
            {{- $to := $.Aliases.Table $side.To -}}
            {{- $fromTable := $.AllTables.Get $side.From -}}
            {{- $toTable = $.AllTables.Get $side.To -}}
            {
              From: {{$.TableVar $side.From}},
              To: {{$.TableVar $side.To}},
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
        }, {{$.TableVar $rel.Foreign}}.Columns.Names(), opts...)
    },
    {{end -}}
  }
}

func (l {{$tAlias.UpSingular}}Preloader) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error) {
	paths := make([]string, 0, len(expands))
	for path := range expands {
		paths = append(paths, path)
	}

	return l.ForExpandPaths(paths, opts...)
}

func (l {{$tAlias.UpSingular}}Preloader) ForExpandPaths(paths []string, opts ...ExpandLoadOption) ([]bob.Mod[*dialect.SelectQuery], error) {
	options := newExpandLoadOptions(opts...)
	tree, err := buildExpandTree(paths, options.maxDepth)
	if err != nil {
		return nil, err
	}

	preloadOpts, err := l.forExpandTree(tree, 0, options)
	if err != nil {
		return nil, err
	}

	mods := make([]bob.Mod[*dialect.SelectQuery], 0, len(preloadOpts))
	for _, opt := range preloadOpts {
		mod, ok := opt.(bob.Mod[*dialect.SelectQuery])
		if !ok {
			return nil, fmt.Errorf("expand preload option %T is not a select query mod", opt)
		}
		mods = append(mods, mod)
	}

	return mods, nil
}

func (l {{$tAlias.UpSingular}}Preloader) forExpandTree(tree expandTree, depth int, opts expandLoadOptions) ([]{{$.Dialect}}.PreloadOption, error) {
	if opts.maxDepth >= 0 && depth > opts.maxDepth {
		return nil, fmt.Errorf("expand path %q exceeds max depth %d", tree.path, opts.maxDepth)
	}

	mods := make([]{{$.Dialect}}.PreloadOption, 0, len(tree.children))
	for _, segment := range tree.sortedSegments() {
		child := *tree.children[segment]
		if child.computedTerminal(opts) {
			continue
		}

		switch segment {
		{{range $rel := $.Relationships.Get $table.Key -}}
		{{- if $rel.IsToMany -}}{{continue}}{{- end -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
		case {{snakecase $relAlias | quote}}:
			var childOpts []{{$.Dialect}}.PreloadOption
			{{if and ($.HasExpandPreloader $rel.Foreign) ($.SameModelSplitComponent $rel.Foreign) -}}
			var err error
			childOpts, err = Preload.{{$fAlias.UpSingular}}.forExpandTree(child, depth+1, opts)
			if err != nil {
				return nil, err
			}
			{{else -}}
			if len(child.children) > 0 {
				{{if $.HasExpandPreloader $rel.Foreign -}}
				return nil, fmt.Errorf("expand path %q cannot be nested because {{$fAlias.UpSingular}} is generated in another model component", child.path)
				{{else -}}
				return nil, fmt.Errorf("expand path %q cannot be nested because {{$fAlias.UpSingular}} has no generated preload relationships", child.path)
				{{end -}}
			}
			{{end -}}
			mods = append(mods, l.{{$relAlias}}(append(childOpts, {{$.Dialect}}.PreloadAs({{snakecase $relAlias | quote}}))...))
		{{end -}}
		default:
			return nil, fmt.Errorf("expand segment %q does not match a relationship on {{$tAlias.UpSingular}}", segment)
		}
	}

	return mods, nil
}


type {{$tAlias.UpSingular}}ThenLoader[Q orm.Loadable] struct {
  {{range $rel := $.Relationships.Get $table.Key -}}
  {{- $relAlias := $tAlias.Relationship $rel.Name -}}
  {{$relAlias}} func(...bob.Mod[*dialect.SelectQuery]) orm.Loader[Q]
  {{end -}}
}

func Build{{$tAlias.UpSingular}}ThenLoader[Q orm.Loadable]() {{$tAlias.UpSingular}}ThenLoader[Q] {
  {{range $rel := $.Relationships.Get $table.Key -}}
    {{$relAlias := $tAlias.Relationship $rel.Name -}}
    type {{$relAlias}}LoadInterface interface{
      Load{{$relAlias}}(context.Context, bob.Executor, ...bob.Mod[*dialect.SelectQuery]) error
    }
  {{end}}

  return {{$tAlias.UpSingular}}ThenLoader[Q]{
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

func (l {{$tAlias.UpSingular}}ThenLoader[Q]) ForExpandMap(expands map[string]struct{}, opts ...ExpandLoadOption) ([]bob.Mod[Q], error) {
	paths := make([]string, 0, len(expands))
	for path := range expands {
		paths = append(paths, path)
	}

	return l.ForExpandPaths(paths, opts...)
}

func (l {{$tAlias.UpSingular}}ThenLoader[Q]) ForExpandPaths(paths []string, opts ...ExpandLoadOption) ([]bob.Mod[Q], error) {
	options := newExpandLoadOptions(opts...)
	tree, err := buildExpandTree(paths, options.maxDepth)
	if err != nil {
		return nil, err
	}

	return l.forExpandTree(tree, 0, options)
}

func (l {{$tAlias.UpSingular}}ThenLoader[Q]) forExpandTree(tree expandTree, depth int, opts expandLoadOptions) ([]bob.Mod[Q], error) {
	if opts.maxDepth >= 0 && depth > opts.maxDepth {
		return nil, fmt.Errorf("expand path %q exceeds max depth %d", tree.path, opts.maxDepth)
	}

	mods := make([]bob.Mod[Q], 0, len(tree.children))
	for _, segment := range tree.sortedSegments() {
		child := *tree.children[segment]
		if child.computedTerminal(opts) {
			continue
		}

		switch segment {
		{{range $rel := $.Relationships.Get $table.Key -}}
		{{- $relAlias := $tAlias.Relationship $rel.Name -}}
		{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
		case {{snakecase $relAlias | quote}}:
			{{if and ($.HasExpandThenLoader $rel.Foreign) ($.SameModelSplitComponent $rel.Foreign) -}}
			childMods, err := SelectThenLoad.{{$fAlias.UpSingular}}.forExpandTree(child, depth+1, opts)
			if err != nil {
				return nil, err
			}
			mods = append(mods, l.{{$relAlias}}(childMods...))
			{{else -}}
			if len(child.children) > 0 {
				{{if $.HasExpandThenLoader $rel.Foreign -}}
				return nil, fmt.Errorf("expand path %q cannot be nested because {{$fAlias.UpSingular}} is generated in another model component", child.path)
				{{else -}}
				return nil, fmt.Errorf("expand path %q cannot be nested because {{$fAlias.UpSingular}} has no generated expand relationships", child.path)
				{{end -}}
			}
			mods = append(mods, l.{{$relAlias}}())
			{{end -}}
		{{end -}}
		default:
			return nil, fmt.Errorf("expand segment %q does not match a relationship on {{$tAlias.UpSingular}}", segment)
		}
	}

	return mods, nil
}

{{range $rel := $.Relationships.Get $table.Key -}}
{{- $isToView := $.AllTables.RelIsView $rel -}}
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
	o.R.Loaded.{{$relAlias}} = false

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
				rel.R.Loaded.{{$invAlias}} = true
			{{- end}}
		}
	{{else -}}
		{{if $invRel.IsToMany -}}
			related.R.{{$invAlias}} = {{$tAlias.UpSingular}}Slice{o}
		{{else -}}
			related.R.{{$invAlias}} =  o
			related.R.Loaded.{{$invAlias}} = true
		{{- end}}
	{{- end}}
	{{- end}}

	o.R.{{$relAlias}} = related
	o.R.Loaded.{{$relAlias}} = true
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

	for _, o := range os {
		if o == nil {
			continue
		}

		o.R.{{$relAlias}} = nil
		o.R.Loaded.{{$relAlias}} = true
	}

	for _, o := range os {
		if o == nil {
			continue
		}

		for _, rel := range {{$fAlias.DownPlural}} {
			{{range $index, $local := $side.FromColumns -}}
        {{- $foreign := index $side.ToColumns $index -}}
        {{- $fromCol := $.AllTables.GetColumn $side.From $local -}}
        {{- $toCol := $.AllTables.GetColumn $side.To $foreign -}}

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
        {{- $toColGet = $.Types.TypeCastExpr $.CurrentPackage $.Importer $fromCol.Type $toCol.Type $toColGet -}}
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
					rel.R.Loaded.{{$invAlias}} = true
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
		mods = append(mods, sm.Columns({{$.TableVar $rel.Foreign}}.Columns))
	}

	q := os.{{relQueryMethodName $tAlias $relAlias}}(append(
		mods,
		{{range $index, $local := $firstSide.FromColumns -}}
			{{- $toCol := index $firstTo.Columns (index $firstSide.ToColumns $index) -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			sm.Columns({{$.TableVar $firstSide.To}}.Columns.{{$toCol}}.As("related_{{$firstSide.From}}.{{$fromCol}}")),
		{{- end}}
	)...)

  {{range $index, $local := $firstSide.FromColumns -}}
    {{- $fromColAlias := index $firstFrom.Columns $local -}}
    {{- $fromCol := $.AllTables.GetColumn $firstSide.From $local -}}
    {{- $fromTyp := $.Types.Get $.CurrentPackage $.Importer $fromCol.Type -}}
    {{$fromColAlias}}Slice := []{{$fromTyp}}{}
  {{end}}

	{{$.Importer.Import "github.com/stephenafamo/scan" -}}
  mapper := scan.Mod(scan.StructMapper[*{{$.ModelType $rel.Foreign}}](), func(ctx context.Context, cols []string) (scan.BeforeFunc, func(any, any) error) {
    return func(row *scan.Row) (any, error) {
      {{range $index, $local := $firstSide.FromColumns -}}
        {{- $fromColAlias := index $firstFrom.Columns $local -}}
        {{- $fromCol := $.AllTables.GetColumn $firstSide.From $local -}}
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

	{{$fAlias.DownPlural}}, err := bob.Allx[bob.SliceTransformer[*{{$.ModelType $rel.Foreign}}, {{$.SliceType $rel.Foreign}}]](ctx, exec, q, mapper)
	if err != nil {
		return err
	}

	for _, o := range os {
		o.R.{{$relAlias}} = nil
		o.R.Loaded.{{$relAlias}} = true
	}

	for _, o := range os {
		for i, rel := range {{$fAlias.DownPlural}} {
			{{range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
      {{- $foreign := index $firstSide.ToColumns $index -}}
      {{- $fromColDef := $.AllTables.GetColumn $firstSide.From $local -}}
      {{- $toColDef := $.AllTables.GetColumn $firstSide.To $foreign -}}

      {{- $fromColGet := (printf "o.%s" $fromCol) -}}
      {{- $toColGet := (printf "%sSlice[i]" $fromCol) -}}

			{{- $typInfo := $.Types.Index ($.AllTables.GetColumn $firstSide.From $local).Type -}}
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
					rel.R.Loaded.{{$invAlias}} = true
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
