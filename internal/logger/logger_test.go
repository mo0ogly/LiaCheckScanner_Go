package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/lia/liacheckscanner_go/internal/models"
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

// -------------------------------------------------------
// SetLogLevel / GetLogLevel
// -------------------------------------------------------

func TestSetLogLevel_GetLogLevel(t *testing.T) {
	l := NewLogger()

	// Default should be INFO.
	if l.GetLogLevel() != models.LogLevelInfo {
		t.Errorf("Default log level: want INFO, got %s", l.GetLogLevel())
	}

	l.SetLogLevel(models.LogLevelDebug)
	if l.GetLogLevel() != models.LogLevelDebug {
		t.Errorf("After SetLogLevel(DEBUG): want DEBUG, got %s", l.GetLogLevel())
	}

	l.SetLogLevel(models.LogLevelError)
	if l.GetLogLevel() != models.LogLevelError {
		t.Errorf("After SetLogLevel(ERROR): want ERROR, got %s", l.GetLogLevel())
	}

	l.SetLogLevel(models.LogLevelCritical)
	if l.GetLogLevel() != models.LogLevelCritical {
		t.Errorf("After SetLogLevel(CRITICAL): want CRITICAL, got %s", l.GetLogLevel())
	}
}

// -------------------------------------------------------
// Critical level
// -------------------------------------------------------

func TestCritical(t *testing.T) {
	l := NewLogger()

	l.Critical("Test", "Critical error occurred")

	entries := l.GetEntries()
	found := false
	for _, e := range entries {
		if e.Level == models.LogLevelCritical && e.Message == "Critical error occurred" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Expected to find Critical-level entry in log entries")
	}
}

// -------------------------------------------------------
// GetEntries / GetRecentEntries / ClearEntries
// -------------------------------------------------------

func TestGetEntries_ReturnsAll(t *testing.T) {
	l := NewLogger()

	l.Info("A", "msg1")
	l.Warning("B", "msg2")
	l.Error("C", "msg3")

	entries := l.GetEntries()
	if len(entries) < 3 {
		t.Errorf("Expected at least 3 entries, got %d", len(entries))
	}
}

func TestGetRecentEntries(t *testing.T) {
	l := NewLogger()

	for i := 0; i < 10; i++ {
		l.Info("Test", fmt.Sprintf("msg %d", i))
	}

	recent := l.GetRecentEntries(3)
	if len(recent) != 3 {
		t.Errorf("Expected 3 recent entries, got %d", len(recent))
	}

	// The last entry should be "msg 9".
	if recent[2].Message != "msg 9" {
		t.Errorf("Last recent entry: want %q, got %q", "msg 9", recent[2].Message)
	}
}

func TestGetRecentEntries_MoreThanAvailable(t *testing.T) {
	l := NewLogger()

	l.Info("Test", "only one")

	recent := l.GetRecentEntries(100)
	if len(recent) != 1 {
		t.Errorf("Expected 1 entry, got %d", len(recent))
	}
}

func TestClearEntries(t *testing.T) {
	l := NewLogger()

	l.Info("Test", "will be cleared")
	l.Error("Test", "also cleared")

	l.ClearEntries()

	entries := l.GetEntries()
	if len(entries) != 0 {
		t.Errorf("Expected 0 entries after clear, got %d", len(entries))
	}
}

// -------------------------------------------------------
// shouldLog filtering
// -------------------------------------------------------

func TestShouldLog_Filtering(t *testing.T) {
	l := NewLogger()

	l.SetLogLevel(models.LogLevelWarning)
	l.ClearEntries()

	l.Debug("Test", "debug - should be filtered")
	l.Info("Test", "info - should be filtered")
	l.Warning("Test", "warning - should appear")
	l.Error("Test", "error - should appear")

	entries := l.GetEntries()
	if len(entries) != 2 {
		t.Errorf("Expected 2 entries (warning + error), got %d", len(entries))
	}
}

// -------------------------------------------------------
// Close
// -------------------------------------------------------

func TestClose(t *testing.T) {
	l := NewLogger()

	err := l.Close()
	if err != nil {
		t.Errorf("Close should not return error: %v", err)
	}

	// After Close, logFile should still be set but closed.
	// A second Close will return an error (file already closed), which is fine.
	_ = l.Close()
}

func TestClose_NilLogFile(t *testing.T) {
	l := &Logger{}
	err := l.Close()
	if err != nil {
		t.Errorf("Close with nil logFile should not return error: %v", err)
	}
}

// -------------------------------------------------------
// cleanupOldLogs
// -------------------------------------------------------

func TestCleanupOldLogs(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{backups: 2}

	// Create 5 log files with staggered modification times.
	for i := 0; i < 5; i++ {
		name := filepath.Join(dir, fmt.Sprintf("test_%d.log", i))
		if err := os.WriteFile(name, []byte("log data"), 0644); err != nil {
			t.Fatalf("WriteFile: %v", err)
		}
		// Stagger mod times so sort works predictably.
		mtime := time.Now().Add(time.Duration(i) * time.Second)
		os.Chtimes(name, mtime, mtime)
	}

	// Wait briefly to ensure file timestamps are distinct.
	time.Sleep(10 * time.Millisecond)

	l.cleanupOldLogs(dir)

	// With backups=2, we keep 3 files (2 backups + 1 current).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}

	logCount := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".log" {
			logCount++
		}
	}

	if logCount != 3 {
		t.Errorf("Expected 3 log files (backups=2 + current), got %d", logCount)
	}
}

func TestCleanupOldLogs_ZeroBackups_NoOp(t *testing.T) {
	dir := t.TempDir()
	l := &Logger{backups: 0}

	for i := 0; i < 3; i++ {
		name := filepath.Join(dir, fmt.Sprintf("test_%d.log", i))
		os.WriteFile(name, []byte("log data"), 0644)
	}

	l.cleanupOldLogs(dir)

	// With backups=0, the function should be a no-op.
	entries, _ := os.ReadDir(dir)
	if len(entries) != 3 {
		t.Errorf("With backups=0, no files should be removed. Got %d files", len(entries))
	}
}

// Helper function to format strings
func formatString(format string, a ...interface{}) string {
	return strings.ReplaceAll(format, "%d", "0")
}
