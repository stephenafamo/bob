package builder

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob/query"
)

// A raw statement with arguments
type statement struct {
	clause string // The clause with ? used for placeholders
	args   []any  // The replacements for the placeholders in order
}

type statementError struct {
	args         int
	placeholders int
	clause       string
}

func (s *statementError) Error() string {
	return fmt.Sprintf(
		"Bad Statement: has %d placeholders but %d args: %s",
		s.placeholders, s.args, s.clause,
	)
}

func (s *statementError) Equal(I error) bool {
	var s2 *statementError
	if errors.As(I, &s2) {
		return s2.args == s.args && s2.placeholders == s.placeholders
	}

	return false
}

func (s statement) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	// replace the args with positional args appropriately
	total := s.convertQuestionMarks(w, d, start)

	if len(s.args) != total {
		return s.args, &statementError{args: len(s.args), placeholders: total, clause: s.clause}
	}

	return s.args, nil
}

// convertQuestionMarks converts each occurrence of ? with $<number>
// where <number> is an incrementing digit starting at startAt.
// If question-mark (?) is escaped using back-slash (\), it will be ignored.
func (s statement) convertQuestionMarks(w io.Writer, d query.Dialect, startAt int) int {
	if startAt == 0 {
		panic("Not a valid start number.")
	}

	paramIndex := 0
	total := 0

	clause := s.clause
	for {
		if paramIndex >= len(clause) {
			break
		}

		clause = clause[paramIndex:]
		paramIndex = strings.IndexByte(clause, '?')

		if paramIndex == -1 {
			w.Write([]byte(clause))
			break
		}

		escapeIndex := strings.Index(clause, `\?`)
		if escapeIndex != -1 && paramIndex > escapeIndex {
			w.Write([]byte(clause[:escapeIndex] + "?"))
			paramIndex++
			continue
		}

		w.Write([]byte(clause[:paramIndex]))
		d.WriteArg(w, startAt)

		total++
		startAt++
		paramIndex++
	}

	return total
}
