{{if .QueryFile.Queries}}

{{range $queryIndex, $query := $.QueryFile.Queries}}
{{- if $query.Config.Batch -}}
{{$upperName := titleCase $query.Name}}
{{$lowerName := untitle $query.Name}}
{{$flatArgs := $query.ArgsByPosition}}

{{$queryResultTypeOne := printf "%sRow" $upperName}}
{{if $query.Config.ResultTypeOne}}
  {{$queryResultTypeOne = $.Types.Get $.CurrentPackage $.Importer $query.Config.ResultTypeOne}}
{{end}}

{{$queryType := (lower $query.Type.String | titleCase)}}

{{/* Only generate batch helpers for queries that return results */}}
{{if $query.Columns}}

{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/drivers/pgx"}}
{{$.Importer.Import "github.com/stephenafamo/scan"}}

// {{$upperName}}Batch allows batching multiple {{$upperName}} calls for efficient execution
type {{$upperName}}Batch struct {
	qb      *pgx.QueuedBatch
	{{if eq $queryType "Select" -}}
	results []{{$queryResultTypeOne}}
	{{- else if eq $queryType "Insert" -}}
	results []{{$queryResultTypeOne}}
	{{- else if eq $queryType "Update" -}}
	results []{{$queryResultTypeOne}}
	{{- else -}}
	results []{{$queryResultTypeOne}}
	{{- end}}
}

// New{{$upperName}}Batch creates a new batch for {{$upperName}} queries
func New{{$upperName}}Batch() *{{$upperName}}Batch {
	return &{{$upperName}}Batch{
		qb: pgx.NewQueuedBatch(),
	}
}

{{$args := list }}
{{range $arg := $query.Args -}}
  {{ $argName := titleCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.CurrentPackage $.Importer $.Types) }}

  {{if gt (len $arg.Children) 0}}
    {{ $argType = printf "%s_%s" $upperName $argName }}
  {{end}}

  {{$args = append $args (printf "%s %s" $argName $argType) }}
{{end}}

// Queue adds a {{$upperName}} query to the batch
func (b *{{$upperName}}Batch) Queue(ctx context.Context, {{join ", " $args}}) error {
	query := {{$upperName}}({{range $i, $arg := $query.Args}}{{if $i}}, {{end}}{{titleCase $arg.Col.Name}}{{end}})

	{{if gt (len $query.Columns) 1 -}}
	var result {{$queryResultTypeOne}}
	{{- if eq $queryType "Select" -}}
	err := pgx.QueueSelectRow(b.qb, ctx, query, scan.StructMapper[{{$queryResultTypeOne}}](), &result)
	{{- else if eq $queryType "Insert" -}}
	err := pgx.QueueInsertRowReturning(b.qb, ctx, query, scan.StructMapper[{{$queryResultTypeOne}}](), &result)
	{{- else if eq $queryType "Update" -}}
	err := pgx.QueueUpdateRowReturning(b.qb, ctx, query, scan.StructMapper[{{$queryResultTypeOne}}](), &result)
	{{- else if eq $queryType "Delete" -}}
	err := pgx.QueueDeleteRowReturning(b.qb, ctx, query, scan.StructMapper[{{$queryResultTypeOne}}](), &result)
	{{- else -}}
	err := pgx.QueueSelectRow(b.qb, ctx, query, scan.StructMapper[{{$queryResultTypeOne}}](), &result)
	{{- end}}
	if err != nil {
		return err
	}
	b.results = append(b.results, result)
	return nil
	{{- else -}}
	{{- $col := index $query.Columns 0 -}}
	{{- $colType := $col.Type $.CurrentPackage $.Importer $.Types -}}
	var result {{$colType}}
	{{- if eq $queryType "Select" -}}
	err := pgx.QueueSelectRow(b.qb, ctx, query, scan.SingleColumnMapper[{{$colType}}](&result), &result)
	{{- else if eq $queryType "Insert" -}}
	err := pgx.QueueInsertRowReturning(b.qb, ctx, query, scan.SingleColumnMapper[{{$colType}}](&result), &result)
	{{- else if eq $queryType "Update" -}}
	err := pgx.QueueUpdateRowReturning(b.qb, ctx, query, scan.SingleColumnMapper[{{$colType}}](&result), &result)
	{{- else if eq $queryType "Delete" -}}
	err := pgx.QueueDeleteRowReturning(b.qb, ctx, query, scan.SingleColumnMapper[{{$colType}}](&result), &result)
	{{- else -}}
	err := pgx.QueueSelectRow(b.qb, ctx, query, scan.SingleColumnMapper[{{$colType}}](&result), &result)
	{{- end}}
	if err != nil {
		return err
	}
	b.results = append(b.results, {{$queryResultTypeOne}}{{"{"}}{{$col.Name}}: result{{"}"}})
	return nil
	{{- end}}
}

// Execute runs the batch and populates all results
// All queued queries are executed in a single round trip to the database
func (b *{{$upperName}}Batch) Execute(ctx context.Context, exec bob.Executor) error {
	return b.qb.Execute(ctx, exec)
}

// Results returns all results after Execute has been called
func (b *{{$upperName}}Batch) Results() []{{$queryResultTypeOne}} {
	return b.results
}

// Len returns the number of queries in the batch
func (b *{{$upperName}}Batch) Len() int {
	return len(b.results)
}

{{end}}
{{/* End if $query.Columns */}}

{{/* For queries without RETURNING (INSERT/UPDATE/DELETE without result) */}}
{{if not $query.Columns}}

{{$.Importer.Import "context"}}
{{$.Importer.Import "github.com/stephenafamo/bob"}}
{{$.Importer.Import "github.com/stephenafamo/bob/drivers/pgx"}}

// {{$upperName}}Batch allows batching multiple {{$upperName}} calls for efficient execution
type {{$upperName}}Batch struct {
	qb *pgx.QueuedBatch
}

// New{{$upperName}}Batch creates a new batch for {{$upperName}} queries
func New{{$upperName}}Batch() *{{$upperName}}Batch {
	return &{{$upperName}}Batch{
		qb: pgx.NewQueuedBatch(),
	}
}

{{$args := list }}
{{range $arg := $query.Args -}}
  {{ $argName := titleCase $arg.Col.Name }}
  {{ $argType := ($arg.Type $.CurrentPackage $.Importer $.Types) }}

  {{if gt (len $arg.Children) 0}}
    {{ $argType = printf "%s_%s" $upperName $argName }}
  {{end}}

  {{$args = append $args (printf "%s %s" $argName $argType) }}
{{end}}

// Queue adds a {{$upperName}} query to the batch
func (b *{{$upperName}}Batch) Queue(ctx context.Context, {{join ", " $args}}) error {
	query := {{$upperName}}({{range $i, $arg := $query.Args}}{{if $i}}, {{end}}{{titleCase $arg.Col.Name}}{{end}})
	return pgx.QueueExec(b.qb, ctx, query)
}

// Execute runs the batch
// All queued queries are executed in a single round trip to the database
func (b *{{$upperName}}Batch) Execute(ctx context.Context, exec bob.Executor) error {
	return b.qb.Execute(ctx, exec)
}

{{end}}
{{/* End if not $query.Columns */}}

{{end}}
{{/* End if $query.Config.Batch */}}
{{end}}
{{/* End range queries */}}

{{end}}
{{/* End if .QueryFile.Queries */}}
