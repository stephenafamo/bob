package clause

import (
	"context"
	"io"

	"github.com/stephenafamo/bob"
)

type Frame struct {
	Defined   bool // whether any of the parts was defined
	Mode      string
	Start     any
	End       any    // can be nil
	Exclusion string // can be empty
}

func (f *Frame) SetMode(mode string) {
	f.Defined = true
	f.Mode = mode
}

func (f *Frame) SetStart(start any) {
	f.Defined = true
	f.Start = start
}

func (f *Frame) SetEnd(end any) {
	f.Defined = true
	f.End = end
}

func (f *Frame) SetExclusion(excl string) {
	f.Defined = true
	f.Exclusion = excl
}

func (f Frame) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if f.Mode == "" {
		f.Mode = "RANGE"
	}

	if f.Start == nil {
		f.Start = "UNBOUNDED PRECEDING"
	}

	var args []any

	w.WriteString(f.Mode)
	w.WriteString(" ")

	if f.End != nil {
		w.WriteString("BETWEEN ")
	}

	startArgs, err := bob.Express(ctx, w, d, start, f.Start)
	if err != nil {
		return nil, err
	}
	args = append(args, startArgs...)

	endArgs, err := bob.ExpressIf(ctx, w, d, start, f.End, f.End != nil, " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, endArgs...)

	_, err = bob.ExpressIf(ctx, w, d, start, f.Exclusion, f.Exclusion != "", " EXCLUDE ", "")
	if err != nil {
		return nil, err
	}

	return args, nil
}
