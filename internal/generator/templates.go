package generator

import (
	"strings"
	"text/template"
)

// GetTemplateFunctions returns the template functions map for use in templates
func GetTemplateFunctions() template.FuncMap {
	return template.FuncMap{
		"indent":              indentFunction,
		"escapeDockerCompose": escapeDockerComposeFunction,
		"join":                joinFunction,
		"add":                 addFunction,
	}
}

// indentFunction indents each line of text by the specified number of spaces
func indentFunction(spaces int, text string) string {
	if text == "" {
		return text
	}

	// Create the indentation string
	indentation := strings.Repeat(" ", spaces)

	// Split text into lines and add indentation to each line
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if line != "" { // Don't indent empty lines
			lines[i] = indentation + line
		}
	}

	return strings.Join(lines, "\n")
}

// escapeDockerComposeFunction escapes single $ with $$ for Docker Compose
// This prevents Docker Compose from trying to substitute shell variables
func escapeDockerComposeFunction(text string) string {
	return strings.ReplaceAll(text, "$", "$$")
}

// joinFunction joins a slice of strings with a separator
func joinFunction(slice []string, separator string) string {
	return strings.Join(slice, separator)
}

// addFunction adds two integers
func addFunction(a, b int) int {
	return a + b
}
