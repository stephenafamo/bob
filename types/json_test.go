package types

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

type testStruct struct {
	Name  string `json:"name"`
	Value int    `json:"value"`
}

type customTextMarshaler struct {
	Value string
}

func (c customTextMarshaler) MarshalText() ([]byte, error) {
	return []byte("custom:" + c.Value), nil
}

func (c *customTextMarshaler) UnmarshalText(text []byte) error {
	expected := "custom:"
	if len(text) < len(expected) {
		return fmt.Errorf("invalid custom text format")
	}
	c.Value = string(text[len(expected):])
	return nil
}

func TestJSON_Value(t *testing.T) {
	t.Run("simple struct", func(t *testing.T) {
		j := NewJSON(testStruct{Name: "test", Value: 42})
		got, err := j.Value()
		if err != nil {
			t.Errorf("Value() error = %v", err)
			return
		}
		// Important: Value() should return a string, not []byte (pgx compatibility)
		if _, ok := got.(string); !ok {
			t.Errorf("Value() should return string type, got %T", got)
		}
		want := `{"name":"test","value":42}`
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Value() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("string value", func(t *testing.T) {
		j := NewJSON("hello")
		got, err := j.Value()
		if err != nil {
			t.Errorf("Value() error = %v", err)
			return
		}
		if _, ok := got.(string); !ok {
			t.Errorf("Value() should return string type, got %T", got)
		}
		want := `"hello"`
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Value() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("null value", func(t *testing.T) {
		j := NewJSON[any](nil)
		got, err := j.Value()
		if err != nil {
			t.Errorf("Value() error = %v", err)
			return
		}
		if _, ok := got.(string); !ok {
			t.Errorf("Value() should return string type, got %T", got)
		}
		want := "null"
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Value() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("raw message", func(t *testing.T) {
		j := NewJSON(json.RawMessage(`{"foo":"bar"}`))
		got, err := j.Value()
		if err != nil {
			t.Errorf("Value() error = %v", err)
			return
		}
		if _, ok := got.(string); !ok {
			t.Errorf("Value() should return string type, got %T", got)
		}
		want := `{"foo":"bar"}`
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Value() mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("array", func(t *testing.T) {
		j := NewJSON([]int{1, 2, 3})
		got, err := j.Value()
		if err != nil {
			t.Errorf("Value() error = %v", err)
			return
		}
		if _, ok := got.(string); !ok {
			t.Errorf("Value() should return string type, got %T", got)
		}
		want := "[1,2,3]"
		if diff := cmp.Diff(want, got); diff != "" {
			t.Errorf("Value() mismatch (-want +got):\n%s", diff)
		}
	})
}

func TestJSON_Scan(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    testStruct
		wantErr bool
	}{
		{
			name:    "scan from string",
			value:   `{"name":"foo","value":123}`,
			want:    testStruct{Name: "foo", Value: 123},
			wantErr: false,
		},
		{
			name:    "scan from bytes",
			value:   []byte(`{"name":"bar","value":456}`),
			want:    testStruct{Name: "bar", Value: 456},
			wantErr: false,
		},
		{
			name:    "scan nil",
			value:   nil,
			want:    testStruct{},
			wantErr: false,
		},
		{
			name:    "scan invalid type",
			value:   123,
			want:    testStruct{},
			wantErr: true,
		},
		{
			name:    "scan invalid json string",
			value:   `{"invalid json`,
			want:    testStruct{},
			wantErr: true,
		},
		{
			name:    "scan invalid json bytes",
			value:   []byte(`{"invalid json`),
			want:    testStruct{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j JSON[testStruct]
			err := j.Scan(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, j.Val); diff != "" {
					t.Errorf("Scan() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestJSON_MarshalvJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    JSON[testStruct]
		want    string
		wantErr bool
	}{
		{
			name:    "marshal struct",
			json:    NewJSON(testStruct{Name: "test", Value: 99}),
			want:    `{"name":"test","value":99}`,
			wantErr: false,
		},
		{
			name:    "marshal empty struct",
			json:    NewJSON(testStruct{}),
			want:    `{"name":"","value":0}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.json.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, string(got)); diff != "" {
				t.Errorf("MarshalJSON() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestJSON_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		want    testStruct
		wantErr bool
	}{
		{
			name:    "unmarshal valid json",
			data:    []byte(`{"name":"alice","value":777}`),
			want:    testStruct{Name: "alice", Value: 777},
			wantErr: false,
		},
		{
			name:    "unmarshal invalid json",
			data:    []byte(`{"invalid`),
			want:    testStruct{},
			wantErr: true,
		},
		{
			name:    "unmarshal empty json",
			data:    []byte(`{}`),
			want:    testStruct{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var j JSON[testStruct]
			err := j.UnmarshalJSON(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if diff := cmp.Diff(tt.want, j.Val); diff != "" {
					t.Errorf("UnmarshalJSON() mismatch (-want +got):\n%s", diff)
				}
			}
		})
	}
}

func TestJSON_MarshalText(t *testing.T) {
	tests := []struct {
		name    string
		json    any
		want    string
		wantErr bool
	}{
		{
			name:    "string value",
			json:    NewJSON("hello world"),
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "integer value",
			json:    NewJSON(42),
			want:    "42",
			wantErr: false,
		},
		{
			name:    "custom text marshaler",
			json:    NewJSON(customTextMarshaler{Value: "test"}),
			want:    "custom:test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got []byte
			var err error

			switch v := tt.json.(type) {
			case JSON[string]:
				got, err = v.MarshalText()
			case JSON[int]:
				got, err = v.MarshalText()
			case JSON[customTextMarshaler]:
				got, err = v.MarshalText()
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(tt.want, string(got)); diff != "" {
				t.Errorf("MarshalText() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}

func TestJSON_UnmarshalText(t *testing.T) {
	tests := []struct {
		name    string
		text    []byte
		target  any
		want    any
		wantErr bool
	}{
		{
			name:    "string value",
			text:    []byte("hello world"),
			target:  &JSON[string]{},
			want:    "hello world",
			wantErr: false,
		},
		{
			name:    "custom text unmarshaler",
			text:    []byte("custom:foo"),
			target:  &JSON[customTextMarshaler]{},
			want:    customTextMarshaler{Value: "foo"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			switch target := tt.target.(type) {
			case *JSON[string]:
				err := target.UnmarshalText(tt.text)
				if (err != nil) != tt.wantErr {
					t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr {
					if diff := cmp.Diff(tt.want, target.Val); diff != "" {
						t.Errorf("UnmarshalText() mismatch (-want +got):\n%s", diff)
					}
				}
			case *JSON[customTextMarshaler]:
				err := target.UnmarshalText(tt.text)
				if (err != nil) != tt.wantErr {
					t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if !tt.wantErr {
					if diff := cmp.Diff(tt.want, target.Val); diff != "" {
						t.Errorf("UnmarshalText() mismatch (-want +got):\n%s", diff)
					}
				}
			}
		})
	}
}

func TestJSON_PostgreSQLCompatibility(t *testing.T) {
	// This test verifies that Value() returns a string for pgx compatibility
	// to prevent the "invalid input syntax for type json" error
	jsonData := NewJSON(map[string]any{
		"id":   1,
		"name": "test",
		"nested": map[string]any{
			"value": 123,
		},
	})

	val, err := jsonData.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	// Critical: Must return string, not []byte for pgx compatibility
	strVal, ok := val.(string)
	if !ok {
		t.Fatalf("Value() must return string for pgx compatibility, got %T", val)
	}

	// Verify it's valid JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(strVal), &result); err != nil {
		t.Errorf("Value() returned invalid JSON: %v", err)
	}

	// Verify the JSON contains our data directly (not wrapped)
	if result["id"] != float64(1) || result["name"] != "test" {
		t.Error("Value() should return the inner value directly, not wrapped")
	}

	nested, ok := result["nested"].(map[string]any)
	if !ok || nested["value"] != float64(123) {
		t.Error("Nested structure not preserved correctly")
	}
}

func TestJSON_RawMessage(t *testing.T) {
	// Test with json.RawMessage specifically as that's what Bob generates
	rawJSON := json.RawMessage(`{"key":"value","number":42}`)
	j := NewJSON(rawJSON)

	// Test Value()
	val, err := j.Value()
	if err != nil {
		t.Fatalf("Value() error = %v", err)
	}

	// Must be a string for pgx
	if _, ok := val.(string); !ok {
		t.Errorf("Value() should return string, got %T", val)
	}

	// Test Scan with string
	var j2 JSON[json.RawMessage]
	if err := j2.Scan(`{"key":"value","number":42}`); err != nil {
		t.Errorf("Scan(string) error = %v", err)
	}

	// Test Scan with bytes
	var j3 JSON[json.RawMessage]
	if err := j3.Scan([]byte(`{"key":"value","number":42}`)); err != nil {
		t.Errorf("Scan([]byte) error = %v", err)
	}

	// Verify the scanned values are valid JSON
	var parsed map[string]any
	if err := json.Unmarshal(j2.Val, &parsed); err != nil {
		t.Errorf("Scanned value is not valid JSON: %v", err)
	}
	if parsed["key"] != "value" || parsed["number"] != float64(42) {
		t.Errorf("Unexpected scanned value: %v", parsed)
	}
}
