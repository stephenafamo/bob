// ErrUniqueConstraint captures all unique constraint errors by explicitly leaving `s` empty.
var ErrUniqueConstraint = &UniqueConstraintError{s: ""}

type UniqueConstraintError struct {
  // schema is the schema where the unique constraint is defined.
  schema string
  // table is the name of the table where the unique constraint is defined.
  table string
  // columns are the columns constituting the unique constraint.
  columns []string
	// s is a string uniquely identifying the constraint in the raw error message returned from the database.
	s string
}

func (e *UniqueConstraintError) Error() string {
  return e.s
}

{{block "unique_constraint_error_detection_method" . -}}
func (e *UniqueConstraintError) Is(target error) bool {
  return false
}
{{end}}
