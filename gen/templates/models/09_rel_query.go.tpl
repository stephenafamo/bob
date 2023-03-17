{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{if $table.Relationships -}}{{$.Importer.Import "github.com/stephenafamo/bob/mods"}}{{end}}

{{range $rel := $table.Relationships -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
func {{$tAlias.DownPlural}}Join{{$relAlias}}[Q dialect.Joinable](ctx context.Context, typ string) bob.Mod[Q] {
	return mods.QueryMods[Q]{
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		dialect.Join[Q](typ, {{$to.UpPlural}}Table.Name(ctx)).On(
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{quote $where.Value}}),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{quote $where.Value}}),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
		),
		{{- end}}
	}
}
{{end}}

{{range $rel := $table.Relationships -}}
{{- $fAlias := $.Aliases.Table $rel.Foreign -}}
{{- $relAlias := $tAlias.Relationship $rel.Name -}}
// {{$relAlias}} starts a query for related objects on {{$rel.Foreign}}
func (o *{{$tAlias.UpSingular}}) {{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) {{$fAlias.UpPlural}}Query {
	return {{$fAlias.UpPlural}}(ctx, exec, append(mods,
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		{{- if gt $index 0 -}}
		sm.InnerJoin({{$from.UpPlural}}Table.Name(ctx)).On(
		{{end -}}
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
				sm.Where({{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
				{{- end -}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{quote $where.Value}}),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{if eq $index 0 -}}sm.Where({{end -}}
				{{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{quote $where.Value}}),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
		{{- if gt $index 0 -}}
		),
		{{- end -}}
		{{- end}}
	)...)
}

{{if le (len $rel.Sides) 1 -}}
{{$side := (index $rel.Sides 0) -}}
{{$fromAlias := $.Aliases.Table $side.From -}}
{{$toAlias := $.Aliases.Table $side.To -}}

func (os {{$tAlias.UpSingular}}Slice) {{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) {{$fAlias.UpPlural}}Query {
  {{if gt (len $side.FromColumns) 0 -}}
	PKArgs := make([]bob.Expression, 0, len(os))
	for _, o := range os {
	PKArgs = append(PKArgs, {{$.Dialect}}.ArgGroup(
		{{- range $index, $local := $side.FromColumns -}}
			{{- $fromCol := index $fromAlias.Columns $local -}}
			o.{{$fromCol}},
		{{- end -}}))
	}
	{{- end}}

	return {{$fAlias.UpPlural}}(ctx, exec, append(mods,
		{{if gt (len $side.FromColumns) 0 -}}
			sm.Where({{$.Dialect}}.Group(
			{{- range $index, $local := $side.FromColumns -}}
				{{- $fromCol := index $fromAlias.Columns $local -}}
				{{- $toCol := index $toAlias.Columns (index $side.ToColumns $index) -}}
				{{$fAlias.UpSingular}}Columns.{{$toCol}},
			{{- end}}).In(PKArgs...)),
		{{- end}}
		{{- range $where := $side.FromWhere}}
			{{- $fromCol := index $fromAlias.Columns $where.Column}}
			sm.Where({{$.Dialect}}.X({{$fromAlias.UpSingular}}Columns.{{$fromCol}}, "=", {{quote $where.Value}})),
		{{- end}}
		{{- range $where := $side.ToWhere}}
			{{- $toCol := index $toAlias.Columns $where.Column}}
			sm.Where({{$.Dialect}}.X({{$toAlias.UpSingular}}Columns.{{$toCol}}, "=", {{quote $where.Value}})),
		{{- end}}
	)...)
}
{{else -}}
{{$firstSide := (index $rel.Sides 0) -}}
{{$firstFrom := $.Aliases.Table $firstSide.From -}}
{{$firstTo := $.Aliases.Table $firstSide.To -}}

{{$lastSide := (index $rel.Sides (sub (len $rel.Sides) 1)) -}}
{{$lastFrom := $.Aliases.Table $lastSide.From -}}
{{$lastTo := $.Aliases.Table $lastSide.To -}}

func (os {{$tAlias.UpSingular}}Slice) {{$relAlias}}(ctx context.Context, exec bob.Executor, mods ...bob.Mod[*dialect.SelectQuery]) {{$fAlias.UpPlural}}Query {
  {{if gt (len $firstSide.FromColumns) 0 -}}
	PKArgs := make([]bob.Expression, 0, len(os))
	for _, o := range os {
	PKArgs = append(PKArgs, {{$.Dialect}}.ArgGroup(
		{{- range $index, $local := $firstSide.FromColumns -}}
			{{- $fromCol := index $firstFrom.Columns $local -}}
			o.{{$fromCol}},
		{{- end -}}))
	}
	{{- end}}


	return {{$fAlias.UpPlural}}(ctx, exec, append(mods,
		{{- range $index := until (len $rel.Sides) | reverse -}}
		{{/* Index counts down */}}
		{{/* This also flips the meaning of $from and $to */}}
		{{- $side := index $rel.Sides $index -}}
		{{- $from := $.Aliases.Table $side.From -}}
		{{- $to := $.Aliases.Table $side.To -}}
		{{- if gt $index 0 -}}
		sm.InnerJoin({{$from.UpPlural}}Table.Name(ctx)).On(
			{{range $i, $local := $side.FromColumns -}}
				{{- $foreign := index $side.ToColumns $i -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns $foreign -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				sm.Where({{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{$where.Value}})),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				sm.Where({{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{$where.Value}})),
			{{- end}}
		),
		{{- else -}}
			{{if gt (len $side.FromColumns) 0 -}}
				sm.Where({{$.Dialect}}.Group(
				{{- range $index, $local := $side.FromColumns -}}
					{{- $fromCol := index $from.Columns $local -}}
					{{- $toCol := index $to.Columns (index $side.ToColumns $index) -}}
					{{$to.UpSingular}}Columns.{{$toCol}},
				{{- end}}).In(PKArgs...)),
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				sm.Where({{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{quote $where.Value}})),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				sm.Where({{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{quote $where.Value}})),
			{{- end}}
		{{- end -}}
		{{- end}}
	)...)
}
{{end -}}


{{end -}}
