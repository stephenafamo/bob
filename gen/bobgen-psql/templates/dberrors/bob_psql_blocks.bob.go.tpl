{{- define "unique_constraint_error_detection_method"}}
func (e *UniqueConstraintError) Is(target error) bool {
	{{if eq $.Driver "github.com/lib/pq" "github.com/jackc/pgx/v5" "github.com/jackc/pgx/v5/stdlib"}}
		{{$errType := ""}}
		{{$constraintNameField := "ConstraintName"}}
		{{if eq $.Driver "github.com/lib/pq"}}
      {{$.Importer.Import "github.com/lib/pq"}}
      {{$errType = "*pq.Error"}}
      {{$constraintNameField = "Constraint"}}
		{{else if hasPrefix "github.com/jackc/pgx/v5" $.Driver}}
      {{$.Importer.Import "github.com/jackc/pgx/v5/pgconn"}}
			{{$errType = "*pgconn.PgError"}}
		{{else}}
			panic("Unsupported driver {{$.Driver}} for UniqueConstraintError detection")
		{{end}}
    err, ok := target.({{$errType}})
    if !ok {
      return false
    }
    return err.Code == "23505" && (e.s == "" || err.{{$constraintNameField}} == e.s)
	{{else}}
    return false
	{{end}}
}
{{end -}}
