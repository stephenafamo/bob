package expr

import (
	"errors"
	"io"
	"strconv"

	"github.com/stephenafamo/typesql/query"
)

var (
	ErrNoFrameMode  = errors.New("No frame mode specified")
	ErrNoFrameStart = errors.New("No frame start specified")
)

const (
	FrameModeRange  = "RANGE"
	FrameModeRows   = "ROWS"
	FrameModeGroups = "GROUPS"
)

const (
	FrameUnboundedPreceding = "UNBOUNDED PRECEDING"
	FrameCurrentRow         = "CURRENT ROW"
	FrameUnboundedFollowing = "UNBOUNDED FOLLOWING"
)

func FrameOffsetPreceding(i int) string {
	return strconv.Itoa(i) + " PRECEDING"
}

func FrameOffsetFollowing(i int) string {
	return strconv.Itoa(i) + " FOLLOWING"
}

const (
	FrameExcludeNoOthers   = "NO OTHERS"
	FrameExcludeCurrentRow = "CURRENT ROW"
	FrameExcludeGroup      = "GROUP"
	FrameExcludeTies       = "TIES"
)

func Frame(mode, start, end, exclusion string) frameClause {
	return frameClause{
		Mode:      mode,
		Start:     start,
		End:       end,
		Exclusion: exclusion,
	}
}

type frameClause struct {
	Mode      string
	Start     string
	End       string // can be empty
	Exclusion string // can be empty
}

func (f frameClause) WriteSQL(w io.Writer, d query.Dialect, start int) ([]any, error) {
	if f.Mode == "" {
		return nil, ErrNoFrameMode
	}

	if f.Start == "" {
		return nil, ErrNoFrameStart
	}

	w.Write([]byte(f.Mode))
	w.Write([]byte(" "))

	if f.End != "" {
		w.Write([]byte("BETWEEN "))
	}

	w.Write([]byte(f.Start))

	if f.End != "" {
		w.Write([]byte(" AND "))
		w.Write([]byte(f.End))
	}

	if f.Exclusion != "" {
		w.Write([]byte(" "))
		w.Write([]byte(f.Exclusion))
	}

	return nil, nil
}
