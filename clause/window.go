package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Window struct {
	BasedOn string // an existing window name
	OrderBy
	partitionBy []any
	Frame
}

func (wi *Window) SetBasedOn(from string) {
	wi.BasedOn = from
}

func (wi *Window) AddPartitionBy(condition ...any) {
	wi.partitionBy = append(wi.partitionBy, condition...)
}

func (wi Window) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if wi.BasedOn != "" {
		w.WriteString(wi.BasedOn)
		w.WriteString(" ")
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, wi.partitionBy, "PARTITION BY ", ", ", " ")
	if err != nil {
		return nil, err
	}

	orderArgs, err := bob.ExpressIf(ctx, w, d, start+len(args), wi.OrderBy,
		len(wi.OrderBy.Expressions) > 0, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, orderArgs...)

	frameArgs, err := bob.ExpressIf(ctx, w, d, start, wi.Frame, wi.Frame.Defined, " ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, frameArgs...)

	return args, nil
}

type NamedWindow struct {
	Name       string
	Definition Window
}

func (n NamedWindow) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	w.WriteString(n.Name)
	w.WriteString(" AS (")
	args, err := bob.Express(ctx, w, d, start, n.Definition)
	w.WriteString(")")

	return args, err
}

type Windows struct {
	Windows []bob.Expression
}

func (wi *Windows) AppendWindow(w bob.Expression) {
	wi.Windows = append(wi.Windows, w)
}

func (wi Windows) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, wi.Windows, "WINDOW ", ", ", "")
}
