{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Name -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}

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
		qm.InnerJoin({{$from.UpPlural}}Table.Name()).On(
		{{end -}}
			{{range $i, $local := $side.FromColumns -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns (index $side.ToColumns $i) -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
				qm.Where({{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$.Dialect}}.Arg(o.{{$fromCol}}))),
				{{- end -}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				{{if eq $index 0 -}}qm.Where({{end -}}
				{{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}})),
				{{- if eq $index 0 -}}),{{- end -}}
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				{{if eq $index 0 -}}qm.Where({{end -}}
				{{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}),
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
	{{range $index, $local := $side.FromColumns -}}
		{{- $fromCol := index $fromAlias.Columns $local -}}
		{{$fromCol}}Args := make([]any, 0, len(os))
		for _, o := range os {
			{{$fromCol}}Args = append({{$fromCol}}Args, {{$.Dialect}}.Arg(o.{{$fromCol}}))
		}
	{{- end}}

	return {{$fAlias.UpPlural}}(ctx, exec, append(mods,
		{{range $index, $local := $side.FromColumns -}}
			{{- $fromCol := index $fromAlias.Columns $local -}}
			{{- $toCol := index $toAlias.Columns (index $side.ToColumns $index) -}}
			qm.Where({{$fAlias.UpSingular}}Columns.{{$toCol}}.In({{$fromCol}}Args...)),
		{{- end}}
		{{- range $where := $side.FromWhere}}
			{{- $fromCol := index $fromAlias.Columns $where.Column}}
			qm.Where({{$.Dialect}}.X({{$fromAlias.UpSingular}}Columns.{{$fromCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
		{{- end}}
		{{- range $where := $side.ToWhere}}
			{{- $toCol := index $toAlias.Columns $where.Column}}
			qm.Where({{$.Dialect}}.X({{$toAlias.UpSingular}}Columns.{{$toCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
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
	{{range $index, $local := $firstSide.FromColumns -}}
		{{- $fromCol := index $firstFrom.Columns $local -}}
		{{$fromCol}}Args := make([]any, 0, len(os))
		for _, o := range os {
			{{$fromCol}}Args = append({{$fromCol}}Args, {{$.Dialect}}.Arg(o.{{$fromCol}}))
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
		qm.InnerJoin({{$from.UpPlural}}Table.Name()).On(
		{{end -}}
			{{range $i, $local := $side.FromColumns -}}
				{{- $foreign := index $side.ToColumns $i -}}
				{{- $fromCol := index $from.Columns $local -}}
				{{- $toCol := index $to.Columns $foreign -}}
				{{- if gt $index 0 -}}
				{{$to.UpSingular}}Columns.{{$toCol}}.EQ({{$from.UpSingular}}Columns.{{$fromCol}}),
				{{- else -}}
					qm.Where({{$to.UpSingular}}Columns.{{$toCol}}.In({{$fromCol}}Args...)),
				{{- end}}
			{{- end}}
			{{- range $where := $side.FromWhere}}
				{{- $fromCol := index $from.Columns $where.Column}}
				qm.Where({{$.Dialect}}.X({{$from.UpSingular}}Columns.{{$fromCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
			{{- end}}
			{{- range $where := $side.ToWhere}}
				{{- $toCol := index $to.Columns $where.Column}}
				qm.Where({{$.Dialect}}.X({{$to.UpSingular}}Columns.{{$toCol}}, "=", {{$.Dialect}}.Arg({{$where.Value}}))),
			{{- end}}
		{{- if gt $index 0}}
		),
		{{- end -}}
		{{- end}}
	)...)
}
{{end -}}


{{end -}}
