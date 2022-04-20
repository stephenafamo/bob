package expr

import (
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/typesql/query"
)

func Statement(clause string, args ...any) query.Expression {
	return statement{
		clause: clause,
		args:   args,
	}
}

// A raw statement with arguments
type statement struct {
	clause string // The clause with ? used for placeholders
	args   []any  // The replacements for the placeholders in order
}

func (s statement) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	// replace the args with positional args appropriately
	total := s.convertQuestionMarks(w, d, start)

	if len(s.args) != total {
		return s.args, fmt.Errorf("Bad Statement: has %d placeholders but %d args: %s",
			total, len(s.args), s.clause)
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
