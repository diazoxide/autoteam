package util

import (
	"testing"
)

func TestStringPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "normal string",
			input:    "hello world",
			expected: "hello world",
		},
		{
			name:     "string with special characters",
			input:    "hello@world!123",
			expected: "hello@world!123",
		},
		{
			name:     "unicode string",
			input:    "héllo wörld 🌍",
			expected: "héllo wörld 🌍",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringPtr(tt.input)

			if result == nil {
				t.Error("Expected pointer to be non-nil")
				return
			}

			if *result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, *result)
			}

			// Test that we get a real pointer (can modify original through pointer)
			original := tt.input
			ptr := StringPtr(original)
			if &original == ptr {
				t.Error("Expected pointer to different memory location")
			}
		})
	}
}

func TestIntPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int
	}{
		{
			name:     "zero",
			input:    0,
			expected: 0,
		},
		{
			name:     "positive number",
			input:    42,
			expected: 42,
		},
		{
			name:     "negative number",
			input:    -123,
			expected: -123,
		},
		{
			name:     "large number",
			input:    2147483647,
			expected: 2147483647,
		},
		{
			name:     "large negative number",
			input:    -2147483648,
			expected: -2147483648,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IntPtr(tt.input)

			if result == nil {
				t.Error("Expected pointer to be non-nil")
				return
			}

			if *result != tt.expected {
				t.Errorf("Expected %d but got %d", tt.expected, *result)
			}

			// Test that we get a real pointer
			original := tt.input
			ptr := IntPtr(original)
			if &original == ptr {
				t.Error("Expected pointer to different memory location")
			}
		})
	}
}

func TestBoolPtr(t *testing.T) {
	tests := []struct {
		name     string
		input    bool
		expected bool
	}{
		{
			name:     "true value",
			input:    true,
			expected: true,
		},
		{
			name:     "false value",
			input:    false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BoolPtr(tt.input)

			if result == nil {
				t.Error("Expected pointer to be non-nil")
				return
			}

			if *result != tt.expected {
				t.Errorf("Expected %t but got %t", tt.expected, *result)
			}

			// Test that we get a real pointer
			original := tt.input
			ptr := BoolPtr(original)
			if &original == ptr {
				t.Error("Expected pointer to different memory location")
			}
		})
	}
}

// Test that the pointer functions create independent memory
func TestPointerIndependence(t *testing.T) {
	t.Run("string pointers are independent", func(t *testing.T) {
		str1 := "original"
		ptr1 := StringPtr(str1)
		ptr2 := StringPtr(str1)

		if ptr1 == ptr2 {
			t.Error("Expected different pointer addresses")
		}

		if *ptr1 != *ptr2 {
			t.Error("Expected same values")
		}
	})

	t.Run("int pointers are independent", func(t *testing.T) {
		num1 := 42
		ptr1 := IntPtr(num1)
		ptr2 := IntPtr(num1)

		if ptr1 == ptr2 {
			t.Error("Expected different pointer addresses")
		}

		if *ptr1 != *ptr2 {
			t.Error("Expected same values")
		}
	})

	t.Run("bool pointers are independent", func(t *testing.T) {
		bool1 := true
		ptr1 := BoolPtr(bool1)
		ptr2 := BoolPtr(bool1)

		if ptr1 == ptr2 {
			t.Error("Expected different pointer addresses")
		}

		if *ptr1 != *ptr2 {
			t.Error("Expected same values")
		}
	})
}

// Benchmark tests to ensure the functions are efficient
func BenchmarkStringPtr(b *testing.B) {
	str := "benchmark string"
	for i := 0; i < b.N; i++ {
		StringPtr(str)
	}
}

func BenchmarkIntPtr(b *testing.B) {
	num := 12345
	for i := 0; i < b.N; i++ {
		IntPtr(num)
	}
}

func BenchmarkBoolPtr(b *testing.B) {
	boolean := true
	for i := 0; i < b.N; i++ {
		BoolPtr(boolean)
	}
}
