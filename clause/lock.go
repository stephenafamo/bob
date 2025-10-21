package clause

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/stephenafamo/bob"
)

var ErrNoLockStrength = errors.New("no lock strength specified")

const (
	LockStrengthUpdate      = "UPDATE"
	LockStrengthNoKeyUpdate = "NO KEY UPDATE"
	LockStrengthShare       = "SHARE"
	LockStrengthKeyShare    = "KEY SHARE"
)

const (
	LockWaitNoWait     = "NOWAIT"
	LockWaitSkipLocked = "SKIP LOCKED"
)

type Locks struct {
	Locks []bob.Expression
}

func (f *Locks) AppendLock(lock bob.Expression) {
	f.Locks = append(f.Locks, lock)
}

type Lock struct {
	Strength string
	Tables   []string
	Wait     string
}

func (f Lock) WriteSQL(ctx context.Context, w io.StringWriter, d bob.Dialect, start int) ([]any, error) {
	if f.Strength == "" {
		return nil, nil
	}

	w.WriteString("FOR ")
	if f.Strength != "" {
		w.WriteString(fmt.Sprintf("%s ", f.Strength))
	}

	args, err := bob.ExpressSlice(ctx, w, d, start, f.Tables, "OF ", ", ", "")
	if err != nil {
		return nil, err
	}

	if f.Wait != "" {
		w.WriteString(" ")
		w.WriteString(f.Wait)
	}

	return args, nil
}
