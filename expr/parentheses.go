package expr

import (
	"io"

	"github.com/stephenafamo/bob/query"
)

// Add parentheses around an expression
func P(exp any) query.Expression {
	return parentheses{inside: exp}
}

// Multiple expressions that will be group together as a single expression
type parentheses struct {
	inside any
}

func (p parentheses) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	return query.ExpressIf(w, d, start, p.inside, p.inside != nil, "(", ")")
}
