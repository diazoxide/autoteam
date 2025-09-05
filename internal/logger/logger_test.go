package logger

import (
	"context"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		expected      LogLevel
		expectError   bool
	}{
		{
			name:        "debug level",
			input:       "debug",
			expected:    DebugLevel,
			expectError: false,
		},
		{
			name:        "debug level uppercase",
			input:       "DEBUG",
			expected:    DebugLevel,
			expectError: false,
		},
		{
			name:        "info level",
			input:       "info",
			expected:    InfoLevel,
			expectError: false,
		},
		{
			name:        "warn level",
			input:       "warn",
			expected:    WarnLevel,
			expectError: false,
		},
		{
			name:        "warning level",
			input:       "warning",
			expected:    WarnLevel,
			expectError: false,
		},
		{
			name:        "error level",
			input:       "error",
			expected:    ErrorLevel,
			expectError: false,
		},
		{
			name:        "invalid level returns info with error",
			input:       "invalid",
			expected:    InfoLevel,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseLogLevel(tt.input)
			
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if result != tt.expected {
				t.Errorf("Expected %s but got %s", tt.expected, result)
			}
		})
	}
}

func TestLogLevel_zapLevel(t *testing.T) {
	tests := []struct {
		name     string
		level    LogLevel
		expected zapcore.Level
	}{
		{
			name:     "debug level",
			level:    DebugLevel,
			expected: zapcore.DebugLevel,
		},
		{
			name:     "info level",
			level:    InfoLevel,
			expected: zapcore.InfoLevel,
		},
		{
			name:     "warn level",
			level:    WarnLevel,
			expected: zapcore.WarnLevel,
		},
		{
			name:     "error level",
			level:    ErrorLevel,
			expected: zapcore.ErrorLevel,
		},
		{
			name:     "invalid level defaults to info",
			level:    LogLevel("invalid"),
			expected: zapcore.InfoLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.level.zapLevel()
			if result != tt.expected {
				t.Errorf("Expected %v but got %v", tt.expected, result)
			}
		})
	}
}

func TestNewLogger(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
	}{
		{
			name:  "debug logger",
			level: DebugLevel,
		},
		{
			name:  "info logger",
			level: InfoLevel,
		},
		{
			name:  "warn logger",
			level: WarnLevel,
		},
		{
			name:  "error logger",
			level: ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, err := NewLogger(tt.level)
			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			
			if logger == nil {
				t.Error("Expected logger to be created but got nil")
			}

			// Test that the logger was created with the correct level
			if logger.Core().Enabled(tt.level.zapLevel()) == false && tt.level != ErrorLevel {
				// For error level, only error and above should be enabled
				// For other levels, they should be enabled at their level
				t.Errorf("Expected logger to be enabled at %v level", tt.level.zapLevel())
			}
		})
	}
}

func TestWithLogger(t *testing.T) {
	logger, err := NewLogger(InfoLevel)
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	ctx := context.Background()
	ctxWithLogger := WithLogger(ctx, logger)

	// Test that the logger was added to context
	if ctxWithLogger.Value(loggerKey) == nil {
		t.Error("Expected logger to be added to context")
	}

	if ctxWithLogger.Value(loggerKey) != logger {
		t.Error("Expected the same logger instance in context")
	}
}

func TestFromContext(t *testing.T) {
	t.Run("context with logger", func(t *testing.T) {
		logger, err := NewLogger(DebugLevel)
		if err != nil {
			t.Fatalf("Failed to create logger: %v", err)
		}

		ctx := WithLogger(context.Background(), logger)
		retrievedLogger := FromContext(ctx)

		if retrievedLogger != logger {
			t.Error("Expected to retrieve the same logger from context")
		}
	})

	t.Run("context without logger", func(t *testing.T) {
		ctx := context.Background()
		logger := FromContext(ctx)

		if logger == nil {
			t.Error("Expected fallback logger to be created")
		}

		// Should be able to log without panic
		logger.Info("test message")
	})
}

func TestSetupContext(t *testing.T) {
	tests := []struct {
		name  string
		level LogLevel
	}{
		{
			name:  "setup debug context",
			level: DebugLevel,
		},
		{
			name:  "setup info context",
			level: InfoLevel,
		},
		{
			name:  "setup warn context",
			level: WarnLevel,
		},
		{
			name:  "setup error context",
			level: ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			ctxWithLogger, err := SetupContext(ctx, tt.level)

			if err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			logger := FromContext(ctxWithLogger)
			if logger == nil {
				t.Error("Expected logger to be available in context")
			}

			// Test that we can log at the specified level
			logger.Info("test message")
		})
	}
}

func TestLoggerIntegration(t *testing.T) {
	// Test full workflow: parse level -> create logger -> add to context -> retrieve
	levelStr := "debug"
	
	level, err := ParseLogLevel(levelStr)
	if err != nil {
		t.Fatalf("Failed to parse log level: %v", err)
	}

	ctx, err := SetupContext(context.Background(), level)
	if err != nil {
		t.Fatalf("Failed to setup context: %v", err)
	}

	logger := FromContext(ctx)
	if logger == nil {
		t.Fatal("Expected logger from context")
	}

	// Test that we can log at different levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")
}

func TestLogLevelConstants(t *testing.T) {
	// Test that our constants are defined correctly
	if DebugLevel != "debug" {
		t.Errorf("Expected DebugLevel to be 'debug', got %s", DebugLevel)
	}
	
	if InfoLevel != "info" {
		t.Errorf("Expected InfoLevel to be 'info', got %s", InfoLevel)
	}
	
	if WarnLevel != "warn" {
		t.Errorf("Expected WarnLevel to be 'warn', got %s", WarnLevel)
	}
	
	if ErrorLevel != "error" {
		t.Errorf("Expected ErrorLevel to be 'error', got %s", ErrorLevel)
	}
}