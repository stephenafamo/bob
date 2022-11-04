package orm

import "fmt"

type BadRelationshipChainError struct {
	Table1  string
	Column1 string
	Value   string
	Table2  string
	Column2 string
}

func (e *BadRelationshipChainError) Error() string {
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
