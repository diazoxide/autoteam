package task

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// TaskResponse represents the expected JSON response from the first layer agent
type TaskResponse struct {
	Tasks []Task `json:"tasks"`
}

// ParseTasksFromOutput parses task list from simple text output (one task per line)
func ParseTasksFromOutput(output string) (*TaskList, error) {
	taskList := NewTaskList()

	// Split output into lines
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Remove bullet points or dashes at the beginning
		line = strings.TrimPrefix(line, "- ")
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "• ")
		line = strings.TrimSpace(line)

		// Skip if still empty
		if line == "" {
			continue
		}

		// Create a simple task with the line as description
		task := NewTask(
			fmt.Sprintf("task-%d", i+1), // Simple ID
			TaskTypeGeneric,             // Generic type
			line,                        // Use the line as title
			line,                        // Also use as description
			PlatformGeneric,             // Generic platform
			"",                          // No completion command needed
			PriorityMedium,              // Default to medium priority
		)

		taskList.AddTask(*task)
	}

	return taskList, nil
}

// CreateEmptyTaskList creates an empty task list for when parsing fails
func CreateEmptyTaskList() *TaskList {
	return NewTaskList()
}

// extractTodoListFromOutput extracts tasks from TODO_LIST format using improved parsing
func extractTodoListFromOutput(output string) ([]string, error) {
	// Look for the TODO_LIST marker first
	todoStart := strings.Index(strings.ToUpper(output), "TODO_LIST:")
	if todoStart == -1 {
		return nil, fmt.Errorf("TODO_LIST format not found")
	}

	// Find the start of the JSON array after TODO_LIST:
	jsonStart := strings.Index(output[todoStart:], "[")
	if jsonStart == -1 {
		return nil, fmt.Errorf("TODO_LIST JSON array not found")
	}
	jsonStart += todoStart

	// Use proper JSON parsing to find the end of the array
	// Track bracket nesting and string escaping
	depth := 0
	inString := false
	escaped := false
	jsonEnd := -1

	for i := jsonStart; i < len(output); i++ {
		char := output[i]

		if escaped {
			escaped = false
			continue
		}

		if char == '\\' && inString {
			escaped = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '[' {
				depth++
			} else if char == ']' {
				depth--
				if depth == 0 {
					jsonEnd = i
					break
				}
			}
		}
	}

	if jsonEnd == -1 {
		return nil, fmt.Errorf("TODO_LIST JSON array not properly closed")
	}

	// Extract the JSON array string
	jsonArrayStr := output[jsonStart : jsonEnd+1]

	// Parse the JSON array
	var tasks []string
	if err := json.Unmarshal([]byte(jsonArrayStr), &tasks); err != nil {
		return nil, fmt.Errorf("failed to parse TODO_LIST JSON array: %w", err)
	}

	// Clean up the tasks (remove empty strings and trim whitespace)
	var cleanTasks []string
	for _, task := range tasks {
		cleanTask := strings.TrimSpace(task)
		if cleanTask != "" {
			cleanTasks = append(cleanTasks, cleanTask)
		}
	}

	return cleanTasks, nil
}

// ParseTasksFromStdout parses agent stdout output and returns a TasksJSON structure
// First tries to extract TODO_LIST format, then falls back to line-by-line parsing
func ParseTasksFromStdout(stdout string) (*TasksJSON, error) {
	tasksJSON := NewTasksJSON()

	if stdout == "" {
		return tasksJSON, nil
	}

	// First, try to extract TODO_LIST format using regex
	todoList, err := extractTodoListFromOutput(stdout)
	if err == nil {
		// Successfully extracted TODO_LIST format (even if empty)
		for _, task := range todoList {
			tasksJSON.AddTodoTask(task)
		}
		return tasksJSON, nil
	}

	// Fallback to line-by-line parsing if TODO_LIST format not found
	lines := strings.Split(strings.TrimSpace(stdout), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines
		if line == "" {
			continue
		}

		// Skip lines that contain TODO_LIST pattern (they should be handled by regex above)
		if strings.Contains(strings.ToUpper(line), "TODO_LIST:") {
			continue
		}

		// Remove common list prefixes (numbers, bullets, dashes)
		line = cleanTaskLine(line)

		// Skip if still empty after cleaning
		if line == "" {
			continue
		}

		// Add as a todo task
		tasksJSON.AddTodoTask(line)
	}

	return tasksJSON, nil
}

// cleanTaskLine removes common list prefixes and cleans up the task line
func cleanTaskLine(line string) string {
	line = strings.TrimSpace(line)

	// Remove numbered list prefixes (1. 2. etc.)
	numberRegex := regexp.MustCompile(`^\d+\.\s*`)
	line = numberRegex.ReplaceAllString(line, "")

	// Remove bullet point prefixes
	line = strings.TrimPrefix(line, "- ")
	line = strings.TrimPrefix(line, "* ")
	line = strings.TrimPrefix(line, "• ")
	line = strings.TrimPrefix(line, "→ ")
	line = strings.TrimPrefix(line, "> ")

	// Remove any remaining leading whitespace
	line = strings.TrimSpace(line)

	return line
}

// LoadTasksJSON loads TasksJSON from a JSON file content
func LoadTasksJSON(jsonContent []byte) (*TasksJSON, error) {
	var tasksJSON TasksJSON
	if err := json.Unmarshal(jsonContent, &tasksJSON); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tasks JSON: %w", err)
	}
	return &tasksJSON, nil
}

// ConvertTasksJSONToTaskList converts TasksJSON to legacy TaskList format for compatibility
func ConvertTasksJSONToTaskList(tasksJSON *TasksJSON) *TaskList {
	taskList := NewTaskList()

	// Convert todo items to tasks
	for i, todoItem := range tasksJSON.Todo {
		task := NewTask(
			fmt.Sprintf("todo-%d", i+1),
			TaskTypeGeneric,
			todoItem,
			todoItem,
			PlatformGeneric,
			"",
			PriorityMedium,
		)
		taskList.AddTask(*task)
	}

	return taskList
}
