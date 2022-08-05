package scanto

import (
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestValueRecording(t *testing.T) {
	Columns := map[string]any{
		"bool":       false,
		"*bool":      ptr(false),
		"string":     "",
		"*string":    ptr(""),
		"[]byte":     []byte(""),
		"*[]byte":    ptr([]byte("")),
		"int":        0,
		"*int":       ptr(0),
		"int64":      int64(0),
		"*int64":     ptr(int64(0)),
		"float32":    float32(0),
		"*float32":   ptr(float32(0)),
		"float64":    float64(0),
		"*float64":   ptr(float64(0)),
		"time.Time":  time.Time{},
		"*time.Time": ptr(time.Time{}),
	}

	vals := &Values{
		types:     make(map[string]reflect.Type, len(Columns)),
		recording: true,
	}

	// record the values
	for k, v := range Columns {
		GetType(vals, k, reflect.TypeOf(v))
	}

	// turn off recording
	vals.recording = false

	// get the values and check the type
	for k, v := range Columns {
		vTyp := reflect.TypeOf(v)
		gotten := GetType(vals, k, vTyp)
		// We expect the zero value because there was no
		// scanning
		expected := reflect.Zero(vTyp).Interface()

		if diff := cmp.Diff(expected, gotten); diff != "" {
			t.Fatalf("diff: %s", diff)
		}
	}
}
