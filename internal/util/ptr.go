package util

// StringPtr returns a pointer to the given string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value.
func IntPtr(i int) *int {
	return &i
}

// BoolPtr returns a pointer to the given boolean value.
func BoolPtr(b bool) *bool {
	return &b
}
