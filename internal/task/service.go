package task

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// Service handles all task persistence and management operations
type Service struct {
	tasksJSONPath string
}

// NewService creates a new task service instance
func NewService(agentDirectory string) *Service {
	return &Service{
		tasksJSONPath: filepath.Join(agentDirectory, "tasks.json"),
	}
}

// LoadTasks loads existing tasks from the tasks.json file with backup recovery
func (s *Service) LoadTasks(ctx context.Context) (*TasksJSON, error) {
	lgr := logger.FromContext(ctx)

	// Try to load existing tasks.json file
	existingData, err := os.ReadFile(s.tasksJSONPath)
	if err != nil {
		if os.IsNotExist(err) {
			lgr.Debug("No existing tasks.json file, returning empty tasks")
			return NewTasksJSON(), nil
		}
		return nil, fmt.Errorf("failed to read tasks.json: %w", err)
	}

	// Parse existing tasks
	tasksJSON, err := LoadTasksJSON(existingData)
	if err != nil {
		lgr.Warn("Failed to parse existing tasks.json, attempting backup recovery", zap.Error(err))

		// Try to recover from backup
		backupPath := s.tasksJSONPath + ".backup"
		if backupData, backupErr := os.ReadFile(backupPath); backupErr == nil {
			if backupTasksJSON, backupParseErr := LoadTasksJSON(backupData); backupParseErr == nil {
				lgr.Info("Successfully recovered tasks from backup file",
					zap.String("backup_path", backupPath),
					zap.Int("todo_count", backupTasksJSON.TodoCount()),
					zap.Int("done_count", backupTasksJSON.DoneCount()))

				// Restore the backup as the main file
				if restoreErr := s.SaveTasks(ctx, backupTasksJSON); restoreErr == nil {
					lgr.Info("Backup restored as main tasks.json file")
					return backupTasksJSON, nil
				} else {
					lgr.Warn("Failed to restore backup file", zap.Error(restoreErr))
				}

				return backupTasksJSON, nil
			} else {
				lgr.Warn("Backup file is also corrupted", zap.Error(backupParseErr))
			}
		} else {
			lgr.Debug("No backup file available for recovery", zap.Error(backupErr))
		}

		// If both main and backup files are corrupted, return empty tasks but don't overwrite
		lgr.Error("Both main and backup tasks.json files are corrupted, returning empty tasks without overwriting existing files")
		return NewTasksJSON(), nil
	}

	lgr.Debug("Tasks loaded successfully",
		zap.String("path", s.tasksJSONPath),
		zap.Int("todo_count", tasksJSON.TodoCount()),
		zap.Int("done_count", tasksJSON.DoneCount()))

	return tasksJSON, nil
}

// SaveTasks saves tasks to the tasks.json file with atomic operations and backup
func (s *Service) SaveTasks(ctx context.Context, tasksJSON *TasksJSON) error {
	lgr := logger.FromContext(ctx)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.tasksJSONPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Validate input data
	if tasksJSON == nil {
		return fmt.Errorf("cannot save nil TasksJSON")
	}

	// Marshal to JSON with indentation for readability
	data, err := json.MarshalIndent(tasksJSON, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal tasks JSON: %w", err)
	}

	// Validate marshaled data is not empty
	if len(data) == 0 {
		return fmt.Errorf("marshaled data is empty, refusing to overwrite tasks.json")
	}

	// Use atomic write with backup strategy
	backupPath := s.tasksJSONPath + ".backup"
	tempPath := s.tasksJSONPath + ".tmp"

	// Create backup of existing file if it exists
	if _, err := os.Stat(s.tasksJSONPath); err == nil {
		if err := s.copyFile(s.tasksJSONPath, backupPath); err != nil {
			lgr.Warn("Failed to create backup, proceeding without backup", zap.Error(err))
		}
	}

	// Write to temporary file first
	if err := os.WriteFile(tempPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write temporary tasks JSON file: %w", err)
	}

	// Atomic rename - this is the critical section
	if err := os.Rename(tempPath, s.tasksJSONPath); err != nil {
		// Clean up temporary file
		os.Remove(tempPath)
		return fmt.Errorf("failed to atomically move tasks JSON file: %w", err)
	}

	// Remove backup after successful write
	os.Remove(backupPath)

	lgr.Debug("Tasks saved successfully",
		zap.String("path", s.tasksJSONPath),
		zap.Int("todo_count", tasksJSON.TodoCount()),
		zap.Int("done_count", tasksJSON.DoneCount()))

	return nil
}

// copyFile copies a file from src to dst
func (s *Service) copyFile(src, dst string) error {
	sourceData, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, sourceData, 0644)
}

// MergeNewTasks loads existing tasks and merges them with new tasks
// This preserves todo and done state across collection cycles
func (s *Service) MergeNewTasks(ctx context.Context, newTasks *TasksJSON) (*TasksJSON, error) {
	lgr := logger.FromContext(ctx)

	// Load existing tasks
	existingTasks, err := s.LoadTasks(ctx)
	if err != nil {
		lgr.Warn("Failed to load existing tasks, using new tasks only", zap.Error(err))
		return newTasks, nil
	}

	lgr.Debug("Merging tasks",
		zap.Int("existing_todo", existingTasks.TodoCount()),
		zap.Int("existing_done", existingTasks.DoneCount()),
		zap.Int("new_todo", newTasks.TodoCount()))

	// Create merged tasks starting with existing tasks
	mergedTasks := NewTasksJSON()

	// Preserve existing todo items
	for _, existingTodo := range existingTasks.Todo {
		mergedTasks.AddTodoTask(existingTodo)
	}

	// Preserve existing done items
	for _, existingDone := range existingTasks.Done {
		mergedTasks.AddDoneTask(existingDone)
	}

	// Add new todo items (avoid duplicates)
	for _, newTodo := range newTasks.Todo {
		if !mergedTasks.ContainsTodoTask(newTodo) {
			mergedTasks.AddTodoTask(newTodo)
		}
	}

	lgr.Info("Tasks merged successfully",
		zap.Int("final_todo_count", mergedTasks.TodoCount()),
		zap.Int("final_done_count", mergedTasks.DoneCount()),
		zap.Int("new_tasks_added", newTasks.TodoCount()))

	return mergedTasks, nil
}

// AddNewTasksAndSave merges new tasks with existing ones and saves the result
func (s *Service) AddNewTasksAndSave(ctx context.Context, newTasks *TasksJSON) (*TasksJSON, error) {
	// Merge with existing tasks
	mergedTasks, err := s.MergeNewTasks(ctx, newTasks)
	if err != nil {
		return nil, fmt.Errorf("failed to merge tasks: %w", err)
	}

	// Save the merged result
	if err := s.SaveTasks(ctx, mergedTasks); err != nil {
		return nil, fmt.Errorf("failed to save merged tasks: %w", err)
	}

	return mergedTasks, nil
}

// MarkTaskCompleted moves a task from todo to done
func (s *Service) MarkTaskCompleted(ctx context.Context, taskDescription string) error {
	lgr := logger.FromContext(ctx)

	// Load existing tasks
	tasksJSON, err := s.LoadTasks(ctx)
	if err != nil {
		return fmt.Errorf("failed to load tasks: %w", err)
	}

	// Check if task exists in todo list
	if !tasksJSON.ContainsTodoTask(taskDescription) {
		lgr.Debug("Task not found in todo list, may have been already completed",
			zap.String("task_description", taskDescription))
		return nil
	}

	// Move task from todo to done
	tasksJSON.MoveToDone(taskDescription)

	// Save updated tasks
	if err := s.SaveTasks(ctx, tasksJSON); err != nil {
		return fmt.Errorf("failed to save updated tasks: %w", err)
	}

	lgr.Info("Task marked as completed",
		zap.String("task_description", taskDescription),
		zap.Int("remaining_todo_count", tasksJSON.TodoCount()),
		zap.Int("done_count", tasksJSON.DoneCount()))

	return nil
}

// CreateEmpty creates an empty tasks.json file
func (s *Service) CreateEmpty(ctx context.Context) error {
	emptyTasks := NewTasksJSON()
	return s.SaveTasks(ctx, emptyTasks)
}

// GetTasksPath returns the path to the tasks.json file
func (s *Service) GetTasksPath() string {
	return s.tasksJSONPath
}

// ConvertToTaskList converts TasksJSON to legacy TaskList format for compatibility
func (s *Service) ConvertToTaskList(tasksJSON *TasksJSON) *TaskList {
	return ConvertTasksJSONToTaskList(tasksJSON)
}
