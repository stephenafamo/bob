package scanto

import (
	"math/rand"
	"reflect"
	"time"
)

func ptr[T any](v T) any {
	val := reflect.ValueOf(v)
	p := reflect.New(val.Type())
	p.Elem().Set(reflect.ValueOf(v))

	return p.Interface()
}

func colSliceFromMap(c [][2]string) []string {
	s := make([]string, 0, len(c))
	for _, def := range c {
		s = append(s, def[0])
	}
	return s
}

func singleRows[T any](vals ...T) rows {
	r := make(rows, len(vals))
	for k, v := range vals {
		r[k] = []any{v}
	}

	return r
}

func randate() time.Time {
	min := time.Date(1970, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	max := time.Date(2070, 1, 0, 0, 0, 0, 0, time.UTC).Unix()
	delta := max - min

	sec := rand.Int63n(delta) + min
	return time.Unix(sec, 0)
}
