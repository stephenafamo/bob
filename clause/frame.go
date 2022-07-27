package clause

import (
	"errors"
	"io"

	"github.com/stephenafamo/bob/query"
)

var (
	ErrNoFrameMode  = errors.New("No frame mode specified")
	ErrNoFrameStart = errors.New("No frame start specified")
)

type Frame struct {
	Mode      string
	Start     any
	End       any    // can be nil
	Exclusion string // can be empty
}

func (f *Frame) SetMode(mode string) {
	f.Mode = mode
}

func (f *Frame) SetStart(start any) {
	f.Start = start
}

func (f *Frame) SetEnd(end any) {
	f.End = end
}

func (f *Frame) SetExclusion(excl string) {
	f.Exclusion = excl
}

func (f Frame) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if f.Mode == "" {
		return nil, ErrNoFrameMode
	}

	if f.Start == nil {
		return nil, ErrNoFrameStart
	}

	var args []any

	w.Write([]byte(f.Mode))
	w.Write([]byte(" "))

	if f.End != nil {
		w.Write([]byte("BETWEEN "))
	}

	startArgs, err := query.Express(w, d, start, f.Start)
	if err != nil {
		return nil, err
	}
	args = append(args, startArgs...)

	endArgs, err := query.ExpressIf(w, d, start, f.End, f.End != "", " AND ", "")
	if err != nil {
		return nil, err
	}
	args = append(args, endArgs...)

	query.ExpressIf(w, d, start, f.Exclusion, f.Exclusion != "", " EXCLUDE ", "")

	return nil, nil
}
