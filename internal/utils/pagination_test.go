package utils

import (
	"encoding/json"
	"testing"
)

func TestEnsureNonNilSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name:     "nil interface",
			input:    nil,
			expected: "[]",
		},
		{
			name:     "nil slice of struct",
			input:    ([]struct{ Name string })(nil),
			expected: "[]",
		},
		{
			name:     "nil slice of string",
			input:    ([]string)(nil),
			expected: "[]",
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: "[]",
		},
		{
			name:     "non-empty slice",
			input:    []string{"a", "b"},
			expected: `["a","b"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := ensureNonNilSlice(tt.input)
			b, err := json.Marshal(normalized)
			if err != nil {
				t.Fatalf("json.Marshal failed: %v", err)
			}
			if string(b) != tt.expected {
				t.Errorf("got %s, want %s", string(b), tt.expected)
			}
		})
	}
}

