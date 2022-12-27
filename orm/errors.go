package orm

import "fmt"

// RelationshipChainError is the error returned when a wrong value is encountered in a relationship chain
type RelationshipChainError struct {
	Table1  string
	Column1 string
	Value   string
	Table2  string
	Column2 string
}

func (e *RelationshipChainError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf(
			"bad relationship chain: %s.%s <> %q",
			e.Table1, e.Column1,
			e.Value,
		)
	}

	return fmt.Sprintf(
		"bad relationship chain: %s.%s <> %s.%s",
		e.Table1, e.Column1,
		e.Table2, e.Column2,
	)
}
