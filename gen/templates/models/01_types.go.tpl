{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}

// {{$tAlias.UpSingular}} is an object representing the database table.
type {{$tAlias.UpSingular}} struct {
	{{- range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
  {{- $typDef :=  index $.Types $column.Type -}}
	{{- $colTyp := or $typDef.AliasOf $column.Type -}}
	{{- $.Importer.ImportList $typDef.Imports -}}
	{{- $orig_col_name := $column.Name -}}
	{{- if $column.Nullable -}}
		{{- $colTyp = printf "null.Val[%s]" $colTyp -}}
		{{ $.Importer.Import "github.com/aarondl/opt/null"}}
	{{- end -}}
	{{- if trim $column.Comment}}{{range $column.Comment | splitList "\n"}}
		// {{ . }}
	{{- end}}{{end -}}
	{{- if ignore $table.Key $orig_col_name $.TagIgnore}}
	{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}" {{generateIgnoreTags $.Tags | trim}}`
	{{- else}}{{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
		{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}" {{generateTags $.Tags $tagName | trim}}`
	{{- end -}}
	{{- end -}}
	{{block "model/fields/additional" $}}{{end}}
	{{- if $.Relationships.Get $table.Key}}

	R {{$tAlias.DownSingular}}R `db:"-" {{generateTags $.Tags $.RelationTag | trim}}`
	{{end -}}
}

// {{$tAlias.UpSingular}}Slice is an alias for a slice of pointers to {{$tAlias.UpSingular}}.
// This should almost always be used instead of []*{{$tAlias.UpSingular}}.
type {{$tAlias.UpSingular}}Slice []*{{$tAlias.UpSingular}}

{{block "model_and_query" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{if not $table.Constraints.Primary -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} view
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewViewx[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]("{{$table.Schema}}","{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} view
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- else -}}
	// {{$tAlias.UpPlural}} contains methods to work with the {{$table.Name}} table
	var {{$tAlias.UpPlural}} = {{$.Dialect}}.NewTablex[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice, *{{$tAlias.UpSingular}}Setter]("{{$table.Schema}}","{{$table.Name}}")
	// {{$tAlias.UpPlural}}Query is a query on the {{$table.Name}} table
	type {{$tAlias.UpPlural}}Query = *{{$.Dialect}}.ViewQuery[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]
{{- end}}
{{- end}}

// {{$tAlias.UpPlural}}Stmt is a prepared statment on {{$table.Name}}
type {{$tAlias.UpPlural}}Stmt = bob.QueryStmt[*{{$tAlias.UpSingular}}, {{$tAlias.UpSingular}}Slice]

{{if $.Relationships.Get $table.Key -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
// {{$tAlias.DownSingular}}R is where relationships are stored.
type {{$tAlias.DownSingular}}R struct {
	{{range $.Relationships.Get $table.Key -}}
	{{- $ftable := $.Aliases.Table .Foreign -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{if .IsToMany -}}
		{{$relAlias}} {{$ftable.UpSingular}}Slice {{if $.Tags}}`{{generateTags $.Tags $relAlias | trim}}`{{end}} // {{.Name}}
	{{else -}}
		{{$relAlias}} *{{$ftable.UpSingular}} {{if $.Tags}}`{{generateTags $.Tags $relAlias | trim}}`{{end}} // {{.Name}}
	{{end}}{{end -}}
}
{{- end}}

{{if or $table.Constraints.Primary ($.Relationships.Get $table.Key) -}}
// {{$tAlias.UpSingular}}Setter is used for insert/upsert/update operations
// All values are optional, and do not have to be set
// Generated columns are not included
type {{$tAlias.UpSingular}}Setter struct {
	{{- range $column := $table.Columns -}}
	{{- if $column.Generated}}{{continue}}{{end -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{- $orig_col_name := $column.Name -}}
  {{- $typDef :=  index $.Types $column.Type -}}
  {{- $colTyp := or $typDef.AliasOf $column.Type -}}
		{{- if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			{{- $colTyp = printf "omitnull.Val[%s]" $colTyp -}}
		{{- else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			{{- $colTyp = printf "omit.Val[%s]" $colTyp -}}
		{{- end -}}
		{{- if ignore $table.Key $orig_col_name $.TagIgnore}}
		{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}" {{generateIgnoreTags $.Tags | trim}}`
		{{- else}}{{$tagName := columnTagName $.StructTagCasing $column.Name $colAlias}}
			{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}" {{generateTags $.Tags $tagName | trim}}`
		{{- end -}}		
	{{end -}}
}

func (s {{$tAlias.UpSingular}}Setter) SetColumns() []string {
  vals := make([]string, 0, {{len $table.NonGeneratedColumns}})
	{{range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !s.{{$colAlias}}.IsUnset() {
			vals = append(vals, {{printf "%q" $column.Name}})
		}

	{{end -}}

	return vals
}

func (s {{$tAlias.UpSingular}}Setter) Overwrite(t *{{$tAlias.UpSingular}}) {
	{{- range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !s.{{$colAlias}}.IsUnset() {
			{{- if not $column.Nullable -}}
				t.{{$colAlias}}, _ = s.{{$colAlias}}.Get()
			{{- else -}}
				t.{{$colAlias}}, _ = s.{{$colAlias}}.GetNull()
			{{- end -}}
		}
	{{end -}}
}

{{block "setter_insert_mod" . -}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/im" $.Dialect)}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) InsertMod() bob.Mod[*dialect.InsertQuery] {
  vals := make([]bob.Expression, {{len $table.NonGeneratedColumns}})
	{{range $index, $column := $table.NonGeneratedColumns -}}
		{{$colAlias := $tAlias.Column $column.Name -}}
		if s.{{$colAlias}}.IsUnset() {
			vals[{{$index}}] = {{$.Dialect}}.Raw("DEFAULT")
		} else {
			vals[{{$index}}] = {{$.Dialect}}.Arg(s.{{$colAlias}})
		}

	{{end -}}

	return im.Values(vals...)
}
{{- end}}

{{block "setter_update_mod" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) Apply(q *dialect.UpdateQuery) {
  um.Set(s.Expressions()...).Apply(q)
}
{{- end}}

{{block "setter_expressions" . -}}
{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
func (s {{$tAlias.UpSingular}}Setter) Expressions(prefix ...string) []bob.Expression {
  exprs := make([]bob.Expression, 0, {{len $table.NonGeneratedColumns}})

  {{$.Importer.Import "github.com/stephenafamo/bob/expr" }}
	{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/um" $.Dialect)}}
	{{range $column := $table.Columns -}}
	{{if $column.Generated}}{{continue}}{{end -}}
	{{$colAlias := $tAlias.Column $column.Name -}}
		if !s.{{$colAlias}}.IsUnset() {
      exprs = append(exprs, expr.Join{Sep: " = ", Exprs: []bob.Expression{
        {{$.Dialect}}.Quote(append(prefix, "{{$column.Name}}")...), 
        {{$.Dialect}}.Arg(s.{{$colAlias}}),
      }})
		}

	{{end -}}

  return exprs
}
{{- end}}


{{- end}}

type {{$tAlias.DownSingular}}ColumnNames struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} string
  {{end -}}
}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
var {{$tAlias.UpSingular}}Columns = build{{$tAlias.UpSingular}}Columns({{quote $table.Key}})

type {{$tAlias.DownSingular}}Columns struct {
  tableAlias string
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} {{$.Dialect}}.Expression
	{{end -}}
}

func (c {{$tAlias.DownSingular}}Columns) Alias() string {
  return c.tableAlias
}

func ({{$tAlias.DownSingular}}Columns) AliasedAs(alias string) {{$tAlias.DownSingular}}Columns {
  return build{{$tAlias.UpSingular}}Columns(alias)
}

func build{{$tAlias.UpSingular}}Columns(alias string) {{$tAlias.DownSingular}}Columns {
  return {{$tAlias.DownSingular}}Columns{
    tableAlias: alias,
    {{range $column := $table.Columns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{$colAlias}}: {{$.Dialect}}.Quote(alias, {{quote $column.Name}}),
    {{end -}}
  }
}


type {{$tAlias.DownSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
	{{range $column := $table.Columns -}}
    {{- $colAlias := $tAlias.Column $column.Name -}}
    {{- $colTyp := or (index $.Types $column.Type).AliasOf $column.Type -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} {{$.Dialect}}.WhereNullMod[Q, {{$colTyp}}]
		{{- else -}}
			{{$colAlias}} {{$.Dialect}}.WhereMod[Q, {{$colTyp}}]
		{{- end}}
  {{end -}}
}

func ({{$tAlias.DownSingular}}Where[Q]) AliasedAs(alias string) {{$tAlias.DownSingular}}Where[Q] {
	return build{{$tAlias.UpSingular}}Where[Q](build{{$tAlias.UpSingular}}Columns(alias))
}

func build{{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable](cols {{$tAlias.DownSingular}}Columns) {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			{{range $column := $table.Columns -}}
      {{- $colTyp := or (index $.Types $column.Type).AliasOf $column.Type -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$colTyp}}](cols.{{$colAlias}}),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$colTyp}}](cols.{{$colAlias}}),
				{{- end}}
			{{end -}}
	}
}

{{if $.Relationships.Get $table.Key -}}
{{$.Importer.Import "context"}}
type {{$tAlias.DownSingular}}Joins[Q dialect.Joinable] struct {
  typ string
	{{range $.Relationships.Get $table.Key -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
  {{- $fAlias := $.Aliases.Table .Foreign -}}
	{{$relAlias}} func(context.Context) modAs[Q, {{$fAlias.DownSingular}}Columns]
  {{end -}}
}

func (j {{$tAlias.DownSingular}}Joins[Q]) aliasedAs(alias string) {{$tAlias.DownSingular}}Joins[Q] {
  return build{{$tAlias.UpSingular}}Joins[Q](build{{$tAlias.UpSingular}}Columns(alias), j.typ)
}

func build{{$tAlias.UpSingular}}Joins[Q dialect.Joinable](cols {{$tAlias.DownSingular}}Columns, typ string) {{$tAlias.DownSingular}}Joins[Q] {
  return {{$tAlias.DownSingular}}Joins[Q]{
    typ: typ,
		{{range $.Relationships.Get $table.Key -}}
			{{$ftable := $.Aliases.Table .Foreign -}}
			{{$relAlias := $tAlias.Relationship .Name -}}
			{{$relAlias}}: {{$tAlias.DownPlural}}Join{{$relAlias}}[Q](cols, typ),
		{{end}}
	}
}
{{- end}}
