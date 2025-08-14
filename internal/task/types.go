package task

import (
	"time"
)

// Task represents a single actionable task
type Task struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Priority      int               `json:"priority"`
	Title         string            `json:"title"`
	Description   string            `json:"description"`
	Platform      string            `json:"platform"`
	CompletionCmd string            `json:"completion_cmd"`
	Context       map[string]string `json:"context"`
	CreatedAt     time.Time         `json:"created_at"`
}

// TaskList represents a collection of tasks with metadata
type TaskList struct {
	Tasks     []Task    `json:"tasks"`
	Timestamp time.Time `json:"timestamp"`
}

// TasksJSON represents the JSON structure for task persistence
type TasksJSON struct {
	Todo []string `json:"todo"`
	Done []string `json:"done"`
}

// Priority constants
const (
	PriorityCritical = 1
	PriorityHigh     = 2
	PriorityMedium   = 3
	PriorityLow      = 4
)

// Common task types
const (
	TaskTypePRReview       = "pr_review"
	TaskTypeIssueAssigned  = "issue_assigned"
	TaskTypeMention        = "mention"
	TaskTypeSlackMessage   = "slack_message"
	TaskTypeFailedWorkflow = "failed_workflow"
	TaskTypeUnreadComment  = "unread_comment"
	TaskTypeGeneric        = "generic"
)

// Platform constants
const (
	PlatformGitHub  = "github"
	PlatformSlack   = "slack"
	PlatformJira    = "jira"
	PlatformGeneric = "generic"
)

// NewTask creates a new task with the current timestamp
func NewTask(id, taskType, title, description, platform, completionCmd string, priority int) *Task {
	return &Task{
		ID:            id,
		Type:          taskType,
		Priority:      priority,
		Title:         title,
		Description:   description,
		Platform:      platform,
		CompletionCmd: completionCmd,
		Context:       make(map[string]string),
		CreatedAt:     time.Now(),
	}
}

// NewTaskList creates a new task list with the current timestamp
func NewTaskList() *TaskList {
	return &TaskList{
		Tasks:     make([]Task, 0),
		Timestamp: time.Now(),
	}
}

// AddTask adds a task to the task list
func (tl *TaskList) AddTask(task Task) {
	tl.Tasks = append(tl.Tasks, task)
}

// GetHighestPriorityTask returns the task with the highest priority (lowest number)
func (tl *TaskList) GetHighestPriorityTask() *Task {
	if len(tl.Tasks) == 0 {
		return nil
	}

	highestPriorityTask := &tl.Tasks[0]
	for i := 1; i < len(tl.Tasks); i++ {
		if tl.Tasks[i].Priority < highestPriorityTask.Priority {
			highestPriorityTask = &tl.Tasks[i]
		}
	}

	return highestPriorityTask
}

// FilterByPriority returns tasks with the specified priority
func (tl *TaskList) FilterByPriority(priority int) []Task {
	var filtered []Task
	for _, task := range tl.Tasks {
		if task.Priority == priority {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterByType returns tasks with the specified type
func (tl *TaskList) FilterByType(taskType string) []Task {
	var filtered []Task
	for _, task := range tl.Tasks {
		if task.Type == taskType {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// FilterByPlatform returns tasks from the specified platform
func (tl *TaskList) FilterByPlatform(platform string) []Task {
	var filtered []Task
	for _, task := range tl.Tasks {
		if task.Platform == platform {
			filtered = append(filtered, task)
		}
	}
	return filtered
}

// Count returns the total number of tasks
func (tl *TaskList) Count() int {
	return len(tl.Tasks)
}

// IsEmpty returns true if there are no tasks
func (tl *TaskList) IsEmpty() bool {
	return len(tl.Tasks) == 0
}

// NewTasksJSON creates a new empty TasksJSON structure
func NewTasksJSON() *TasksJSON {
	return &TasksJSON{
		Todo: make([]string, 0),
		Done: make([]string, 0),
	}
}

// AddTodoTask adds a task to the todo list
func (tj *TasksJSON) AddTodoTask(task string) {
	if task != "" {
		tj.Todo = append(tj.Todo, task)
	}
}

// MoveToDone moves a task from todo to done
func (tj *TasksJSON) MoveToDone(task string) {
	// Remove from todo
	for i, todoTask := range tj.Todo {
		if todoTask == task {
			tj.Todo = append(tj.Todo[:i], tj.Todo[i+1:]...)
			break
		}
	}
	// Add to done
	tj.Done = append(tj.Done, task)
}

// HasTasks returns true if there are any tasks (todo or done)
func (tj *TasksJSON) HasTasks() bool {
	return len(tj.Todo) > 0 || len(tj.Done) > 0
}

// TodoCount returns the number of todo tasks
func (tj *TasksJSON) TodoCount() int {
	return len(tj.Todo)
}

// DoneCount returns the number of done tasks
func (tj *TasksJSON) DoneCount() int {
	return len(tj.Done)
}

// AddDoneTask adds a task to the done list
func (tj *TasksJSON) AddDoneTask(task string) {
	if task != "" {
		tj.Done = append(tj.Done, task)
	}
}

// ContainsTodoTask checks if a task is already in the todo list
func (tj *TasksJSON) ContainsTodoTask(task string) bool {
	for _, todoTask := range tj.Todo {
		if todoTask == task {
			return true
		}
	}
	return false
}
