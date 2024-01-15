package expr

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob"
)

type Raw []byte

func (r Raw) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write(r)
	return nil, nil
}

func RawQuery(d bob.Dialect, q string, args ...any) bob.BaseQuery[Clause] {
	return bob.BaseQuery[Clause]{
		Expression: Clause{query: q, args: args},
		Dialect:    d,
	}
}

// A Raw Clause with arguments
type Clause struct {
	query string // The clause with ? used for placeholders
	args  []any  // The replacements for the placeholders in order
}

func (r Clause) WriteSQL(w io.Writer, d bob.Dialect, start int) ([]any, error) {
	// replace the args with positional args appropriately
	total, args, err := r.convertQuestionMarks(w, d, start)
	if err != nil {
		return nil, err
	}

	if len(r.args) != total {
		return r.args, &rawError{args: len(r.args), placeholders: total, clause: r.query}
	}

	return args, nil
}

// convertQuestionMarks converts each occurrence of ? with $<number>
// where <number> is an incrementing digit starting at startAt.
// If question-mark (?) is escaped using back-slash (\), it will be ignored.
func (r Clause) convertQuestionMarks(w io.Writer, d bob.Dialect, startAt int) (int, []any, error) {
	if startAt == 0 {
		panic("Not a valid start number.")
	}

	paramIndex := 0
	total := 0
	var args []any

	clause := r.query
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

		var arg any
		if total < len(r.args) {
			arg = r.args[total]
		}
		if ex, ok := arg.(bob.Expression); ok {
			eargs, err := ex.WriteSQL(w, d, startAt)
			if err != nil {
				return total, nil, err
			}
			args = append(args, eargs...)
			startAt += len(eargs)
		} else {
			d.WriteArg(w, startAt)
			args = append(args, arg)
			startAt++
		}

		total++
		paramIndex++
	}

	return total, args, nil
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
