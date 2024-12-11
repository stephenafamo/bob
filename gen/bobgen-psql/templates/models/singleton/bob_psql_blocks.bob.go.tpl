{{- define "unique_constraint_error_detection_method"}}
func (e *UniqueConstraintError) Is(target error) bool {
	{{$supportedDrivers := list "github.com/lib/pq" "github.com/jackc/pgx" "github.com/jackc/pgx/v4" "github.com/jackc/pgx/v5"}}
	{{if not (has $.DriverName $supportedDrivers)}}
	return false
	{{else}}
		{{$errType := ""}}
		{{$constraintNameField := "ConstraintName"}}
		{{if eq $.DriverName "github.com/jackc/pgx/v4" "github.com/jackc/pgx/v5"}}
			{{$errType = "*pgconn.PgError"}}
			{{if eq $.DriverName "github.com/jackc/pgx/v4" }}
				{{$.Importer.Import "github.com/jackc/pgconn"}}
			{{else}}
				{{$.Importer.Import "github.com/jackc/pgx/v5/pgconn"}}
			{{end}}
		{{else}}
			{{$.Importer.Import $.DriverName}}
			{{$errType = "pgx.PgError"}}
			{{if eq $.DriverName "github.com/lib/pq" }}
				{{$errType = "*pq.Error"}}
				{{$constraintNameField = "Constraint"}}
			{{end}}
		{{end}}
	err, ok := target.({{$errType}})
	if !ok {
		return false
	}
	return err.Code == "23505" && (e.s == "" || err.{{$constraintNameField}} == e.s)
	{{end}}
}
{{end -}}
