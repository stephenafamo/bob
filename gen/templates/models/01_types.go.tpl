{{$table := .Table}}
{{$tAlias := .Aliases.Table $table.Key -}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}

// {{$tAlias.UpSingular}} is an object representing the database table.
type {{$tAlias.UpSingular}} struct {
	{{- range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{- $colTyp := $column.Type -}}
	{{- $.Importer.ImportList (index $.Types $column.Type).Imports -}}
	{{- $orig_col_name := $column.Name -}}
	{{- if $column.Nullable -}}
		{{- $colTyp = printf "null.Val[%s]" $column.Type -}}
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
	{{- $colTyp := "" -}}
		{{- if $column.Nullable -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omitnull" -}}
			{{- $colTyp = printf "omitnull.Val[%s]" $column.Type -}}
		{{- else -}}
			{{- $.Importer.Import "github.com/aarondl/opt/omit" -}}
			{{- $colTyp = printf "omit.Val[%s]" $column.Type -}}
		{{- end -}}
		{{$colAlias}} {{$colTyp}} `db:"{{dbTag $table $column}}"`
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

{{if $.Relationships.Get $table.Key -}}
type {{$tAlias.DownSingular}}RelationshipJoins[Q dialect.Joinable] struct {
	{{range $.Relationships.Get $table.Key -}}
	{{- $relAlias := $tAlias.Relationship .Name -}}
	{{$relAlias}} bob.Mod[Q]
  {{end -}}
}

func build{{$tAlias.UpSingular}}RelationshipJoins[Q dialect.Joinable](ctx context.Context, typ string) {{$tAlias.DownSingular}}RelationshipJoins[Q] {
  return {{$tAlias.DownSingular}}RelationshipJoins[Q]{
		{{range $.Relationships.Get $table.Key -}}
			{{$ftable := $.Aliases.Table .Foreign -}}
			{{$relAlias := $tAlias.Relationship .Name -}}
			{{$relAlias}}: {{$tAlias.DownPlural}}Join{{$relAlias}}[Q](ctx, typ),
		{{end}}
	}
}

{{$.Importer.Import "github.com/stephenafamo/bob/clause"}}
func {{$tAlias.DownPlural}}Join[Q dialect.Joinable](ctx context.Context) joinSet[{{$tAlias.DownSingular}}RelationshipJoins[Q]] {
	return joinSet[{{$tAlias.DownSingular}}RelationshipJoins[Q]] {
	  InnerJoin: build{{$tAlias.UpSingular}}RelationshipJoins[Q](ctx, clause.InnerJoin),
	  LeftJoin: build{{$tAlias.UpSingular}}RelationshipJoins[Q](ctx, clause.LeftJoin),
	  RightJoin: build{{$tAlias.UpSingular}}RelationshipJoins[Q](ctx, clause.RightJoin),
	}
}
{{- end}}

{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s" $.Dialect)}}
var {{$tAlias.UpSingular}}Columns = struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}} {{$.Dialect}}.Expression
	{{end -}}
}{
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
	{{$colAlias}}: {{$.Dialect}}.Quote({{quote $table.Key}}, {{quote $column.Name}}),
	{{end -}}
}

type {{$tAlias.DownSingular}}Where[Q {{$.Dialect}}.Filterable] struct {
	{{range $column := $table.Columns -}}
	{{- $colAlias := $tAlias.Column $column.Name -}}
		{{- if $column.Nullable -}}
			{{$colAlias}} {{$.Dialect}}.WhereNullMod[Q, {{$column.Type}}]
		{{- else -}}
			{{$colAlias}} {{$.Dialect}}.WhereMod[Q, {{$column.Type}}]
		{{- end}}
  {{end -}}
}

func {{$tAlias.UpSingular}}Where[Q {{$.Dialect}}.Filterable]() {{$tAlias.DownSingular}}Where[Q] {
	return {{$tAlias.DownSingular}}Where[Q]{
			{{range $column := $table.Columns -}}
			{{- $colAlias := $tAlias.Column $column.Name -}}
				{{- if $column.Nullable -}}
					{{$colAlias}}: {{$.Dialect}}.WhereNull[Q, {{$column.Type}}]({{$tAlias.UpSingular}}Columns.{{$colAlias}}),
				{{- else -}}
					{{$colAlias}}: {{$.Dialect}}.Where[Q, {{$column.Type}}]({{$tAlias.UpSingular}}Columns.{{$colAlias}}),
				{{- end}}
			{{end -}}
	}
}
