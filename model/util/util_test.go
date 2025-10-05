package util

import (
	"testing"
)

func TestToCamelCase(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"hello world", "hello_world"},
		{"Hello World", "Hello_World"},
		{"HELLO WORLD", "HELLO_WORLD"},
		{"helloWORLD", "hello_WORLD"},
		{"helloWorld", "hello_World"},
		{"hello_World", "hello_World"},
		{"hello_world_", "hello_world_"},
		{"hello__world", "hello__world"},
		{"", ""},
	}

	for _, tc := range testCases {
		result := ToCamelCase(tc.input)
		if result != tc.expected {
			t.Errorf("ToCamelCase(%q) = %q; want %q", tc.input, result, tc.expected)
		}
	}
}