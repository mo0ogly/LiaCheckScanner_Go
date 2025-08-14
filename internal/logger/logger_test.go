package logger

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestLoggerCreation tests the creation of a Logger instance
func TestLoggerCreation(t *testing.T) {
	logger := NewLogger()

	if logger == nil {
		t.Fatal("Logger should not be nil")
	}
}

// TestLogLevels tests different log levels
func TestLogLevels(t *testing.T) {
	tempDir := t.TempDir()
	_ = filepath.Join(tempDir, "test.log")

	logger := NewLogger()

	// Test different log levels
	testCases := []struct {
		level   string
		message string
	}{
		{"INFO", "Test info message"},
		{"WARNING", "Test warning message"},
		{"ERROR", "Test error message"},
		{"DEBUG", "Test debug message"},
	}

	for _, tc := range testCases {
		switch tc.level {
		case "INFO":
			logger.Info("Test", tc.message)
		case "WARNING":
			logger.Warning("Test", tc.message)
		case "ERROR":
			logger.Error("Test", tc.message)
		case "DEBUG":
			logger.Debug("Test", tc.message)
		}
	}
}

// TestLogFormat tests log message format
func TestLogFormat(t *testing.T) {
	logger := NewLogger()

	// Test log message format
	component := "TestComponent"
	message := "Test message"

	logger.Info(component, message)

	// The logger should not panic or crash
	// We can't easily test the output format without file access,
	// but we can ensure the logger handles the calls properly
}

// TestLogWithData tests logging with additional data
func TestLogWithData(t *testing.T) {
	logger := NewLogger()

	// Test logging with data
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	logger.Info("Test", "Message with data", data)

	// The logger should handle data properly
}

// TestLogPerformance tests logging performance
func TestLogPerformance(t *testing.T) {
	logger := NewLogger()

	// Test performance with many log messages
	start := time.Now()
	for i := 0; i < 1000; i++ {
		logger.Info("Test", fmt.Sprintf("Message %d", i))
	}
	duration := time.Since(start)

	// Logging 1000 messages should be reasonably fast (less than 1 second)
	if duration > time.Second {
		t.Errorf("Logging 1000 messages took too long: %v", duration)
	}
}

// TestLogConcurrency tests logging under concurrent access
func TestLogConcurrency(t *testing.T) {
	logger := NewLogger()
	done := make(chan bool, 10)

	// Test concurrent logging
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				logger.Info("Test", fmt.Sprintf("Concurrent message %d-%d", id, j))
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// The logger should handle concurrent access without crashing
}

// TestLogLevelFiltering tests log level filtering
func TestLogLevelFiltering(t *testing.T) {
	logger := NewLogger()

	// Test that all log levels work without error
	logger.Debug("Test", "Debug message")
	logger.Info("Test", "Info message")
	logger.Warning("Test", "Warning message")
	logger.Error("Test", "Error message")

	// All log calls should complete without error
}

// TestLogWithSpecialCharacters tests logging with special characters
func TestLogWithSpecialCharacters(t *testing.T) {
	logger := NewLogger()

	// Test logging with special characters
	specialMessages := []string{
		"Message with émojis 🚀",
		"Message with quotes \"test\"",
		"Message with newlines\nand tabs\t",
		"Message with unicode: 你好世界",
		"Message with special chars: !@#$%^&*()",
	}

	for _, msg := range specialMessages {
		logger.Info("Test", msg)
	}

	// The logger should handle special characters properly
}

// TestLogWithEmptyMessages tests logging with empty messages
func TestLogWithEmptyMessages(t *testing.T) {
	logger := NewLogger()

	// Test logging with empty messages
	logger.Info("Test", "")
	logger.Warning("Test", "")
	logger.Error("Test", "")
	logger.Debug("Test", "")

	// The logger should handle empty messages gracefully
}

// TestLogWithNilData tests logging with nil data
func TestLogWithNilData(t *testing.T) {
	logger := NewLogger()

	// Test logging with nil data
	logger.Info("Test", "Message with nil data", nil)

	// The logger should handle nil data gracefully
}

// BenchmarkLoggerCreation benchmarks logger creation
func BenchmarkLoggerCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = NewLogger()
	}
}

// BenchmarkLogging benchmarks logging performance
func BenchmarkLogging(b *testing.B) {
	logger := NewLogger()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark", "Test message")
	}
}

// BenchmarkLoggingWithData benchmarks logging with data
func BenchmarkLoggingWithData(b *testing.B) {
	logger := NewLogger()
	data := map[string]interface{}{
		"key1": "value1",
		"key2": 123,
		"key3": true,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark", "Test message with data", data)
	}
}

// Helper function to format strings
func formatString(format string, a ...interface{}) string {
	return strings.ReplaceAll(format, "%d", "0")
}
