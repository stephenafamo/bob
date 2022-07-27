package builder

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob/query"
)

func RawQuery(d query.Dialect, q string, args ...any) query.BaseQuery[Raw] {
	return query.BaseQuery[Raw]{
		Expression: Raw{clause: q, args: args},
		Dialect:    d,
	}
}

// A Raw Raw with arguments
type Raw struct {
	clause string // The clause with ? used for placeholders
	args   []any  // The replacements for the placeholders in order
}

func (r Raw) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	// replace the args with positional args appropriately
	total := r.convertQuestionMarks(w, d, start)

	if len(r.args) != total {
		return r.args, &rawError{args: len(r.args), placeholders: total, clause: r.clause}
	}

	return r.args, nil
}

// convertQuestionMarks converts each occurrence of ? with $<number>
// where <number> is an incrementing digit starting at startAt.
// If question-mark (?) is escaped using back-slash (\), it will be ignored.
func (r Raw) convertQuestionMarks(w io.Writer, d query.Dialect, startAt int) int {
	if startAt == 0 {
		panic("Not a valid start number.")
	}

	paramIndex := 0
	total := 0

	clause := r.clause
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

type rawError struct {
	args         int
	placeholders int
	clause       string
}

func (s *rawError) Error() string {
	return fmt.Sprintf(
		"Bad Statement: has %d placeholders but %d args: %s",
		s.placeholders, s.args, s.clause,
	)
}

func (s *rawError) Equal(I error) bool {
	var s2 *rawError
	if errors.As(I, &s2) {
		return s2.args == s.args && s2.placeholders == s.placeholders
	}

	return false
}
