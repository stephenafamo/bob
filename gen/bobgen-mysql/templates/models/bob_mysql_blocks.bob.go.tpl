{{- define "helpers/where_variables"}}
{{$.Importer.Import (printf "github.com/stephenafamo/bob/dialect/%s/dialect" $.Dialect)}}
var (
	SelectWhere = Where[*dialect.SelectQuery]()
	UpdateWhere = Where[*dialect.UpdateQuery]()
	DeleteWhere = Where[*dialect.DeleteQuery]()
)
{{- end -}}

{{- define "helpers/then_load_variables"}}
var (
	SelectThenLoad = getThenLoaders[*dialect.SelectQuery]()
)
{{end -}}


{{define "unique_constraint_error_detection_method" -}}
{{$.Importer.Import "strings"}}
{{$.Importer.Import "mysqlDriver" "github.com/go-sql-driver/mysql"}}
func (e *UniqueConstraintError) Is(target error) bool {
  err, ok := target.(*mysqlDriver.MySQLError)
  if !ok {
    return false
  }
  return err.Number == 1062 && strings.Contains(err.Message, e.s)
}
{{end -}}
