package extractor

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// ---------------------------------------------------------------------------
// Integration tests: exercise the full IP-extraction pipeline with mock data
// on disk (no network calls).
// ---------------------------------------------------------------------------

// TestIntegration_FullPipeline creates a temp directory tree with several .nft
// files containing known IPs, runs the extraction pipeline, and verifies that
// the correct unique IPs are returned.
func TestIntegration_FullPipeline(t *testing.T) {
	dir := t.TempDir()

	// Create subdirectories mimicking a scanner repository layout.
	scannerDir := filepath.Join(dir, "scanners")
	if err := os.MkdirAll(scannerDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// shodan.nft: 3 unique IPv4
	shodanContent := `# Shodan scanner IPs
table inet filter {
    set shodan_v4 {
        type ipv4_addr
        flags interval
        elements = {
            198.20.69.74,
            198.20.69.98,
            71.6.146.185,
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(scannerDir, "shodan.nft"), []byte(shodanContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// censys.nft: 2 unique + 1 duplicate of shodan
	censysContent := `# Censys scanner IPs
set censys_v4 {
    type ipv4_addr
    flags interval
    elements = {
        162.142.125.10,
        167.94.138.50,
        198.20.69.74,
    }
}
`
	if err := os.WriteFile(filepath.Join(scannerDir, "censys.nft"), []byte(censysContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// binaryedge.nft: 1 unique + 1 CIDR block
	binaryedgeContent := `set binaryedge_v4 {
    elements = {
        45.143.200.10,
        45.143.201.0/24,
    }
}
`
	if err := os.WriteFile(filepath.Join(scannerDir, "binaryedge.nft"), []byte(binaryedgeContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// A non-.nft file that should be ignored.
	if err := os.WriteFile(filepath.Join(scannerDir, "readme.txt"), []byte("99.99.99.99\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, scannerDir)
	ips, err := ext.parseFilesForIPs(scannerDir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	// Expected unique IPs (total 6 after deduplication):
	expectedIPs := []string{
		"162.142.125.10",
		"167.94.138.50",
		"198.20.69.74",
		"198.20.69.98",
		"45.143.200.10",
		"45.143.201.0/24",
		"71.6.146.185",
	}

	sort.Strings(ips)

	if len(ips) != len(expectedIPs) {
		t.Fatalf("Expected %d unique IPs, got %d: %v", len(expectedIPs), len(ips), ips)
	}

	for i, want := range expectedIPs {
		if ips[i] != want {
			t.Errorf("ips[%d]: want %q, got %q", i, want, ips[i])
		}
	}
}

// TestIntegration_Deduplication verifies that IPs appearing in multiple files
// and multiple times within the same file are properly deduplicated.
func TestIntegration_Deduplication(t *testing.T) {
	dir := t.TempDir()

	// File 1: same IP repeated 5 times.
	f1Content := "10.0.0.1\n10.0.0.1\n10.0.0.1\n10.0.0.1\n10.0.0.1\n"
	if err := os.WriteFile(filepath.Join(dir, "dup1.nft"), []byte(f1Content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// File 2: same IP as file 1, plus a new one.
	f2Content := "10.0.0.1\n10.0.0.2\n"
	if err := os.WriteFile(filepath.Join(dir, "dup2.nft"), []byte(f2Content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// File 3: same IPs as both files.
	f3Content := "10.0.0.1\n10.0.0.2\n"
	if err := os.WriteFile(filepath.Join(dir, "dup3.nft"), []byte(f3Content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("Expected exactly 2 unique IPs after dedup, got %d: %v", len(ips), ips)
	}

	sort.Strings(ips)
	if ips[0] != "10.0.0.1" || ips[1] != "10.0.0.2" {
		t.Errorf("Expected [10.0.0.1, 10.0.0.2], got %v", ips)
	}
}

// TestIntegration_ExportJSON extracts IPs, creates ScannerData entries, saves
// them as JSON, and verifies the output file is valid JSON with the expected
// structure.
func TestIntegration_ExportJSON(t *testing.T) {
	dir := t.TempDir()

	nftContent := "192.168.10.1\n192.168.10.2\n192.168.10.3\n"
	if err := os.WriteFile(filepath.Join(dir, "test.nft"), []byte(nftContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	// Build ScannerData entries from the IPs.
	var data []models.ScannerData
	for i, ip := range ips {
		data = append(data, models.ScannerData{
			ID:          "int_" + ip,
			IPOrCIDR:    ip,
			ScannerName: "integration_test",
			ScannerType: models.ScannerTypeOther,
			RiskLevel:   ext.getRiskLevel(40 + i*20),
			Tags:        []string{"integration"},
		})
	}

	jsonFile := "integration_output.json"
	if err := ext.SaveToJSON(data, jsonFile); err != nil {
		t.Fatalf("SaveToJSON: %v", err)
	}

	// Read back and parse.
	raw, err := os.ReadFile(filepath.Join(ext.config.ResultsDir, jsonFile))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var loaded []models.ScannerData
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("JSON Unmarshal: %v", err)
	}

	if len(loaded) != len(data) {
		t.Fatalf("JSON record count: want %d, got %d", len(data), len(loaded))
	}

	// Verify each IP appears in the output.
	ipSet := map[string]bool{}
	for _, d := range loaded {
		ipSet[d.IPOrCIDR] = true
	}
	for _, ip := range ips {
		if !ipSet[ip] {
			t.Errorf("IP %q missing from JSON output", ip)
		}
	}
}

// TestIntegration_ExportCSV extracts IPs, creates ScannerData entries, saves
// them as CSV, and verifies the output file has the correct header and data
// rows.
func TestIntegration_ExportCSV(t *testing.T) {
	dir := t.TempDir()

	nftContent := "172.16.0.1\n172.16.0.2\n"
	if err := os.WriteFile(filepath.Join(dir, "scan.nft"), []byte(nftContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	var data []models.ScannerData
	for _, ip := range ips {
		data = append(data, models.ScannerData{
			ID:          "csv_" + ip,
			IPOrCIDR:    ip,
			ScannerName: "csv_test",
			ScannerType: models.ScannerTypeCensys,
			CountryCode: "DE",
			CountryName: "Germany",
			RiskLevel:   "Low",
			Tags:        []string{"csv", "integration"},
		})
	}

	csvFile := "integration_output.csv"
	if err := ext.SaveToCSV(data, csvFile); err != nil {
		t.Fatalf("SaveToCSV: %v", err)
	}

	// Read back and parse.
	f, err := os.Open(filepath.Join(ext.config.ResultsDir, csvFile))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("CSV ReadAll: %v", err)
	}

	// First row is the header.
	if len(records) < 2 {
		t.Fatalf("Expected at least header + 1 data row, got %d rows", len(records))
	}

	header := records[0]
	if header[0] != "ID" || header[1] != "IP/CIDR" {
		t.Errorf("CSV header: want [ID, IP/CIDR, ...], got %v", header[:2])
	}

	// Should have header + 2 data rows.
	if len(records) != 3 {
		t.Errorf("Expected 3 CSV rows (1 header + 2 data), got %d", len(records))
	}

	// Verify IPs appear in data rows.
	foundIPs := map[string]bool{}
	for _, row := range records[1:] {
		foundIPs[row[1]] = true
	}
	for _, ip := range ips {
		if !foundIPs[ip] {
			t.Errorf("IP %q missing from CSV output", ip)
		}
	}
}

// TestIntegration_EmptyDirectory verifies that parsing an empty directory
// returns no IPs and no error.
func TestIntegration_EmptyDirectory(t *testing.T) {
	dir := t.TempDir()

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs on empty dir: %v", err)
	}

	if len(ips) != 0 {
		t.Errorf("Expected 0 IPs from empty directory, got %d: %v", len(ips), ips)
	}
}

// TestIntegration_MalformedNFTFiles verifies that malformed .nft files are
// handled gracefully: files with no valid IPs should produce empty results,
// and the pipeline should not return an error.
func TestIntegration_MalformedNFTFiles(t *testing.T) {
	dir := t.TempDir()

	// File with only comments.
	commentsOnly := `# This file has no IPs
# Just comments
# Nothing useful here
`
	if err := os.WriteFile(filepath.Join(dir, "comments_only.nft"), []byte(commentsOnly), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// File with garbage content (no valid IP patterns).
	garbage := `this is not a valid nft file
random text without any IP addresses
foo bar baz qux
elements = { hello, world }
`
	if err := os.WriteFile(filepath.Join(dir, "garbage.nft"), []byte(garbage), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// File with empty content.
	if err := os.WriteFile(filepath.Join(dir, "empty.nft"), []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// File with only whitespace.
	if err := os.WriteFile(filepath.Join(dir, "whitespace.nft"), []byte("   \n\n   \n\t\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs with malformed files: %v", err)
	}

	if len(ips) != 0 {
		t.Errorf("Expected 0 IPs from malformed files, got %d: %v", len(ips), ips)
	}
}

// TestIntegration_MalformedMixedWithValid verifies that a directory containing
// both malformed and valid .nft files only extracts IPs from the valid ones.
func TestIntegration_MalformedMixedWithValid(t *testing.T) {
	dir := t.TempDir()

	// Valid file.
	validContent := "10.10.10.10\n20.20.20.20\n"
	if err := os.WriteFile(filepath.Join(dir, "valid.nft"), []byte(validContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Malformed file (no IPs).
	malformedContent := "this has no IPs at all\nnothing here\n"
	if err := os.WriteFile(filepath.Join(dir, "malformed.nft"), []byte(malformedContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Empty file.
	if err := os.WriteFile(filepath.Join(dir, "empty.nft"), []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	sort.Strings(ips)
	expected := []string{"10.10.10.10", "20.20.20.20"}

	if len(ips) != len(expected) {
		t.Fatalf("Expected %d IPs, got %d: %v", len(expected), len(ips), ips)
	}
	for i, want := range expected {
		if ips[i] != want {
			t.Errorf("ips[%d]: want %q, got %q", i, want, ips[i])
		}
	}
}

// TestIntegration_NestedDirectories verifies that the pipeline walks nested
// subdirectories and finds .nft files inside them.
func TestIntegration_NestedDirectories(t *testing.T) {
	dir := t.TempDir()

	nested := filepath.Join(dir, "level1", "level2")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	// Top level file.
	if err := os.WriteFile(filepath.Join(dir, "top.nft"), []byte("1.1.1.1\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Level 1 file.
	if err := os.WriteFile(filepath.Join(dir, "level1", "mid.nft"), []byte("2.2.2.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Level 2 file.
	if err := os.WriteFile(filepath.Join(nested, "deep.nft"), []byte("3.3.3.3\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	sort.Strings(ips)
	expected := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

	if len(ips) != len(expected) {
		t.Fatalf("Expected %d IPs from nested dirs, got %d: %v", len(expected), len(ips), ips)
	}
	for i, want := range expected {
		if ips[i] != want {
			t.Errorf("ips[%d]: want %q, got %q", i, want, ips[i])
		}
	}
}

// TestIntegration_ScannerTypeMapping verifies that the mapIPsToScanners
// function correctly associates IPs with their scanner names and types
// based on file names.
func TestIntegration_ScannerTypeMapping(t *testing.T) {
	dir := t.TempDir()

	files := map[string]string{
		"shodan.nft":       "100.0.0.1\n",
		"censys.nft":       "100.0.0.2\n",
		"binaryedge.nft":   "100.0.0.3\n",
		"rapid7.nft":       "100.0.0.4\n",
		"shadowserver.nft": "100.0.0.5\n",
		"custom.nft":       "100.0.0.6\n",
	}

	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatalf("WriteFile %s: %v", name, err)
		}
	}

	ext := newTestExtractor(t, dir)
	allIPs := []string{"100.0.0.1", "100.0.0.2", "100.0.0.3", "100.0.0.4", "100.0.0.5", "100.0.0.6"}
	mapping := ext.mapIPsToScanners(allIPs)

	expectations := map[string]struct {
		scannerName string
		scannerType models.ScannerType
		sourceFile  string
	}{
		"100.0.0.1": {"shodan", models.ScannerTypeShodan, "shodan.nft"},
		"100.0.0.2": {"censys", models.ScannerTypeCensys, "censys.nft"},
		"100.0.0.3": {"binaryedge", models.ScannerTypeBinaryEdge, "binaryedge.nft"},
		"100.0.0.4": {"rapid7", models.ScannerTypeRapid7, "rapid7.nft"},
		"100.0.0.5": {"shadowserver", models.ScannerTypeShadowServer, "shadowserver.nft"},
		"100.0.0.6": {"custom", models.ScannerTypeOther, "custom.nft"},
	}

	for ip, want := range expectations {
		info, ok := mapping[ip]
		if !ok {
			t.Errorf("IP %q not found in mapping", ip)
			continue
		}
		if info.Name != want.scannerName {
			t.Errorf("IP %q scanner name: want %q, got %q", ip, want.scannerName, info.Name)
		}
		if info.Type != want.scannerType {
			t.Errorf("IP %q scanner type: want %q, got %q", ip, want.scannerType, info.Type)
		}
		if info.SourceFile != want.sourceFile {
			t.Errorf("IP %q source file: want %q, got %q", ip, want.sourceFile, info.SourceFile)
		}
	}
}

// TestIntegration_IPv6Pipeline tests the full pipeline with IPv6 addresses.
func TestIntegration_IPv6Pipeline(t *testing.T) {
	dir := t.TempDir()

	ipv6Content := `set scanners_v6 {
    type ipv6_addr
    flags interval
    elements = {
        2001:0db8:85a3:0000:0000:8a2e:0370:7334,
        fd00::1,
        2001:0db8:85a3:0000:0000:8a2e:0370:7334,
    }
}
`
	if err := os.WriteFile(filepath.Join(dir, "ipv6.nft"), []byte(ipv6Content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	// At least the two distinct IPv6 addresses should be present (duplicate removed).
	hasFullV6 := false
	hasShortV6 := false
	for _, ip := range ips {
		if strings.Contains(ip, "2001:0db8") || strings.Contains(ip, "2001:db8") {
			hasFullV6 = true
		}
		if strings.Contains(ip, "fd00") {
			hasShortV6 = true
		}
	}

	if !hasFullV6 {
		t.Errorf("Expected full IPv6 address in results, got: %v", ips)
	}
	if !hasShortV6 {
		t.Errorf("Expected short IPv6 address (fd00::1) in results, got: %v", ips)
	}
}

// TestIntegration_JSONAndCSVConsistency extracts the same data, saves to both
// JSON and CSV, and verifies both contain the same set of IPs.
func TestIntegration_JSONAndCSVConsistency(t *testing.T) {
	dir := t.TempDir()

	nftContent := "10.20.30.1\n10.20.30.2\n10.20.30.3\n"
	if err := os.WriteFile(filepath.Join(dir, "consistency.nft"), []byte(nftContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	var data []models.ScannerData
	for _, ip := range ips {
		data = append(data, models.ScannerData{
			ID:          "cons_" + ip,
			IPOrCIDR:    ip,
			ScannerName: "consistency",
			ScannerType: models.ScannerTypeOther,
			RiskLevel:   "Medium",
			Tags:        []string{"test"},
		})
	}

	// Save both formats.
	if err := ext.SaveToJSON(data, "consistency.json"); err != nil {
		t.Fatalf("SaveToJSON: %v", err)
	}
	if err := ext.SaveToCSV(data, "consistency.csv"); err != nil {
		t.Fatalf("SaveToCSV: %v", err)
	}

	// Load JSON and collect IPs.
	rawJSON, err := os.ReadFile(filepath.Join(ext.config.ResultsDir, "consistency.json"))
	if err != nil {
		t.Fatalf("ReadFile JSON: %v", err)
	}
	var jsonData []models.ScannerData
	if err := json.Unmarshal(rawJSON, &jsonData); err != nil {
		t.Fatalf("JSON Unmarshal: %v", err)
	}
	jsonIPs := make([]string, len(jsonData))
	for i, d := range jsonData {
		jsonIPs[i] = d.IPOrCIDR
	}
	sort.Strings(jsonIPs)

	// Load CSV and collect IPs.
	csvF, err := os.Open(filepath.Join(ext.config.ResultsDir, "consistency.csv"))
	if err != nil {
		t.Fatalf("Open CSV: %v", err)
	}
	defer csvF.Close()
	records, err := csv.NewReader(csvF).ReadAll()
	if err != nil {
		t.Fatalf("CSV ReadAll: %v", err)
	}
	var csvIPs []string
	for _, row := range records[1:] { // skip header
		csvIPs = append(csvIPs, row[1]) // IP/CIDR is column index 1
	}
	sort.Strings(csvIPs)

	// Compare.
	if len(jsonIPs) != len(csvIPs) {
		t.Fatalf("JSON has %d IPs, CSV has %d IPs", len(jsonIPs), len(csvIPs))
	}
	for i := range jsonIPs {
		if jsonIPs[i] != csvIPs[i] {
			t.Errorf("IP mismatch at %d: JSON=%q, CSV=%q", i, jsonIPs[i], csvIPs[i])
		}
	}
}
