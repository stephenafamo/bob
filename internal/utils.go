package internal

import (
	"hash/maphash"
)

func ToAnySlice[T any, Ts ~[]T](slice Ts) []any {
	ret := make([]any, len(slice))
	for i, val := range slice {
		ret[i] = val
	}

	return ret
}

func FirstNonEmpty[T comparable, Ts ~[]T](slice Ts) T {
	var zero T
	for _, val := range slice {
		if val != zero {
			return val
		}
	}

	return zero
}

func FilterNonZero[T comparable](s []T) []T {
	var zero T
	filtered := make([]T, 0, len(s))

	for _, v := range s {
		if v == zero {
			continue
		}
		filtered = append(filtered, v)
	}

	return filtered
}

func RandInt() int64 {
	out := int64(new(maphash.Hash).Sum64())

	if out < 0 {
		return -out % 10000
	}

	return out % 10000
}

func SliceMatch[T comparable, Ts ~[]T](a, b Ts) bool {
	if len(a) != len(b) {
		return false
	}

	if len(a) == 0 {
		return false
	}

	var matches int
	for _, v1 := range a {
		for _, v2 := range b {
			if v1 == v2 {
				matches++
			}
		}
	}

	return matches == len(a)
}

// Check if any one of the lists contains all the columns
func AllColsInList(cols []string, lists ...[]string) bool {
ColumnsLoop:
	for _, col := range cols {
		for _, list := range lists {
			for _, sideCol := range list {
				if col == sideCol {
					continue ColumnsLoop
				}
			}
		}
		return false
	}

	return true
}

func RemoveDuplicates[T comparable, Ts ~[]T](slice Ts) Ts {
	seen := make(map[T]struct{}, len(slice))
	final := make(Ts, 0, len(slice))

	for _, v := range slice {
		if _, ok := seen[v]; ok {
			continue
		}

		seen[v] = struct{}{}
		final = append(final, v)
	}

	return final
}

func InList[T comparable](s []T, val T) bool {
	for _, v := range s {
		if v == val {
			return true
		}
	}

	return false
}
