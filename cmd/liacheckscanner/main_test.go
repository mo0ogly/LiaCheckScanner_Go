package main

import (
	"encoding/csv"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// -------------------------------------------------------
// createRequiredDirectories
// -------------------------------------------------------

func TestCreateRequiredDirectories(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := createRequiredDirectories(); err != nil {
		t.Fatalf("createRequiredDirectories: %v", err)
	}

	expected := []string{
		"logs",
		"results",
		"data",
		"config",
		"assets/icons",
		"build",
		"build/data",
		"internet-scanners",
	}

	for _, d := range expected {
		info, err := os.Stat(d)
		if err != nil {
			t.Errorf("Directory %q should exist: %v", d, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%q should be a directory", d)
		}
	}
}

func TestCreateRequiredDirectories_Idempotent(t *testing.T) {
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	if err := createRequiredDirectories(); err != nil {
		t.Fatalf("First call: %v", err)
	}
	if err := createRequiredDirectories(); err != nil {
		t.Fatalf("Second call should succeed: %v", err)
	}
}

// -------------------------------------------------------
// writeCSVToStdout
// -------------------------------------------------------

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	output, _ := io.ReadAll(r)
	os.Stdout = oldStdout
	return string(output)
}

func TestWriteCSVToStdout_EmptyData(t *testing.T) {
	output := captureStdout(t, func() {
		writeCSVToStdout([]models.ScannerData{})
	})

	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}

	// Should have exactly 1 row (header only).
	if len(records) != 1 {
		t.Fatalf("Expected 1 row (header), got %d", len(records))
	}

	if len(records[0]) != len(models.CSVHeaders) {
		t.Errorf("Header columns: want %d, got %d", len(models.CSVHeaders), len(records[0]))
	}

	for i, h := range models.CSVHeaders {
		if records[0][i] != h {
			t.Errorf("Header[%d]: want %q, got %q", i, h, records[0][i])
		}
	}
}

func TestWriteCSVToStdout_WithData(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	data := []models.ScannerData{
		{
			ID:          "s1",
			IPOrCIDR:    "1.2.3.4",
			ScannerName: "shodan",
			ScannerType: models.ScannerTypeShodan,
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			Tags:        []string{"tag1"},
		},
		{
			ID:          "s2",
			IPOrCIDR:    "5.6.7.8",
			ScannerName: "censys",
			ScannerType: models.ScannerTypeCensys,
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			Tags:        []string{"tag2"},
		},
	}

	output := captureStdout(t, func() {
		writeCSVToStdout(data)
	})

	reader := csv.NewReader(strings.NewReader(output))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV parse error: %v", err)
	}

	// Header + 2 data rows.
	if len(records) != 3 {
		t.Fatalf("Expected 3 rows (1 header + 2 data), got %d", len(records))
	}

	// Verify data row IPs.
	if records[1][1] != "1.2.3.4" {
		t.Errorf("Row 1 IP: want %q, got %q", "1.2.3.4", records[1][1])
	}
	if records[2][1] != "5.6.7.8" {
		t.Errorf("Row 2 IP: want %q, got %q", "5.6.7.8", records[2][1])
	}
}
