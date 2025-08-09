{{- define "unique_constraint_error_detection_method" -}}
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
