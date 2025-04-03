package pgtypes

import (
	"fmt"
	"testing"
)

func TestInetScan(t *testing.T) {
	for _, text := range []any{
		"192.168.100.128/25",
		[]byte("192.168.100.128/25"),
		"128.0.0.0/16",
		"172.24.0.1",
		"172.24.0.1/32",
		[]byte("172.24.0.1"),
		[]byte("172.24.0.1/32"),
	} {
		var i Inet
		if err := i.Scan(text); err != nil {
			t.Errorf("%#v: %v", text, err)
		}
		fmt.Println(i)
		if val, err := i.Value(); err != nil {
			t.Errorf("%#v: %v", val, err)
		}
	}
}
