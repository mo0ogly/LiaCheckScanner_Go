package gui

import (
	"encoding/csv"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// -------------------------------------------------------
// CountUniqueIPs
// -------------------------------------------------------

func TestCountUniqueIPs(t *testing.T) {
	tests := []struct {
		name string
		data []models.ScannerData
		want int
	}{
		{"empty", nil, 0},
		{"duplicates", []models.ScannerData{
			{IPOrCIDR: "1.1.1.1"},
			{IPOrCIDR: "1.1.1.1"},
			{IPOrCIDR: "2.2.2.2"},
		}, 2},
		{"all different", []models.ScannerData{
			{IPOrCIDR: "1.1.1.1"},
			{IPOrCIDR: "2.2.2.2"},
			{IPOrCIDR: "3.3.3.3"},
		}, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountUniqueIPs(tc.data); got != tc.want {
				t.Errorf("CountUniqueIPs() = %d, want %d", got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// CountUniqueCountries
// -------------------------------------------------------

func TestCountUniqueCountries(t *testing.T) {
	tests := []struct {
		name string
		data []models.ScannerData
		want int
	}{
		{"empty", nil, 0},
		{"empty codes ignored", []models.ScannerData{
			{CountryCode: ""},
			{CountryCode: "US"},
		}, 1},
		{"duplicates", []models.ScannerData{
			{CountryCode: "US"},
			{CountryCode: "US"},
			{CountryCode: "FR"},
		}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountUniqueCountries(tc.data); got != tc.want {
				t.Errorf("CountUniqueCountries() = %d, want %d", got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// CountUniqueScanners
// -------------------------------------------------------

func TestCountUniqueScanners(t *testing.T) {
	tests := []struct {
		name string
		data []models.ScannerData
		want int
	}{
		{"empty", nil, 0},
		{"duplicates", []models.ScannerData{
			{ScannerName: "Shodan"},
			{ScannerName: "Shodan"},
			{ScannerName: "Censys"},
		}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountUniqueScanners(tc.data); got != tc.want {
				t.Errorf("CountUniqueScanners() = %d, want %d", got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// CountHighRisk
// -------------------------------------------------------

func TestCountHighRisk(t *testing.T) {
	tests := []struct {
		name string
		data []models.ScannerData
		want int
	}{
		{"empty", nil, 0},
		{"none high", []models.ScannerData{
			{RiskLevel: "Low"},
			{RiskLevel: "Medium"},
		}, 0},
		{"mixed", []models.ScannerData{
			{RiskLevel: "High"},
			{RiskLevel: "Low"},
			{RiskLevel: "High"},
		}, 2},
		{"all high", []models.ScannerData{
			{RiskLevel: "High"},
			{RiskLevel: "High"},
		}, 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountHighRisk(tc.data); got != tc.want {
				t.Errorf("CountHighRisk() = %d, want %d", got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// CountRiskLevels
// -------------------------------------------------------

func TestCountRiskLevels(t *testing.T) {
	tests := []struct {
		name string
		data []models.ScannerData
		want int
	}{
		{"empty", nil, 0},
		{"three distinct", []models.ScannerData{
			{RiskLevel: "High"},
			{RiskLevel: "Medium"},
			{RiskLevel: "Low"},
			{RiskLevel: "High"},
		}, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := CountRiskLevels(tc.data); got != tc.want {
				t.Errorf("CountRiskLevels() = %d, want %d", got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// FilterAdvancedSearch
// -------------------------------------------------------

func TestFilterAdvancedSearch_NoFilters(t *testing.T) {
	data := []models.ScannerData{
		{IPOrCIDR: "1.2.3.4", ScannerName: "Shodan", CountryCode: "US", RiskLevel: "High"},
		{IPOrCIDR: "5.6.7.8", ScannerName: "Censys", CountryCode: "FR", RiskLevel: "Low"},
	}
	results := FilterAdvancedSearch(data, "", "All Countries", "All Scanners", "All Risk Levels")
	if len(results) != 2 {
		t.Errorf("Expected all 2 results, got %d", len(results))
	}
}

func TestFilterAdvancedSearch_QueryFilter(t *testing.T) {
	data := []models.ScannerData{
		{IPOrCIDR: "1.2.3.4"},
		{IPOrCIDR: "1.2.3.5"},
		{IPOrCIDR: "9.9.9.9"},
	}
	results := FilterAdvancedSearch(data, "1.2.3", "All Countries", "All Scanners", "All Risk Levels")
	if len(results) != 2 {
		t.Errorf("Expected 2 matches for '1.2.3', got %d", len(results))
	}
}

func TestFilterAdvancedSearch_CountryFilter(t *testing.T) {
	data := []models.ScannerData{
		{IPOrCIDR: "1.1.1.1", CountryCode: "US"},
		{IPOrCIDR: "2.2.2.2", CountryCode: "FR"},
	}
	results := FilterAdvancedSearch(data, "", "US", "All Scanners", "All Risk Levels")
	if len(results) != 1 || results[0].CountryCode != "US" {
		t.Errorf("Expected 1 US result, got %d", len(results))
	}
}

func TestFilterAdvancedSearch_CombinedFilters(t *testing.T) {
	data := []models.ScannerData{
		{IPOrCIDR: "1.1.1.1", ScannerName: "Shodan", CountryCode: "US", RiskLevel: "High"},
		{IPOrCIDR: "2.2.2.2", ScannerName: "Shodan", CountryCode: "FR", RiskLevel: "High"},
		{IPOrCIDR: "3.3.3.3", ScannerName: "Censys", CountryCode: "US", RiskLevel: "Low"},
	}
	results := FilterAdvancedSearch(data, "", "US", "Shodan", "High")
	if len(results) != 1 || results[0].IPOrCIDR != "1.1.1.1" {
		t.Errorf("Expected 1 combined-filter match, got %d", len(results))
	}
}

func TestFilterAdvancedSearch_CaseInsensitive(t *testing.T) {
	data := []models.ScannerData{
		{IPOrCIDR: "10.0.0.1", ScannerName: "Shodan"},
	}
	results := FilterAdvancedSearch(data, "SHODAN", "All Countries", "All Scanners", "All Risk Levels")
	if len(results) != 1 {
		t.Errorf("Case-insensitive query should match, got %d results", len(results))
	}
}

// -------------------------------------------------------
// CalculatePagination
// -------------------------------------------------------

func TestCalculatePagination(t *testing.T) {
	tests := []struct {
		name                                                 string
		dataLen, itemsPerPage, currentPage                   int
		wantTotal, wantPage, wantStart, wantEnd              int
	}{
		{"100 items 25/page", 100, 25, 1, 4, 1, 0, 25},
		{"100 items page 3", 100, 25, 3, 4, 3, 50, 75},
		{"page beyond total", 100, 25, 10, 4, 4, 75, 100},
		{"page 0 clamped", 100, 25, 0, 4, 1, 0, 25},
		{"0 items", 0, 25, 1, 1, 1, 0, 0},
		{"partial last page", 30, 25, 2, 2, 2, 25, 30},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			total, page, start, end := CalculatePagination(tc.dataLen, tc.itemsPerPage, tc.currentPage)
			if total != tc.wantTotal {
				t.Errorf("totalPages: got %d, want %d", total, tc.wantTotal)
			}
			if page != tc.wantPage {
				t.Errorf("validPage: got %d, want %d", page, tc.wantPage)
			}
			if start != tc.wantStart {
				t.Errorf("startIdx: got %d, want %d", start, tc.wantStart)
			}
			if end != tc.wantEnd {
				t.Errorf("endIdx: got %d, want %d", end, tc.wantEnd)
			}
		})
	}
}

// -------------------------------------------------------
// LoadCSVData
// -------------------------------------------------------

func writeCSVFile(t *testing.T, dir, name string, rows [][]string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Create %s: %v", path, err)
	}
	defer f.Close()
	w := csv.NewWriter(f)
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	w.Flush()
	return path
}

func TestLoadCSVData_ValidFile(t *testing.T) {
	dir := t.TempDir()
	rows := [][]string{
		models.CSVHeaders,
		{"id1", "1.2.3.4", "Shodan", "shodan", "shodan.nft", "US", "United States",
			"ISP1", "Org1", "NET1", "H1", "1.2.3.0/24", "arin",
			"1.2.3.0", "1.2.3.255", "v4", "DIRECT", "PARENT1",
			"2020-01-01", "2023-06-15", "AS123", "TestAS", "test.example.com",
			"85", "42", "Data Center", "example.com",
			"2024-06-15 12:00:00", "2024-06-15 12:00:00",
			"extracted,shodan", "note1", "High",
			"2024-06-15 12:00:00", "abuse@test.com", "tech@test.com"},
	}
	path := writeCSVFile(t, dir, "test.csv", rows)

	data, err := LoadCSVData(path)
	if err != nil {
		t.Fatalf("LoadCSVData: %v", err)
	}
	if len(data) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(data))
	}
	if data[0].IPOrCIDR != "1.2.3.4" {
		t.Errorf("IP: want %q, got %q", "1.2.3.4", data[0].IPOrCIDR)
	}
	if data[0].CountryCode != "US" {
		t.Errorf("Country: want %q, got %q", "US", data[0].CountryCode)
	}
	if data[0].AbuseConfidenceScore != 85 {
		t.Errorf("Score: want 85, got %d", data[0].AbuseConfidenceScore)
	}
	if data[0].AbuseEmail != "abuse@test.com" {
		t.Errorf("AbuseEmail: want %q, got %q", "abuse@test.com", data[0].AbuseEmail)
	}
	if len(data[0].Tags) != 2 {
		t.Errorf("Tags: want 2, got %d", len(data[0].Tags))
	}
}

func TestLoadCSVData_MissingFile(t *testing.T) {
	_, err := LoadCSVData("/nonexistent/path/test.csv")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

func TestLoadCSVData_HeaderOnly(t *testing.T) {
	dir := t.TempDir()
	rows := [][]string{models.CSVHeaders}
	path := writeCSVFile(t, dir, "header_only.csv", rows)

	_, err := LoadCSVData(path)
	if err == nil {
		t.Error("Expected error for header-only file")
	}
	if err != nil && !strings.Contains(err.Error(), "insufficient data") {
		t.Errorf("Expected 'insufficient data' error, got: %v", err)
	}
}
