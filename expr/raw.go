package expr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/stephenafamo/bob"
)

type Raw string

func (r Raw) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return nil, r.WriteSQLTo(ctx, w, d, start, nil)
}

func (r Raw) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	w.WriteString(string(r))
	return nil
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

func (r Clause) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	var args []any
	total, err := r.writeSQLTo(ctx, w, d, start, &args)
	if err != nil {
		return nil, err
	}

	if len(r.args) != total {
		return nil, &rawError{args: len(r.args), placeholders: total, clause: r.query}
	}

	return args, nil
}

func (r Clause) WriteSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) error {
	origLen := len(*args)
	origNil := *args == nil
	total, err := r.writeSQLTo(ctx, w, d, start, args)
	if err != nil {
		if origNil && origLen == 0 {
			*args = nil
		} else {
			*args = (*args)[:origLen]
		}
		return err
	}

	if len(r.args) != total {
		if origNil && origLen == 0 {
			*args = nil
		} else {
			*args = (*args)[:origLen]
		}
		return &rawError{args: len(r.args), placeholders: total, clause: r.query}
	}

	return nil
}

func (r Clause) writeSQLTo(ctx context.Context, w io.StringWriter, d bob.Dialect, start int, args *[]any) (int, error) {
	// replace the args with positional args appropriately
	if start == 0 {
		panic("Not a valid start number.")
	}

	baseLen := len(*args)
	paramIndex := 0
	total := 0

	clause := r.query
	for paramIndex < len(clause) {
		clause = clause[paramIndex:]
		paramIndex = strings.IndexByte(clause, '?')

		if paramIndex == -1 {
			w.WriteString(clause)
			break
		}

		escapeIndex := strings.Index(clause, `\?`)
		if escapeIndex != -1 && paramIndex > escapeIndex {
			w.WriteString(clause[:escapeIndex] + "?")
			paramIndex++
			continue
		}

		w.WriteString(clause[:paramIndex])

		var arg any
		if total < len(r.args) {
			arg = r.args[total]
		}

		if ex, ok := arg.(bob.Expression); ok {
			if err := bob.ExpressTo(ctx, w, d, start+len(*args)-baseLen, ex, args); err != nil {
				return total, err
			}
		} else {
			d.WriteArg(w, start+len(*args)-baseLen)
			*args = append(*args, arg)
		}

		total++
		paramIndex++
	}

	return total, nil
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
