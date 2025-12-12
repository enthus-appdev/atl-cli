package output

import (
	"bytes"
	"strings"
	"testing"
)

// TestJSON tests the JSON function outputs properly formatted JSON.
func TestJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		contains []string // Strings that should be in the output
	}{
		{
			name: "simple struct",
			data: struct {
				Name string `json:"name"`
				Age  int    `json:"age"`
			}{Name: "Alice", Age: 30},
			contains: []string{`"name": "Alice"`, `"age": 30`},
		},
		{
			name: "map",
			data: map[string]string{"key": "value"},
			contains: []string{`"key": "value"`},
		},
		{
			name: "slice",
			data: []int{1, 2, 3},
			contains: []string{"1", "2", "3"},
		},
		{
			name: "nested struct",
			data: struct {
				User struct {
					Name string `json:"name"`
				} `json:"user"`
			}{User: struct {
				Name string `json:"name"`
			}{Name: "Bob"}},
			contains: []string{`"user":`, `"name": "Bob"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			err := JSON(buf, tt.data)
			if err != nil {
				t.Errorf("JSON() error = %v", err)
				return
			}

			output := buf.String()
			for _, want := range tt.contains {
				if !strings.Contains(output, want) {
					t.Errorf("JSON() output missing %q\nGot: %s", want, output)
				}
			}
		})
	}
}

// TestJSONIndentation tests that JSON output is properly indented.
func TestJSONIndentation(t *testing.T) {
	data := map[string]interface{}{
		"outer": map[string]string{
			"inner": "value",
		},
	}

	buf := &bytes.Buffer{}
	err := JSON(buf, data)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	output := buf.String()
	// Check for 2-space indentation
	if !strings.Contains(output, "  ") {
		t.Error("JSON() output should have 2-space indentation")
	}
}

// TestJSONCompact tests the JSONCompact function outputs compact JSON.
func TestJSONCompact(t *testing.T) {
	data := struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}{Name: "Alice", Age: 30}

	buf := &bytes.Buffer{}
	err := JSONCompact(buf, data)
	if err != nil {
		t.Fatalf("JSONCompact() error = %v", err)
	}

	output := buf.String()
	// Compact JSON should not have pretty-printing indentation
	// It will still have a newline at the end from Encode
	if strings.Contains(output, "  ") {
		t.Error("JSONCompact() output should not have indentation")
	}
	if !strings.Contains(output, `"name":"Alice"`) {
		t.Errorf("JSONCompact() output = %q, missing expected content", output)
	}
}

// TestJSONString tests the JSONString function.
func TestJSONString(t *testing.T) {
	data := struct {
		Message string `json:"message"`
	}{Message: "Hello"}

	result, err := JSONString(data)
	if err != nil {
		t.Fatalf("JSONString() error = %v", err)
	}

	if !strings.Contains(result, `"message": "Hello"`) {
		t.Errorf("JSONString() = %q, missing expected content", result)
	}
}

// TestJSONStringError tests JSONString with an unencodable value.
func TestJSONStringError(t *testing.T) {
	// Channels cannot be JSON encoded
	ch := make(chan int)

	_, err := JSONString(ch)
	if err == nil {
		t.Error("JSONString() with channel should return error")
	}
}

// TestJSONNilData tests JSON with nil data.
func TestJSONNilData(t *testing.T) {
	buf := &bytes.Buffer{}
	err := JSON(buf, nil)
	if err != nil {
		t.Errorf("JSON(nil) error = %v", err)
	}
	if buf.String() != "null\n" {
		t.Errorf("JSON(nil) = %q, want %q", buf.String(), "null\n")
	}
}

// TestJSONEmptyStruct tests JSON with empty struct.
func TestJSONEmptyStruct(t *testing.T) {
	buf := &bytes.Buffer{}
	err := JSON(buf, struct{}{})
	if err != nil {
		t.Errorf("JSON(struct{}{}) error = %v", err)
	}
	if buf.String() != "{}\n" {
		t.Errorf("JSON(struct{}{}) = %q, want %q", buf.String(), "{}\n")
	}
}

// TestJSONSpecialCharacters tests JSON escaping of special characters.
func TestJSONSpecialCharacters(t *testing.T) {
	data := map[string]string{
		"message": "Hello \"World\"\nNew line",
	}

	buf := &bytes.Buffer{}
	err := JSON(buf, data)
	if err != nil {
		t.Fatalf("JSON() error = %v", err)
	}

	output := buf.String()
	// Check that quotes and newlines are properly escaped
	if !strings.Contains(output, `\"World\"`) {
		t.Errorf("JSON() should escape quotes, got: %s", output)
	}
	if !strings.Contains(output, `\n`) {
		t.Errorf("JSON() should escape newlines, got: %s", output)
	}
}
