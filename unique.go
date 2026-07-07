package bob

import "sync/atomic"

//nolint:gochecknoglobals
var uniqueCounter atomic.Uint64

// NextUniqueInt returns a unique positive integer suitable for SQL alias suffixes.
// Values start at 10001 and increment. On uint64 overflow the counter restarts at 1.
func NextUniqueInt() uint64 {
	n := uniqueCounter.Add(1)
	if n == 0 {
		n = 1
	}

	return n + 10000
}
