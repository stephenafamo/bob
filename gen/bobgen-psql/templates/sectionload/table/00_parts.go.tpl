{{$table := .Table}}
{{- $tAlias := .Aliases.Table $table.Key}}
{{- $.Importer.Import "models" modelsPkg}}
{{- $.Importer.Import "sectionload" sectionloadPkg}}
{{- $qualifiedName := $table.Name}}
{{- if $table.Schema}}{{$qualifiedName = printf "%s.%s" $table.Schema $table.Name}}{{end}}

// {{$tAlias.UpSingular}}Parts is the section parts for {{$qualifiedName}}.
var {{$tAlias.UpSingular}}Parts = sectionload.TableParts[*models.{{$tAlias.UpSingular}}]{
	Schema: {{quote $table.Schema}},
	Table:  {{quote $table.Name}},
	Columns: []sectionload.Column{
		{{range $column := $table.Columns -}}
		{Name: {{quote $column.Name}}, DBType: {{quote $column.DBType}}},
		{{end -}}
	},
	NewBuf: func() sectionload.Buf[*models.{{$tAlias.UpSingular}}] { return new({{$tAlias.DownSingular}}SectionBuf) },
	AsSlice: func(v []*models.{{$tAlias.UpSingular}}) any { return models.{{$tAlias.UpSingular}}Slice(v) },
}

// {{$tAlias.DownSingular}}SectionBuf is the scan buffer for one row of the
// {{$table.Name}} section. Every field is null-tolerant: rows belonging to
// other sections carry NULL in ALL of these columns, including columns that
// are NOT NULL in the database.
type {{$tAlias.DownSingular}}SectionBuf struct {
	{{range $column := $table.Columns -}}
	{{$tAlias.Column $column.Name}} {{$.Types.GetNullable $.CurrentPackage $.Importer $column.Type true}}
	{{end -}}
}

// TargetsFor returns scan destinations for exactly the columns in cols, in
// that order. An empty cols returns every column in
// {{$tAlias.UpSingular}}Parts.Columns order — and that is still the only form
// used until an Edge's ColumnPlan restricts columns.
func (b *{{$tAlias.DownSingular}}SectionBuf) TargetsFor(cols []sectionload.Column) []any {
	if len(cols) == 0 {
		return []any{
			{{range $column := $table.Columns -}}
			&b.{{$tAlias.Column $column.Name}},
			{{end -}}
		}
	}
	out := make([]any, len(cols))
	for i, c := range cols {
		switch c.Name {
		{{range $column := $table.Columns -}}
		case {{quote $column.Name}}:
			out[i] = &b.{{$tAlias.Column $column.Name}}
		{{end -}}
		default:
			panic("sectionload: unknown column " + c.Name + " for {{$qualifiedName}}")
		}
	}
	return out
}

// BuildFor converts the scanned values into the bob model, copying only
// modelCols. A nil modelCols copies every column. A non-nil, empty
// modelCols copies zero columns: a
// restricted ColumnPlan whose Model has no entries (every non-key column
// excluded) still has a distinct, non-nil Model, and must not be confused
// with "unrestricted". Columns scanned into b
// but absent from modelCols (Internal columns) are never copied. Only
// called for rows that belong to this section, so NOT NULL columns are
// guaranteed to carry values when selected.
func (b *{{$tAlias.DownSingular}}SectionBuf) BuildFor(modelCols []sectionload.Column) *models.{{$tAlias.UpSingular}} {
	if modelCols == nil {
		return &models.{{$tAlias.UpSingular}}{
			{{range $column := $table.Columns -}}
			{{$colAlias := $tAlias.Column $column.Name -}}
			{{if $column.Nullable -}}
			{{$colAlias}}: b.{{$colAlias}},
			{{else -}}
			{{$colAlias}}: {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $column.Type (printf "b.%s" $colAlias) true}},
			{{end -}}
			{{end -}}
		}
	}
	m := &models.{{$tAlias.UpSingular}}{}
	for _, c := range modelCols {
		switch c.Name {
		{{range $column := $table.Columns -}}
		{{$colAlias := $tAlias.Column $column.Name -}}
		case {{quote $column.Name}}:
			{{if $column.Nullable -}}
			m.{{$colAlias}} = b.{{$colAlias}}
			{{else -}}
			m.{{$colAlias}} = {{$.Types.UnwrapNullExpr $.CurrentPackage $.Importer $column.Type (printf "b.%s" $colAlias) true}}
			{{end -}}
		{{end -}}
		}
	}
	return m
}
