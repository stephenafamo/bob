package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type IWindow interface {
	SetFrom(string)
	AddPartitionBy(...any)
	AddOrderBy(...any)
	SetMode(string)
	SetStart(any)
	SetEnd(any)
	SetExclusion(string)
}

type Window struct {
	From        string // an existing window name
	orderBy     []any
	partitionBy []any
	Frame
}

func (wi *Window) SetFrom(from string) {
	wi.From = from
}

func (wi *Window) AddPartitionBy(condition ...any) {
	wi.partitionBy = append(wi.partitionBy, condition...)
}

func (wi *Window) AddOrderBy(order ...any) {
	wi.orderBy = append(wi.orderBy, order...)
}

func (wi Window) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	if wi.From != "" {
		w.Write([]byte(wi.From))
		w.Write([]byte(" "))
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, wi.partitionBy, "PARTITION BY ", ", ", " ")
	if err != nil {
		return nil, err
	}

	orderArgs, err := bob.ExpressSlice(ctx, w, d, start, wi.orderBy, "ORDER BY ", ", ", "")
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
	Definition any
}

func (n NamedWindow) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	w.Write([]byte(n.Name))
	w.Write([]byte(" AS ("))
	args, err := bob.Express(ctx, w, d, start, n.Definition)
	w.Write([]byte(")"))

	return args, err
}

type Windows struct {
	Windows []NamedWindow
}

func (wi *Windows) AppendWindow(w NamedWindow) {
	wi.Windows = append(wi.Windows, w)
}

func (wi Windows) WriteSQL(ctx context.Context, w io.Writer, d bob.Dialect, start int) ([]any, error) {
	return bob.ExpressSlice(ctx, w, d, start, wi.Windows, "WINDOW ", ", ", "")
}
