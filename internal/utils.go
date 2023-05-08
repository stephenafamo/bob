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
