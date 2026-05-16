package im

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

// ConflictTargetExpr is a chainable helper expression for a single ON CONFLICT
// target item, e.g. email COLLATE "en_US" text_pattern_ops.
//
// Values built by this helper are passed directly to OnConflict(...).
type ConflictTargetExpr struct {
	expression any
	collation  any
	opClass    any
}

// ConflictTarget creates a helper for a single ON CONFLICT target item.
//
// The target is rendered as provided. This can be a column-like item ("email")
// or an expression (psql.Raw("lower(email)")). If extra grouping is needed,
// pass a grouped expression explicitly.
func ConflictTarget(target any) ConflictTargetExpr {
	return ConflictTargetExpr{expression: target}
}

// Collate appends COLLATE <name> for this target item.
//
// The collation can be provided as a string token or any bob.Expression.
func (c ConflictTargetExpr) Collate(collation any) ConflictTargetExpr {
	c.collation = collation
	return c
}

// OpClass appends an operator class token for this target item.
//
// The operator class can be provided as a string token or any bob.Expression.
func (c ConflictTargetExpr) OpClass(opClass any) ConflictTargetExpr {
	c.opClass = opClass
	return c
}

func (c ConflictTargetExpr) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	args, err := bob.Express(ctx, w, d, start, c.expression)
	if err != nil {
		return nil, err
	}

	if c.collation != nil {
		if collation, ok := c.collation.(string); !ok || collation != "" {
			w.WriteString(" COLLATE ")

			collationArgs, err := bob.Express(ctx, w, d, start+len(args), c.collation)
			if err != nil {
				return args, err
			}
			args = append(args, collationArgs...)
		}
	}

	if c.opClass != nil {
		if opClass, ok := c.opClass.(string); !ok || opClass != "" {
			w.WriteString(" ")

			opClassArgs, err := bob.Express(ctx, w, d, start+len(args), c.opClass)
			if err != nil {
				return args, err
			}
			args = append(args, opClassArgs...)
		}
	}

	return args, nil
}
