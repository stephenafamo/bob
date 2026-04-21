package dialect

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"strconv"

	"github.com/stephenafamo/bob"
	"github.com/stephenafamo/bob/clause"
)

func isSimpleTableRef(ref clause.TableRef) bool {
	return ref.Expression != nil &&
		ref.Alias == "" &&
		len(ref.Columns) == 0 &&
		!ref.Only &&
		!ref.Lateral &&
		!ref.WithOrdinality &&
		ref.IndexedBy == nil &&
		len(ref.Partitions) == 0 &&
		len(ref.IndexHints) == 0 &&
		len(ref.Joins) == 0
}

type queryWriter struct {
	ctx   context.Context
	w     io.StringWriter
	args  []any
	start int
}

func (w *queryWriter) argPos() int {
	return w.start + len(w.args)
}

func (w *queryWriter) appendArgs(args []any) {
	w.args = append(w.args, args...)
}

func (w *queryWriter) writeExpression(value bob.Expression) error {
	args, err := value.WriteSQL(w.ctx, w.w, Dialect, w.argPos())
	if err != nil {
		return err
	}
	w.appendArgs(args)
	return nil
}

func (w *queryWriter) writeAny(value any) error {
	switch v := value.(type) {
	case nil:
		_, _ = w.w.WriteString("NULL")
	case string:
		_, _ = w.w.WriteString(v)
	case []byte:
		_, _ = w.w.WriteString(string(v))
	case int:
		_, _ = w.w.WriteString(strconv.Itoa(v))
	case int8:
		_, _ = w.w.WriteString(strconv.FormatInt(int64(v), 10))
	case int16:
		_, _ = w.w.WriteString(strconv.FormatInt(int64(v), 10))
	case int32:
		_, _ = w.w.WriteString(strconv.FormatInt(int64(v), 10))
	case int64:
		_, _ = w.w.WriteString(strconv.FormatInt(v, 10))
	case uint:
		_, _ = w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint8:
		_, _ = w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint16:
		_, _ = w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint32:
		_, _ = w.w.WriteString(strconv.FormatUint(uint64(v), 10))
	case uint64:
		_, _ = w.w.WriteString(strconv.FormatUint(v, 10))
	case sql.NamedArg:
		return fmt.Errorf("named args are not supported by psql dialect")
	case bob.Expression:
		return w.writeExpression(v)
	default:
		_, _ = w.w.WriteString(fmt.Sprint(v))
	}

	return nil
}

func (w *queryWriter) writeSliceAny(values []any, sep string) error {
	for i, value := range values {
		if i > 0 {
			_, _ = w.w.WriteString(sep)
		}
		if err := w.writeAny(value); err != nil {
			return err
		}
	}
	return nil
}

func (w *queryWriter) writeSliceExpr(values []bob.Expression, sep string) error {
	for i, value := range values {
		if i > 0 {
			_, _ = w.w.WriteString(sep)
		}
		if err := w.writeExpression(value); err != nil {
			return err
		}
	}
	return nil
}

func (w *queryWriter) writeOrderExprs(values []bob.Expression) error {
	for i, value := range values {
		if i > 0 {
			_, _ = w.w.WriteString(", ")
		}

		switch order := value.(type) {
		case clause.OrderDef:
			if err := w.writeAny(order.Expression); err != nil {
				return err
			}
			if order.Collation != "" {
				_, _ = w.w.WriteString(" COLLATE ")
				Dialect.WriteQuoted(w.w, order.Collation)
			}
			if order.Direction != "" {
				_, _ = w.w.WriteString(" ")
				_, _ = w.w.WriteString(order.Direction)
			}
			if order.Nulls != "" {
				_, _ = w.w.WriteString(" NULLS ")
				_, _ = w.w.WriteString(order.Nulls)
			}
		default:
			if err := w.writeExpression(value); err != nil {
				return err
			}
		}
	}
	return nil
}
