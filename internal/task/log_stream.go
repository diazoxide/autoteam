package task

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"autoteam/internal/logger"

	"go.uber.org/zap"
)

// StreamingLogger handles streaming logs to separate files per task
type StreamingLogger struct {
	logDir string
}

// NewStreamingLogger creates a new streaming logger for the given directory
func NewStreamingLogger(workingDir string) *StreamingLogger {
	logDir := filepath.Join(workingDir, "logs")
	return &StreamingLogger{
		logDir: logDir,
	}
}

// NormalizeTaskText normalizes a task description for use as a filename
// Format: "{Notification ID} - {NOTIFICATION URL} - {NOTIFICATION TEXT}"
func NormalizeTaskText(taskDescription string) string {
	// Extract the notification text part (everything after the second " - ")
	parts := strings.Split(taskDescription, " - ")
	var notificationText string
	if len(parts) >= 3 {
		notificationText = strings.Join(parts[2:], " - ")
	} else {
		notificationText = taskDescription
	}

	// Replace invalid filename characters with underscores
	reg := regexp.MustCompile(`[^\w\-_\s\.]+`)
	normalized := reg.ReplaceAllString(notificationText, "_")

	// Replace spaces with underscores and collapse multiple underscores
	normalized = strings.ReplaceAll(normalized, " ", "_")
	normalized = regexp.MustCompile(`_+`).ReplaceAllString(normalized, "_")

	// Convert to lowercase
	normalized = strings.ToLower(normalized)

	// Trim underscores from start and end
	normalized = strings.Trim(normalized, "_")

	// Limit length to 100 characters to avoid filesystem issues
	if len(normalized) > 100 {
		normalized = normalized[:100]
		// Trim trailing underscore if truncated
		normalized = strings.TrimRight(normalized, "_")
	}

	// Ensure it's not empty
	if normalized == "" {
		normalized = "unknown_task"
	}

	return normalized
}

// CreateLogFile creates a log file for the given task and returns a file writer
func (sl *StreamingLogger) CreateLogFile(ctx context.Context, taskDescription string) (*os.File, error) {
	lgr := logger.FromContext(ctx)

	// Ensure logs directory exists
	if err := os.MkdirAll(sl.logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Normalize task text for filename and add timestamp
	normalizedName := NormalizeTaskText(taskDescription)
	timestamp := time.Now().Format("20060102-150405") // YYYYMMDD-HHMMSS format
	logFileName := fmt.Sprintf("%s-%s.log", timestamp, normalizedName)
	logFilePath := filepath.Join(sl.logDir, logFileName)

	lgr.Info("Creating log file for task",
		zap.String("task_description", taskDescription),
		zap.String("normalized_name", normalizedName),
		zap.String("timestamp", timestamp),
		zap.String("log_file", logFilePath))

	// Create/open the log file for appending
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// Write header with timestamp
	header := fmt.Sprintf("=== Task Execution Log - %s ===\nTask: %s\n\n",
		getCurrentTimestamp(), taskDescription)
	if _, err := file.WriteString(header); err != nil {
		file.Close()
		return nil, fmt.Errorf("failed to write log header: %w", err)
	}

	return file, nil
}

// getCurrentTimestamp returns current timestamp in a readable format
func getCurrentTimestamp() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

// StreamingWriter wraps an io.Writer to provide streaming functionality
type StreamingWriter struct {
	file   *os.File
	writer io.Writer
}

// NewStreamingWriter creates a new streaming writer that writes to both file and original writer
func NewStreamingWriter(file *os.File, originalWriter io.Writer) *StreamingWriter {
	return &StreamingWriter{
		file:   file,
		writer: io.MultiWriter(file, originalWriter),
	}
}

// Write implements io.Writer interface
func (sw *StreamingWriter) Write(p []byte) (n int, err error) {
	return sw.writer.Write(p)
}

// Close closes the underlying file
func (sw *StreamingWriter) Close() error {
	if sw.file != nil {
		return sw.file.Close()
	}
	return nil
}
