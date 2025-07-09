package bob

import (
	"context"
	"database/sql/driver"
	"fmt"
	"io"
)

type RawNamedArgError struct {
	Name string
}

func (e RawNamedArgError) Error() string {
	return fmt.Sprintf("raw named arg %q used without rebinding", e.Name)
}

// named args should ONLY be used to prepare statements
type namedArg string

// Value implements the driver.Valuer interface.
// it always returns an error because named args should only be used to prepare statements
func (n namedArg) Value() (driver.Value, error) {
	return nil, RawNamedArgError{string(n)}
}

// Named args should ONLY be used to prepare statements
func Named(names ...string) Expression {
	return named{names: names}
}

// NamedGroup is like Named, but wraps in parentheses
func NamedGroup(names ...string) Expression {
	return named{names: names, grouped: true}
}

type named struct {
	names   []string
	grouped bool
}

func (a named) WriteSQL(ctx context.Context, w io.Writer, d Dialect, start int) ([]any, error) {
	if len(a.names) == 0 {
		return nil, nil
	}

	args := make([]any, len(a.names))

	if a.grouped {
		w.Write([]byte(openPar))
	}

	for k, name := range a.names {
		if k > 0 {
			w.Write([]byte(commaSpace))
		}

		d.WriteArg(w, start+k)
		args[k] = namedArg(name)
	}

	if a.grouped {
		w.Write([]byte(closePar))
	}

	return args, nil
}
