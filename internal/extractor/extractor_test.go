package extractor

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

// newTestExtractor creates an Extractor with a real Logger and a DatabaseConfig
// pointing at the provided temp directory.
func newTestExtractor(t *testing.T, localPath string) *Extractor {
	t.Helper()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		RepoURL:    "https://example.com/repo",
		LocalPath:  localPath,
		ResultsDir: filepath.Join(localPath, "results"),
		LogsDir:    filepath.Join(localPath, "logs"),
	}
	return NewExtractor(cfg, log)
}

// -------------------------------------------------------
// extractIPsFromNFTFile
// -------------------------------------------------------

func TestExtractIPsFromNFTFile_IPv4(t *testing.T) {
	dir := t.TempDir()
	nftFile := filepath.Join(dir, "test.nft")

	content := `# scanner definitions
table inet filter {
    set scanners_v4 {
        type ipv4_addr
        flags interval
        elements = {
            192.168.1.1,
            10.0.0.1,
            8.8.8.8,
        }
    }
}
`
	if err := os.WriteFile(nftFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	ips, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
	if err != nil {
		t.Fatalf("extractIPsFromNFTFile: %v", err)
	}

	expected := map[string]bool{
		"192.168.1.1": false,
		"10.0.0.1":    false,
		"8.8.8.8":     false,
	}

	for _, ip := range ips {
		if _, ok := expected[ip]; ok {
			expected[ip] = true
		}
	}

	for ip, found := range expected {
		if !found {
			t.Errorf("Expected IPv4 %s to be extracted, but it was not", ip)
		}
	}
}

func TestExtractIPsFromNFTFile_CIDRBlocks(t *testing.T) {
	dir := t.TempDir()
	nftFile := filepath.Join(dir, "cidr.nft")

	content := `set blocked {
    type ipv4_addr
    flags interval
    elements = {
        10.0.0.0/8,
        172.16.0.0/12,
        192.168.0.0/16,
    }
}
`
	if err := os.WriteFile(nftFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	ips, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
	if err != nil {
		t.Fatalf("extractIPsFromNFTFile: %v", err)
	}

	expected := []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}
	for _, want := range expected {
		found := false
		for _, ip := range ips {
			if ip == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected CIDR %s to be extracted, but it was not. Got: %v", want, ips)
		}
	}
}

func TestExtractIPsFromNFTFile_IPv6(t *testing.T) {
	dir := t.TempDir()
	nftFile := filepath.Join(dir, "v6.nft")

	content := `set scanners_v6 {
    type ipv6_addr
    flags interval
    elements = {
        2001:0db8:85a3:0000:0000:8a2e:0370:7334,
        fd00::1,
        ::1,
    }
}
`
	if err := os.WriteFile(nftFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	ips, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
	if err != nil {
		t.Fatalf("extractIPsFromNFTFile: %v", err)
	}

	// At minimum the full IPv6 and the short-form fd00::1 should be found.
	if len(ips) == 0 {
		t.Fatal("Expected at least some IPv6 addresses to be extracted")
	}

	foundFull := false
	foundShort := false
	for _, ip := range ips {
		if strings.Contains(ip, "2001:0db8") || strings.Contains(ip, "2001:db8") {
			foundFull = true
		}
		if strings.Contains(ip, "fd00") {
			foundShort = true
		}
	}
	if !foundFull {
		t.Errorf("Expected full IPv6 address to be extracted, got: %v", ips)
	}
	if !foundShort {
		t.Errorf("Expected short IPv6 address (fd00::1) to be extracted, got: %v", ips)
	}
}

func TestExtractIPsFromNFTFile_SkipsComments(t *testing.T) {
	dir := t.TempDir()
	nftFile := filepath.Join(dir, "comments.nft")

	content := `# This is a comment with an IP 1.2.3.4
# Another comment 5.6.7.8

    10.20.30.40
`
	if err := os.WriteFile(nftFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	ips, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
	if err != nil {
		t.Fatalf("extractIPsFromNFTFile: %v", err)
	}

	// Only the non-comment IP should appear.
	for _, ip := range ips {
		if ip == "1.2.3.4" || ip == "5.6.7.8" {
			t.Errorf("Comment IP %s should not be extracted", ip)
		}
	}

	found := false
	for _, ip := range ips {
		if ip == "10.20.30.40" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 10.20.30.40 to be extracted from non-comment line, got: %v", ips)
	}
}

func TestExtractIPsFromNFTFile_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	nftFile := filepath.Join(dir, "empty.nft")

	if err := os.WriteFile(nftFile, []byte(""), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	ips, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
	if err != nil {
		t.Fatalf("extractIPsFromNFTFile: %v", err)
	}

	if len(ips) != 0 {
		t.Errorf("Expected 0 IPs from empty file, got %d: %v", len(ips), ips)
	}
}

func TestExtractIPsFromNFTFile_NonexistentFile(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	_, err := ext.extractIPsFromNFTFile(filepath.Join(dir, "nope.nft"), ipv4Re, ipv6Re)
	if err == nil {
		t.Fatal("Expected error for nonexistent file")
	}
}

// -------------------------------------------------------
// parseFilesForIPs  (walks directory, deduplicates)
// -------------------------------------------------------

func TestParseFilesForIPs_BasicWalk(t *testing.T) {
	dir := t.TempDir()

	// Create two .nft files with overlapping IPs.
	nft1 := filepath.Join(dir, "shodan.nft")
	nft2 := filepath.Join(dir, "censys.nft")

	if err := os.WriteFile(nft1, []byte("1.1.1.1\n2.2.2.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(nft2, []byte("2.2.2.2\n3.3.3.3\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	// Should deduplicate: 1.1.1.1, 2.2.2.2, 3.3.3.3
	if len(ips) != 3 {
		t.Errorf("Expected 3 unique IPs, got %d: %v", len(ips), ips)
	}

	sort.Strings(ips)
	expected := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}
	for i, want := range expected {
		if i >= len(ips) || ips[i] != want {
			t.Errorf("ips[%d]: want %q, got %v", i, want, ips)
		}
	}
}

func TestParseFilesForIPs_IgnoresNonNFTFiles(t *testing.T) {
	dir := t.TempDir()

	// .nft file with real IPs
	nft := filepath.Join(dir, "test.nft")
	if err := os.WriteFile(nft, []byte("4.4.4.4\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// .txt file should be ignored
	txt := filepath.Join(dir, "other.txt")
	if err := os.WriteFile(txt, []byte("5.5.5.5\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	for _, ip := range ips {
		if ip == "5.5.5.5" {
			t.Error("parseFilesForIPs should not extract IPs from .txt files")
		}
	}

	found := false
	for _, ip := range ips {
		if ip == "4.4.4.4" {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected 4.4.4.4 from .nft file, got: %v", ips)
	}
}

func TestParseFilesForIPs_SkipsDotDirs(t *testing.T) {
	dir := t.TempDir()

	// Create a hidden directory with an .nft file.
	hiddenDir := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(hiddenDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	nft := filepath.Join(hiddenDir, "scan.nft")
	if err := os.WriteFile(nft, []byte("9.9.9.9\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	for _, ip := range ips {
		if ip == "9.9.9.9" {
			t.Error("parseFilesForIPs should skip directories starting with '.'")
		}
	}
}

func TestParseFilesForIPs_NonexistentDir(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	_, err := ext.parseFilesForIPs(filepath.Join(dir, "does-not-exist"))
	if err == nil {
		t.Fatal("Expected error for nonexistent directory")
	}
}

func TestParseFilesForIPs_Deduplication(t *testing.T) {
	dir := t.TempDir()

	// Same IP in the same file repeated multiple times.
	nft := filepath.Join(dir, "dups.nft")
	content := "1.2.3.4\n1.2.3.4\n1.2.3.4\n5.6.7.8\n"
	if err := os.WriteFile(nft, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips, err := ext.parseFilesForIPs(dir)
	if err != nil {
		t.Fatalf("parseFilesForIPs: %v", err)
	}

	if len(ips) != 2 {
		t.Errorf("Expected 2 unique IPs after dedup, got %d: %v", len(ips), ips)
	}
}

// -------------------------------------------------------
// getScannerType
// -------------------------------------------------------

func TestGetScannerType(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	tests := []struct {
		name string
		want models.ScannerType
	}{
		{"shodan", models.ScannerTypeShodan},
		{"Shodan", models.ScannerTypeShodan},
		{"SHODAN", models.ScannerTypeShodan},
		{"censys", models.ScannerTypeCensys},
		{"binaryedge", models.ScannerTypeBinaryEdge},
		{"rapid7", models.ScannerTypeRapid7},
		{"shadowserver", models.ScannerTypeShadowServer},
		{"unknown_scanner", models.ScannerTypeOther},
		{"custom", models.ScannerTypeOther},
		{"", models.ScannerTypeOther},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ext.getScannerType(tc.name)
			if got != tc.want {
				t.Errorf("getScannerType(%q) = %q, want %q", tc.name, got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// getCountryName
// -------------------------------------------------------

func TestGetCountryName(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	tests := []struct {
		code string
		want string
	}{
		{"FR", "France"},
		{"US", "United States"},
		{"DE", "Germany"},
		{"GB", "United Kingdom"},
		{"CA", "Canada"},
		{"AU", "Australia"},
		{"JP", "Japan"},
		{"BR", "Brazil"},
		{"IN", "India"},
		{"RU", "Russia"},
		{"ZZ", "Unknown"},
		{"", "Unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			got := ext.getCountryName(tc.code)
			if got != tc.want {
				t.Errorf("getCountryName(%q) = %q, want %q", tc.code, got, tc.want)
			}
		})
	}
}

// -------------------------------------------------------
// getRiskLevel
// -------------------------------------------------------

func TestGetRiskLevel(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	tests := []struct {
		score int
		want  string
	}{
		{100, "Critical"},
		{80, "Critical"},
		{79, "High"},
		{60, "High"},
		{59, "Medium"},
		{40, "Medium"},
		{39, "Low"},
		{20, "Low"},
		{19, "Very Low"},
		{0, "Very Low"},
		{-5, "Very Low"},
	}

	for _, tc := range tests {
		got := ext.getRiskLevel(tc.score)
		if got != tc.want {
			t.Errorf("getRiskLevel(%d) = %q, want %q", tc.score, got, tc.want)
		}
	}
}


// -------------------------------------------------------
// applyCache / updateCache
// -------------------------------------------------------

func TestApplyCache_Miss(t *testing.T) {
	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{},
	}
	data := &models.ScannerData{IPOrCIDR: "1.2.3.4"}

	applied := cache.applyCache("1.2.3.4", data)
	if applied {
		t.Error("applyCache should return false for cache miss")
	}
}

func TestApplyCache_Hit(t *testing.T) {
	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{
			"1.2.3.4": {
				RDAPName:    "TestNet",
				RDAPHandle:  "NET-1-2-3-0-1",
				RDAPCIDR:    "1.2.3.0/24",
				Registry:    "arin",
				CountryCode: "US",
				CountryName: "United States",
				ISP:         "TestISP",
				ASN:         "AS12345",
				AbuseEmail:  "abuse@test.com",
				TechEmail:   "tech@test.com",
			},
		},
	}

	data := &models.ScannerData{IPOrCIDR: "1.2.3.4"}
	applied := cache.applyCache("1.2.3.4", data)
	if !applied {
		t.Fatal("applyCache should return true for cache hit")
	}

	if data.RDAPName != "TestNet" {
		t.Errorf("RDAPName: want %q, got %q", "TestNet", data.RDAPName)
	}
	if data.RDAPCIDR != "1.2.3.0/24" {
		t.Errorf("RDAPCIDR: want %q, got %q", "1.2.3.0/24", data.RDAPCIDR)
	}
	if data.CountryCode != "US" {
		t.Errorf("CountryCode: want %q, got %q", "US", data.CountryCode)
	}
	if data.ISP != "TestISP" {
		t.Errorf("ISP: want %q, got %q", "TestISP", data.ISP)
	}
	if data.ASN != "AS12345" {
		t.Errorf("ASN: want %q, got %q", "AS12345", data.ASN)
	}
	if data.AbuseEmail != "abuse@test.com" {
		t.Errorf("AbuseEmail: want %q, got %q", "abuse@test.com", data.AbuseEmail)
	}
	if data.TechEmail != "tech@test.com" {
		t.Errorf("TechEmail: want %q, got %q", "tech@test.com", data.TechEmail)
	}
}

func TestUpdateCache_AddsEntry(t *testing.T) {
	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{},
	}

	data := &models.ScannerData{
		IPOrCIDR:    "5.6.7.8",
		RDAPName:    "CachedNet",
		RDAPHandle:  "HANDLE-1",
		CountryCode: "FR",
		CountryName: "France",
		ISP:         "FrenchISP",
		AbuseEmail:  "abuse@fr.com",
	}

	cache.updateCache("5.6.7.8", data)

	entry, ok := cache.Entries["5.6.7.8"]
	if !ok {
		t.Fatal("updateCache should add an entry for the IP")
	}
	if entry.RDAPName != "CachedNet" {
		t.Errorf("CachedEntry.RDAPName: want %q, got %q", "CachedNet", entry.RDAPName)
	}
	if entry.CountryCode != "FR" {
		t.Errorf("CachedEntry.CountryCode: want %q, got %q", "FR", entry.CountryCode)
	}
	if entry.CachedAt == "" {
		t.Error("CachedAt should be set")
	}
	// Verify the timestamp is parseable as RFC3339.
	if _, err := time.Parse(time.RFC3339, entry.CachedAt); err != nil {
		t.Errorf("CachedAt should be RFC3339: %v", err)
	}
}

func TestUpdateCache_OverwritesExisting(t *testing.T) {
	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{
			"1.1.1.1": {RDAPName: "OldName"},
		},
	}

	data := &models.ScannerData{
		IPOrCIDR: "1.1.1.1",
		RDAPName: "NewName",
	}

	cache.updateCache("1.1.1.1", data)

	entry := cache.Entries["1.1.1.1"]
	if entry.RDAPName != "NewName" {
		t.Errorf("Overwritten RDAPName: want %q, got %q", "NewName", entry.RDAPName)
	}
}

// -------------------------------------------------------
// IsIPProcessed
// -------------------------------------------------------

func TestIsIPProcessed(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	tracker := &models.RDAPProgressTracker{
		ProcessedIPs: []string{"1.1.1.1", "2.2.2.2"},
	}

	if !ext.IsIPProcessed("1.1.1.1", tracker) {
		t.Error("1.1.1.1 should be processed")
	}
	if !ext.IsIPProcessed("2.2.2.2", tracker) {
		t.Error("2.2.2.2 should be processed")
	}
	if ext.IsIPProcessed("3.3.3.3", tracker) {
		t.Error("3.3.3.3 should NOT be processed")
	}
}

func TestIsIPProcessed_EmptyTracker(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	tracker := &models.RDAPProgressTracker{
		ProcessedIPs: []string{},
	}

	if ext.IsIPProcessed("1.1.1.1", tracker) {
		t.Error("No IP should be processed in empty tracker")
	}
}

// -------------------------------------------------------
// SaveToJSON / LoadFromJSON round-trip
// -------------------------------------------------------

func TestSaveToJSON_LoadFromJSON_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	now := time.Now().Truncate(time.Second)
	data := []models.ScannerData{
		{
			ID:          "scanner_1",
			IPOrCIDR:    "10.0.0.1",
			ScannerName: "shodan",
			ScannerType: models.ScannerTypeShodan,
			CountryCode: "US",
			RiskLevel:   "High",
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			CreatedAt:   now,
			UpdatedAt:   now,
			Tags:        []string{"test"},
		},
	}

	filename := "test_output.json"
	if err := ext.SaveToJSON(data, filename); err != nil {
		t.Fatalf("SaveToJSON: %v", err)
	}

	loaded, err := ext.LoadFromJSON(filename)
	if err != nil {
		t.Fatalf("LoadFromJSON: %v", err)
	}

	if len(loaded) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(loaded))
	}

	if loaded[0].IPOrCIDR != "10.0.0.1" {
		t.Errorf("IPOrCIDR: want %q, got %q", "10.0.0.1", loaded[0].IPOrCIDR)
	}
	if loaded[0].ScannerName != "shodan" {
		t.Errorf("ScannerName: want %q, got %q", "shodan", loaded[0].ScannerName)
	}
	if loaded[0].RiskLevel != "High" {
		t.Errorf("RiskLevel: want %q, got %q", "High", loaded[0].RiskLevel)
	}
}

func TestLoadFromJSON_MissingFile(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	_, err := ext.LoadFromJSON("does_not_exist.json")
	if err == nil {
		t.Fatal("LoadFromJSON should return error for missing file")
	}
}

// -------------------------------------------------------
// SaveToCSV
// -------------------------------------------------------

func TestSaveToCSV_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	now := time.Now()
	data := []models.ScannerData{
		{
			ID:          "csv_1",
			IPOrCIDR:    "192.168.1.100",
			ScannerName: "censys",
			ScannerType: models.ScannerTypeCensys,
			SourceFile:  "censys.nft",
			CountryCode: "DE",
			CountryName: "Germany",
			RiskLevel:   "Medium",
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			Tags:        []string{"tag1", "tag2"},
		},
	}

	filename := "test_output.csv"
	if err := ext.SaveToCSV(data, filename); err != nil {
		t.Fatalf("SaveToCSV: %v", err)
	}

	csvPath := filepath.Join(ext.config.ResultsDir, filename)
	content, err := os.ReadFile(csvPath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	csv := string(content)
	// Check that the header is present.
	if !strings.Contains(csv, "ID") || !strings.Contains(csv, "IP/CIDR") {
		t.Error("CSV should contain header row")
	}
	// Check data row.
	if !strings.Contains(csv, "192.168.1.100") {
		t.Error("CSV should contain IP data")
	}
	if !strings.Contains(csv, "censys") {
		t.Error("CSV should contain scanner name")
	}
}

// -------------------------------------------------------
// mapIPsToScanners
// -------------------------------------------------------

func TestMapIPsToScanners(t *testing.T) {
	dir := t.TempDir()

	// Create .nft files.
	shodan := filepath.Join(dir, "shodan.nft")
	censys := filepath.Join(dir, "censys.nft")
	if err := os.WriteFile(shodan, []byte("10.0.0.1\n10.0.0.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.WriteFile(censys, []byte("10.0.0.2\n10.0.0.3\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}

	mapping := ext.mapIPsToScanners(ips)

	// 10.0.0.1 should come from shodan
	if info, ok := mapping["10.0.0.1"]; ok {
		if info.SourceFile != "shodan.nft" {
			t.Errorf("10.0.0.1 SourceFile: want %q, got %q", "shodan.nft", info.SourceFile)
		}
		if info.Type != models.ScannerTypeShodan {
			t.Errorf("10.0.0.1 Type: want %q, got %q", models.ScannerTypeShodan, info.Type)
		}
	} else {
		t.Error("10.0.0.1 should be in mapping")
	}

	// 10.0.0.3 should come from censys
	if info, ok := mapping["10.0.0.3"]; ok {
		if info.SourceFile != "censys.nft" {
			t.Errorf("10.0.0.3 SourceFile: want %q, got %q", "censys.nft", info.SourceFile)
		}
		if info.Type != models.ScannerTypeCensys {
			t.Errorf("10.0.0.3 Type: want %q, got %q", models.ScannerTypeCensys, info.Type)
		}
	} else {
		t.Error("10.0.0.3 should be in mapping")
	}
}

// -------------------------------------------------------
// NewExtractor
// -------------------------------------------------------

func TestNewExtractor(t *testing.T) {
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		RepoURL:   "https://example.com/repo",
		LocalPath: "/tmp/test",
	}

	ext := NewExtractor(cfg, log)
	if ext == nil {
		t.Fatal("NewExtractor should not return nil")
	}
	if ext.logger != log {
		t.Error("Logger should be set")
	}
	if ext.config.RepoURL != "https://example.com/repo" {
		t.Error("Config RepoURL should be set")
	}
	if ext.apiClient == nil {
		t.Error("apiClient should be initialized")
	}
}

// -------------------------------------------------------
// rdapCache save / load round-trip (on-disk)
// -------------------------------------------------------

func TestRdapCache_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "cache.json")

	// Create and save a cache.
	c := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{
			"1.2.3.4": {
				RDAPName:    "TestEntry",
				CountryCode: "US",
				CachedAt:    time.Now().Format(time.RFC3339),
			},
		},
		Path: cachePath,
	}
	c.save()

	// Verify file exists.
	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	// Parse back.
	var loaded rdapCache
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	entry, ok := loaded.Entries["1.2.3.4"]
	if !ok {
		t.Fatal("Expected entry for 1.2.3.4")
	}
	if entry.RDAPName != "TestEntry" {
		t.Errorf("RDAPName: want %q, got %q", "TestEntry", entry.RDAPName)
	}
}

// -------------------------------------------------------
// Mixed IPv4/IPv6 in one file
// -------------------------------------------------------

func TestExtractIPsFromNFTFile_MixedV4V6(t *testing.T) {
	dir := t.TempDir()
	nftFile := filepath.Join(dir, "mixed.nft")

	content := `table inet filter {
    set blocked_v4 { elements = { 1.2.3.4, 5.6.7.8/24 } }
    set blocked_v6 { elements = { 2001:db8::1, fd00::/64 } }
}
`
	if err := os.WriteFile(nftFile, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	ips, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
	if err != nil {
		t.Fatalf("extractIPsFromNFTFile: %v", err)
	}

	hasV4 := false
	hasCIDR := false
	hasV6 := false
	for _, ip := range ips {
		if ip == "1.2.3.4" {
			hasV4 = true
		}
		if ip == "5.6.7.8/24" {
			hasCIDR = true
		}
		if strings.Contains(ip, "2001:db8") || strings.Contains(ip, "2001:0db8") {
			hasV6 = true
		}
	}

	if !hasV4 {
		t.Errorf("Expected IPv4 1.2.3.4, got: %v", ips)
	}
	if !hasCIDR {
		t.Errorf("Expected CIDR 5.6.7.8/24, got: %v", ips)
	}
	if !hasV6 {
		t.Errorf("Expected IPv6 address, got: %v", ips)
	}
}

// -------------------------------------------------------
// RateLimiter
// -------------------------------------------------------

func TestRateLimiter_ZeroRate_NoBlock(t *testing.T) {
	rl := NewRateLimiter(0)
	start := time.Now()
	for i := 0; i < 10; i++ {
		rl.Wait()
	}
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Zero-rate limiter should not block, but took %v", elapsed)
	}
}

func TestRateLimiter_NegativeRate_NoBlock(t *testing.T) {
	rl := NewRateLimiter(-5)
	start := time.Now()
	rl.Wait()
	rl.Wait()
	elapsed := time.Since(start)
	if elapsed > 100*time.Millisecond {
		t.Errorf("Negative-rate limiter should not block, but took %v", elapsed)
	}
}

func TestRateLimiter_EnforcesRate(t *testing.T) {
	// 10 requests per second => 100ms between each
	rl := NewRateLimiter(10)
	start := time.Now()
	rl.Wait()
	rl.Wait()
	rl.Wait()
	elapsed := time.Since(start)
	// 3 calls should take at least 200ms (2 intervals of 100ms after the first)
	if elapsed < 180*time.Millisecond {
		t.Errorf("Expected at least ~200ms for 3 calls at 10 rps, got %v", elapsed)
	}
}

func TestNewRateLimiter_PositiveRate(t *testing.T) {
	rl := NewRateLimiter(2) // 2 req/s => 500ms interval
	if rl.interval != 500*time.Millisecond {
		t.Errorf("Expected interval 500ms, got %v", rl.interval)
	}
}

// -------------------------------------------------------
// Cache TTL
// -------------------------------------------------------

func TestLoadRDAPCache_EvictsExpiredEntries(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(dir, "rdap_cache.json")

	// Create a cache file with one fresh entry and one expired entry.
	fresh := time.Now().Add(-1 * time.Hour).Format(time.RFC3339)
	expired := time.Now().Add(-200 * time.Hour).Format(time.RFC3339) // older than 168h default

	cacheData := map[string]interface{}{
		"entries": map[string]interface{}{
			"1.1.1.1": map[string]interface{}{
				"rdap_name": "Fresh",
				"cached_at": fresh,
			},
			"2.2.2.2": map[string]interface{}{
				"rdap_name": "Expired",
				"cached_at": expired,
			},
		},
	}
	raw, _ := json.Marshal(cacheData)
	if err := os.WriteFile(cachePath, raw, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Override the cache path by creating an extractor that will load from
	// the build/data directory.  We simulate by placing the cache in the
	// expected location.
	buildDataDir := filepath.Join(dir, "build", "data")
	if err := os.MkdirAll(buildDataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	buildCachePath := filepath.Join(buildDataDir, "rdap_cache.json")
	if err := os.WriteFile(buildCachePath, raw, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Change to the temp directory so loadRDAPCache finds build/data/rdap_cache.json
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ext := newTestExtractor(t, dir)
	// default TTL is 168 hours
	cache := ext.loadRDAPCache()

	if _, ok := cache.Entries["1.1.1.1"]; !ok {
		t.Error("Fresh entry (1.1.1.1) should not have been evicted")
	}
	if _, ok := cache.Entries["2.2.2.2"]; ok {
		t.Error("Expired entry (2.2.2.2) should have been evicted")
	}
}

func TestCacheTTL_DefaultValue(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)
	// CacheTTLHours is 0 by default in test config => should default to 168h
	ttl := ext.cacheTTL()
	if ttl != 168*time.Hour {
		t.Errorf("Expected default TTL of 168h, got %v", ttl)
	}
}

func TestCacheTTL_CustomValue(t *testing.T) {
	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		LocalPath:     dir,
		ResultsDir:    filepath.Join(dir, "results"),
		CacheTTLHours: 48,
	}
	ext := NewExtractor(cfg, log)
	ttl := ext.cacheTTL()
	if ttl != 48*time.Hour {
		t.Errorf("Expected TTL of 48h, got %v", ttl)
	}
}

func TestCleanExpiredCache(t *testing.T) {
	dir := t.TempDir()
	buildDataDir := filepath.Join(dir, "build", "data")
	if err := os.MkdirAll(buildDataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	expired := time.Now().Add(-200 * time.Hour).Format(time.RFC3339)
	cacheData := map[string]interface{}{
		"entries": map[string]interface{}{
			"3.3.3.3": map[string]interface{}{
				"rdap_name": "OldEntry",
				"cached_at": expired,
			},
		},
	}
	raw, _ := json.Marshal(cacheData)
	buildCachePath := filepath.Join(buildDataDir, "rdap_cache.json")
	if err := os.WriteFile(buildCachePath, raw, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ext := newTestExtractor(t, dir)
	ext.CleanExpiredCache()

	// Reload and check that expired entries were removed.
	cache := ext.loadRDAPCache()
	if len(cache.Entries) != 0 {
		t.Errorf("Expected 0 entries after cleaning, got %d", len(cache.Entries))
	}
}

// -------------------------------------------------------
// BuildBaseRecords
// -------------------------------------------------------

func TestBuildBaseRecords(t *testing.T) {
	dir := t.TempDir()

	// Create a .nft file so mapIPsToScanners works.
	nft := filepath.Join(dir, "shodan.nft")
	if err := os.WriteFile(nft, []byte("10.0.0.1\n10.0.0.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	ext := newTestExtractor(t, dir)
	ips := []string{"10.0.0.1", "10.0.0.2"}
	records := ext.BuildBaseRecords(ips)

	if len(records) != 2 {
		t.Fatalf("Expected 2 records, got %d", len(records))
	}

	if records[0].IPOrCIDR != "10.0.0.1" {
		t.Errorf("First record IP: want %q, got %q", "10.0.0.1", records[0].IPOrCIDR)
	}
	if records[0].RiskLevel != "unknown" {
		t.Errorf("RiskLevel should be 'unknown', got %q", records[0].RiskLevel)
	}
	if records[1].ID != "scanner_2" {
		t.Errorf("Second record ID: want %q, got %q", "scanner_2", records[1].ID)
	}
}

// -------------------------------------------------------
// NewExtractor with RateLimiter integration
// -------------------------------------------------------

func TestNewExtractor_CreatesRateLimiter(t *testing.T) {
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		RepoURL:     "https://example.com/repo",
		LocalPath:   "/tmp/test",
		APIThrottle: 2.0, // 2 seconds between requests => 0.5 rps
	}

	ext := NewExtractor(cfg, log)
	if ext.rateLimiter == nil {
		t.Fatal("rateLimiter should be initialized")
	}
	// 0.5 rps => 2 second interval
	expected := 2 * time.Second
	if ext.rateLimiter.interval != expected {
		t.Errorf("Expected interval %v, got %v", expected, ext.rateLimiter.interval)
	}
}

func TestNewExtractor_ZeroThrottle_NoLimiting(t *testing.T) {
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		RepoURL:     "https://example.com/repo",
		LocalPath:   "/tmp/test",
		APIThrottle: 0,
	}

	ext := NewExtractor(cfg, log)
	if ext.rateLimiter == nil {
		t.Fatal("rateLimiter should be initialized even with zero throttle")
	}
	// Zero throttle => interval should be 0 (no limiting)
	if ext.rateLimiter.interval != 0 {
		t.Errorf("Expected interval 0 for zero throttle, got %v", ext.rateLimiter.interval)
	}
}

// -------------------------------------------------------
// IsIPProcessed with ProcessedIPSet (O(1) lookup)
// -------------------------------------------------------

func TestIsIPProcessed_WithSet(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	tracker := &models.RDAPProgressTracker{
		ProcessedIPs:   []string{"1.1.1.1", "2.2.2.2"},
		ProcessedIPSet: map[string]struct{}{"1.1.1.1": {}, "2.2.2.2": {}},
	}

	if !ext.IsIPProcessed("1.1.1.1", tracker) {
		t.Error("1.1.1.1 should be processed (via set)")
	}
	if ext.IsIPProcessed("3.3.3.3", tracker) {
		t.Error("3.3.3.3 should NOT be processed (via set)")
	}
}

// -------------------------------------------------------
// SaveProgressTracker / LoadProgressTracker / ClearProgressTracker
// -------------------------------------------------------

func TestSaveLoadProgressTracker_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	buildDataDir := filepath.Join(dir, "build", "data")
	if err := os.MkdirAll(buildDataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ext := newTestExtractor(t, dir)

	tracker := &models.RDAPProgressTracker{
		TotalRecords:     100,
		ProcessedRecords: 42,
		ProcessedIPs:     []string{"1.1.1.1", "2.2.2.2"},
		StartedAt:        time.Now().Format(time.RFC3339),
		Workers:          4,
		Throttle:         1.0,
		Completed:        false,
	}

	if err := ext.SaveProgressTracker(tracker); err != nil {
		t.Fatalf("SaveProgressTracker: %v", err)
	}

	loaded := ext.LoadProgressTracker()
	if loaded.TotalRecords != 100 {
		t.Errorf("TotalRecords: want 100, got %d", loaded.TotalRecords)
	}
	if loaded.ProcessedRecords != 42 {
		t.Errorf("ProcessedRecords: want 42, got %d", loaded.ProcessedRecords)
	}
	if len(loaded.ProcessedIPs) != 2 {
		t.Errorf("ProcessedIPs: want 2, got %d", len(loaded.ProcessedIPs))
	}
	if loaded.ProcessedIPSet == nil {
		t.Error("ProcessedIPSet should be built by LoadProgressTracker")
	}
	if _, ok := loaded.ProcessedIPSet["1.1.1.1"]; !ok {
		t.Error("ProcessedIPSet should contain 1.1.1.1")
	}
}

func TestClearProgressTracker(t *testing.T) {
	dir := t.TempDir()
	buildDataDir := filepath.Join(dir, "build", "data")
	if err := os.MkdirAll(buildDataDir, 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ext := newTestExtractor(t, dir)

	tracker := &models.RDAPProgressTracker{
		TotalRecords: 10,
		ProcessedIPs: []string{"1.1.1.1"},
	}
	if err := ext.SaveProgressTracker(tracker); err != nil {
		t.Fatalf("SaveProgressTracker: %v", err)
	}

	if err := ext.ClearProgressTracker(); err != nil {
		t.Fatalf("ClearProgressTracker: %v", err)
	}

	// After clearing, loading should return an empty tracker.
	loaded := ext.LoadProgressTracker()
	if len(loaded.ProcessedIPs) != 0 {
		t.Errorf("Expected 0 IPs after clear, got %d", len(loaded.ProcessedIPs))
	}
}

func TestLoadProgressTracker_MissingFile(t *testing.T) {
	dir := t.TempDir()

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	ext := newTestExtractor(t, dir)
	tracker := ext.LoadProgressTracker()

	if tracker == nil {
		t.Fatal("LoadProgressTracker should never return nil")
	}
	if len(tracker.ProcessedIPs) != 0 {
		t.Errorf("Expected empty ProcessedIPs, got %d", len(tracker.ProcessedIPs))
	}
	if tracker.ProcessedIPSet == nil {
		t.Error("ProcessedIPSet should be initialized even for missing file")
	}
}

// -------------------------------------------------------
// performRDAPFull with httptest
// -------------------------------------------------------

func TestPerformRDAPFull_AllRegistriesFail(t *testing.T) {
	// Server that always returns 404 for all requests.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		LocalPath:  dir,
		ResultsDir: filepath.Join(dir, "results"),
	}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{srv.URL + "/ip/"}
	ext.apiClient = &http.Client{Timeout: 2 * time.Second}

	data := &models.ScannerData{IPOrCIDR: "192.0.2.1"}
	err := ext.performRDAPFull("192.0.2.1", data)

	// Should return an error since all registries fail (timeout or connection error).
	if err == nil {
		t.Fatal("performRDAPFull should return error when all registries fail")
	}
	if !strings.Contains(err.Error(), "no RDAP registry responded") {
		t.Errorf("Expected 'no RDAP registry responded' error, got: %v", err)
	}
}

func TestEnrichWithAPIUsingCache_UpdatesCacheAfterEnrichment(t *testing.T) {
	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		LocalPath:  dir,
		ResultsDir: filepath.Join(dir, "results"),
	}
	ext := NewExtractor(cfg, log)
	// Very short timeout so external requests fail fast.
	ext.apiClient = &http.Client{Timeout: 100 * time.Millisecond}

	cache := &rdapCache{Entries: map[string]models.RDAPCacheEntry{}, Path: filepath.Join(dir, "cache.json")}
	data := &models.ScannerData{IPOrCIDR: "10.0.0.1"}

	_ = ext.enrichUsingCache(data, cache)

	// Even if enrichment fails, the cache should have been updated.
	if _, ok := cache.Entries["10.0.0.1"]; !ok {
		t.Error("Cache should have an entry for 10.0.0.1 after enrichment attempt")
	}
}

// -------------------------------------------------------
// enrichUsingCache (rdapCache)
// -------------------------------------------------------

func TestEnrichUsingCache_CacheHit(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{
			"10.0.0.1": {
				RDAPName:    "CachedName",
				CountryCode: "DE",
				CountryName: "Germany",
				ISP:         "CachedISP",
			},
		},
		Path: filepath.Join(dir, "cache.json"),
	}

	data := &models.ScannerData{IPOrCIDR: "10.0.0.1"}
	err := ext.enrichUsingCache(data, cache)
	if err != nil {
		t.Fatalf("enrichUsingCache: %v", err)
	}

	// Should have been populated from cache.
	if data.RDAPName != "CachedName" {
		t.Errorf("RDAPName: want %q, got %q", "CachedName", data.RDAPName)
	}
	if data.CountryCode != "DE" {
		t.Errorf("CountryCode: want %q, got %q", "DE", data.CountryCode)
	}
}

// -------------------------------------------------------
// ScannerDataToCSVRow / CSVHeaders
// -------------------------------------------------------

func TestCSVHeaders_Length(t *testing.T) {
	if len(models.CSVHeaders) != 35 {
		t.Errorf("Expected 35 CSV headers, got %d", len(models.CSVHeaders))
	}
}

func TestScannerDataToCSVRow_MatchesHeaders(t *testing.T) {
	now := time.Now()
	data := models.ScannerData{
		ID:          "test_1",
		IPOrCIDR:    "1.2.3.4",
		ScannerName: "shodan",
		ScannerType: models.ScannerTypeShodan,
		Tags:        []string{"a", "b"},
		LastSeen:    now,
		FirstSeen:   now,
		ExportDate:  now,
	}

	row := models.ScannerDataToCSVRow(data)
	if len(row) != len(models.CSVHeaders) {
		t.Errorf("Row length %d should match headers length %d", len(row), len(models.CSVHeaders))
	}
	if row[0] != "test_1" {
		t.Errorf("First column (ID): want %q, got %q", "test_1", row[0])
	}
	if row[1] != "1.2.3.4" {
		t.Errorf("Second column (IP): want %q, got %q", "1.2.3.4", row[1])
	}
}

// -------------------------------------------------------
// performRDAPFull with httptest (injectable endpoints)
// -------------------------------------------------------

func TestPerformRDAPFull_Success(t *testing.T) {
	rdapJSON := `{
		"name": "ACME-NET",
		"handle": "NET-192-0-2-0-1",
		"port43": "whois.arin.net",
		"startAddress": "192.0.2.0",
		"endAddress": "192.0.2.255",
		"ipVersion": "v4",
		"type": "DIRECT ALLOCATION",
		"parentHandle": "NET-192-0-0-0-0",
		"events": [
			{"eventAction": "registration", "eventDate": "2020-01-01T00:00:00Z"},
			{"eventAction": "last changed", "eventDate": "2023-06-15T12:00:00Z"}
		],
		"entities": [
			{
				"roles": ["abuse"],
				"vcardArray": ["vcard", [
					["fn", {}, "text", "ACME Abuse"],
					["email", {}, "text", "abuse@acme.example"]
				]]
			},
			{
				"roles": ["technical"],
				"vcardArray": ["vcard", [
					["email", {}, "text", "tech@acme.example"]
				]]
			}
		],
		"network": {
			"cidr0_cidrs": [{"v4prefix": "192.0.2.0", "length": 24}]
		}
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(rdapJSON))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{srv.URL + "/ip/"}

	data := &models.ScannerData{IPOrCIDR: "192.0.2.1"}
	err := ext.performRDAPFull("192.0.2.1", data)
	if err != nil {
		t.Fatalf("performRDAPFull: %v", err)
	}

	if data.RDAPName != "ACME-NET" {
		t.Errorf("RDAPName: want %q, got %q", "ACME-NET", data.RDAPName)
	}
	if data.RDAPHandle != "NET-192-0-2-0-1" {
		t.Errorf("RDAPHandle: want %q, got %q", "NET-192-0-2-0-1", data.RDAPHandle)
	}
	if data.StartAddress != "192.0.2.0" {
		t.Errorf("StartAddress: want %q, got %q", "192.0.2.0", data.StartAddress)
	}
	if data.EndAddress != "192.0.2.255" {
		t.Errorf("EndAddress: want %q, got %q", "192.0.2.255", data.EndAddress)
	}
	if data.IPVersion != "v4" {
		t.Errorf("IPVersion: want %q, got %q", "v4", data.IPVersion)
	}
	if data.RDAPType != "DIRECT ALLOCATION" {
		t.Errorf("RDAPType: want %q, got %q", "DIRECT ALLOCATION", data.RDAPType)
	}
	if data.ParentHandle != "NET-192-0-0-0-0" {
		t.Errorf("ParentHandle: want %q, got %q", "NET-192-0-0-0-0", data.ParentHandle)
	}
	if data.EventRegistration != "2020-01-01T00:00:00Z" {
		t.Errorf("EventRegistration: want %q, got %q", "2020-01-01T00:00:00Z", data.EventRegistration)
	}
	if data.EventLastChanged != "2023-06-15T12:00:00Z" {
		t.Errorf("EventLastChanged: want %q, got %q", "2023-06-15T12:00:00Z", data.EventLastChanged)
	}
	if data.RDAPCIDR != "192.0.2.0/24" {
		t.Errorf("RDAPCIDR: want %q, got %q", "192.0.2.0/24", data.RDAPCIDR)
	}
	if data.AbuseEmail != "abuse@acme.example" {
		t.Errorf("AbuseEmail: want %q, got %q", "abuse@acme.example", data.AbuseEmail)
	}
	if data.TechEmail != "tech@acme.example" {
		t.Errorf("TechEmail: want %q, got %q", "tech@acme.example", data.TechEmail)
	}
	if data.Organization != "ACME-NET" {
		t.Errorf("Organization: want %q, got %q", "ACME-NET", data.Organization)
	}
	if data.Registry != "whois.arin.net" {
		t.Errorf("Registry: want %q, got %q", "whois.arin.net", data.Registry)
	}
}

func TestPerformRDAPFull_InvalidJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("{not valid json"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{srv.URL + "/ip/"}

	data := &models.ScannerData{IPOrCIDR: "192.0.2.1"}
	err := ext.performRDAPFull("192.0.2.1", data)
	if err == nil {
		t.Fatal("performRDAPFull should return error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "no RDAP registry responded") {
		t.Errorf("Expected 'no RDAP registry responded', got: %v", err)
	}
}

func TestPerformRDAPFull_Server500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{srv.URL + "/ip/"}
	// Use a short timeout to speed up the retry cycle.
	ext.apiClient = &http.Client{Timeout: 1 * time.Second}

	data := &models.ScannerData{IPOrCIDR: "192.0.2.1"}
	err := ext.performRDAPFull("192.0.2.1", data)
	if err == nil {
		t.Fatal("performRDAPFull should return error for 500 responses")
	}
}

// -------------------------------------------------------
// performGeoLookupExtended with httptest
// -------------------------------------------------------

func TestPerformGeoLookupExtended_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status": "success",
			"countryCode": "FR",
			"country": "France",
			"isp": "OVH SAS",
			"as": "AS16276 OVH SAS",
			"reverse": "ns1.ovh.net"
		}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.geoBaseURL = srv.URL + "/json/"

	cc, country, isp, asStr, reverse := ext.performGeoLookupExtended("1.2.3.4")

	if cc != "FR" {
		t.Errorf("countryCode: want %q, got %q", "FR", cc)
	}
	if country != "France" {
		t.Errorf("country: want %q, got %q", "France", country)
	}
	if isp != "OVH SAS" {
		t.Errorf("isp: want %q, got %q", "OVH SAS", isp)
	}
	if asStr != "AS16276 OVH SAS" {
		t.Errorf("as: want %q, got %q", "AS16276 OVH SAS", asStr)
	}
	if reverse != "ns1.ovh.net" {
		t.Errorf("reverse: want %q, got %q", "ns1.ovh.net", reverse)
	}
}

func TestPerformGeoLookupExtended_FailStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "fail", "message": "reserved range"}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.geoBaseURL = srv.URL + "/json/"

	cc, country, isp, asStr, reverse := ext.performGeoLookupExtended("10.0.0.1")

	if cc != "" || country != "" || isp != "" || asStr != "" || reverse != "" {
		t.Errorf("Expected all empty for fail status, got cc=%q country=%q isp=%q as=%q reverse=%q",
			cc, country, isp, asStr, reverse)
	}
}

func TestPerformGeoLookupExtended_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.geoBaseURL = srv.URL + "/json/"
	ext.apiClient = &http.Client{Timeout: 1 * time.Second}

	cc, country, isp, asStr, reverse := ext.performGeoLookupExtended("1.2.3.4")

	if cc != "" || country != "" || isp != "" || asStr != "" || reverse != "" {
		t.Errorf("Expected all empty for server error, got cc=%q country=%q isp=%q as=%q reverse=%q",
			cc, country, isp, asStr, reverse)
	}
}

// -------------------------------------------------------
// GeoLookupContinent with httptest
// -------------------------------------------------------

func TestGeoLookupContinent_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"status": "success",
			"continent": "Europe",
			"continentCode": "EU",
			"country": "France",
			"countryCode": "FR"
		}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.geoBaseURL = srv.URL + "/json/"

	continent, continentCode, country, countryCode, err := ext.GeoLookupContinent("1.2.3.4")
	if err != nil {
		t.Fatalf("GeoLookupContinent: %v", err)
	}

	if continent != "Europe" {
		t.Errorf("continent: want %q, got %q", "Europe", continent)
	}
	if continentCode != "EU" {
		t.Errorf("continentCode: want %q, got %q", "EU", continentCode)
	}
	if country != "France" {
		t.Errorf("country: want %q, got %q", "France", country)
	}
	if countryCode != "FR" {
		t.Errorf("countryCode: want %q, got %q", "FR", countryCode)
	}
}

func TestGeoLookupContinent_FailStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "fail", "message": "reserved range"}`))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.geoBaseURL = srv.URL + "/json/"

	_, _, _, _, err := ext.GeoLookupContinent("10.0.0.1")
	if err == nil {
		t.Fatal("GeoLookupContinent should return error for fail status")
	}
}

func TestGeoLookupContinent_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir, ResultsDir: filepath.Join(dir, "results")}
	ext := NewExtractor(cfg, log)
	ext.geoBaseURL = srv.URL + "/json/"
	ext.apiClient = &http.Client{Timeout: 1 * time.Second}

	_, _, _, _, err := ext.GeoLookupContinent("1.2.3.4")
	if err == nil {
		t.Fatal("GeoLookupContinent should return error for server error")
	}
}

// -------------------------------------------------------
// httpGetWithRetry
// -------------------------------------------------------

func TestHttpGetWithRetry_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir}
	ext := NewExtractor(cfg, log)
	ext.apiClient = srv.Client()

	resp, err := ext.httpGetWithRetry(srv.URL)
	if err != nil {
		t.Fatalf("httpGetWithRetry: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("StatusCode: want 200, got %d", resp.StatusCode)
	}
}

func TestHttpGetWithRetry_RetriesOn5xx(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir}
	ext := NewExtractor(cfg, log)
	ext.apiClient = srv.Client()

	resp, err := ext.httpGetWithRetry(srv.URL)
	if err != nil {
		t.Fatalf("httpGetWithRetry should succeed after retries: %v", err)
	}
	resp.Body.Close()

	if attempts != 3 {
		t.Errorf("Expected 3 attempts (2 failures + 1 success), got %d", attempts)
	}
}

func TestHttpGetWithRetry_Returns4xxWithoutRetry(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir}
	ext := NewExtractor(cfg, log)
	ext.apiClient = srv.Client()

	resp, err := ext.httpGetWithRetry(srv.URL)
	if err != nil {
		t.Fatalf("httpGetWithRetry should not error for 404: %v", err)
	}
	resp.Body.Close()

	if attempts != 1 {
		t.Errorf("4xx should not be retried, expected 1 attempt, got %d", attempts)
	}
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("StatusCode: want 404, got %d", resp.StatusCode)
	}
}

func TestHttpGetWithRetry_RetriesOn429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts <= 1 {
			w.Header().Set("Retry-After", "0")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir}
	ext := NewExtractor(cfg, log)
	ext.apiClient = srv.Client()

	resp, err := ext.httpGetWithRetry(srv.URL)
	if err != nil {
		t.Fatalf("httpGetWithRetry should succeed after 429 retry: %v", err)
	}
	resp.Body.Close()

	if attempts != 2 {
		t.Errorf("Expected 2 attempts (1 x 429 + 1 success), got %d", attempts)
	}
}

func TestHttpGetWithRetry_AllRetriesFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{LocalPath: dir}
	ext := NewExtractor(cfg, log)
	ext.apiClient = srv.Client()

	_, err := ext.httpGetWithRetry(srv.URL)
	if err == nil {
		t.Fatal("httpGetWithRetry should return error when all retries fail")
	}
	if !strings.Contains(err.Error(), "after") {
		t.Errorf("Error should mention retries, got: %v", err)
	}
}

// -------------------------------------------------------
// enrichData with worker pool
// -------------------------------------------------------

func TestEnrichData_WorkerPool(t *testing.T) {
	// Set up RDAP and Geo mock servers.
	rdapSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name": "TestNet", "handle": "HANDLE-1", "startAddress": "10.0.0.0", "endAddress": "10.0.0.255"}`))
	}))
	defer rdapSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "countryCode": "US", "country": "United States", "isp": "TestISP", "as": "AS1234 TestAS", "reverse": "test.example.com"}`))
	}))
	defer geoSrv.Close()

	dir := t.TempDir()

	// Create .nft files.
	nft := filepath.Join(dir, "shodan.nft")
	if err := os.WriteFile(nft, []byte("10.0.0.1\n10.0.0.2\n10.0.0.3\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	// Change to temp dir so cache files are written there.
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		LocalPath:   dir,
		ResultsDir:  filepath.Join(dir, "results"),
		Parallelism: 2,
	}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{rdapSrv.URL + "/ip/"}
	ext.geoBaseURL = geoSrv.URL + "/json/"

	ips := []string{"10.0.0.1", "10.0.0.2", "10.0.0.3"}
	results, err := ext.enrichData(ips)
	if err != nil {
		t.Fatalf("enrichData: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("Expected 3 results, got %d", len(results))
	}

	for i, r := range results {
		if r.RDAPName != "TestNet" {
			t.Errorf("results[%d].RDAPName: want %q, got %q", i, "TestNet", r.RDAPName)
		}
		if r.CountryCode != "US" {
			t.Errorf("results[%d].CountryCode: want %q, got %q", i, "US", r.CountryCode)
		}
		if r.ISP != "TestISP" {
			t.Errorf("results[%d].ISP: want %q, got %q", i, "TestISP", r.ISP)
		}
		if r.ASN != "AS1234 TestAS" {
			t.Errorf("results[%d].ASN: want %q, got %q", i, "AS1234 TestAS", r.ASN)
		}
		if r.ASName != "TestAS" {
			t.Errorf("results[%d].ASName: want %q, got %q", i, "TestAS", r.ASName)
		}
	}
}

func TestEnrichData_SequentialMode(t *testing.T) {
	rdapSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name": "SeqNet"}`))
	}))
	defer rdapSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status": "success", "countryCode": "DE", "country": "Germany", "isp": "Hetzner", "as": "", "reverse": ""}`))
	}))
	defer geoSrv.Close()

	dir := t.TempDir()
	nft := filepath.Join(dir, "censys.nft")
	if err := os.WriteFile(nft, []byte("10.0.0.1\n10.0.0.2\n"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		LocalPath:   dir,
		ResultsDir:  filepath.Join(dir, "results"),
		Parallelism: 1, // Sequential.
	}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{rdapSrv.URL + "/ip/"}
	ext.geoBaseURL = geoSrv.URL + "/json/"

	results, err := ext.enrichData([]string{"10.0.0.1", "10.0.0.2"})
	if err != nil {
		t.Fatalf("enrichData: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("Expected 2 results, got %d", len(results))
	}

	for i, r := range results {
		if r.RDAPName != "SeqNet" {
			t.Errorf("results[%d].RDAPName: want %q, got %q", i, "SeqNet", r.RDAPName)
		}
		if r.CountryCode != "DE" {
			t.Errorf("results[%d].CountryCode: want %q, got %q", i, "DE", r.CountryCode)
		}
	}
}

// -------------------------------------------------------
// retryDelay / retryAfterDelay
// -------------------------------------------------------

func TestRetryDelay_ExponentialGrowth(t *testing.T) {
	d0 := retryDelay(0)
	d1 := retryDelay(1)
	d2 := retryDelay(2)

	// With jitter, d1 should be roughly 2x d0 (within tolerance).
	// Base: 500ms, 1000ms, 2000ms + up to 25% jitter.
	if d0 < 500*time.Millisecond || d0 > 625*time.Millisecond {
		t.Errorf("retryDelay(0) should be 500-625ms, got %v", d0)
	}
	if d1 < 1000*time.Millisecond || d1 > 1250*time.Millisecond {
		t.Errorf("retryDelay(1) should be 1000-1250ms, got %v", d1)
	}
	if d2 < 2000*time.Millisecond || d2 > 2500*time.Millisecond {
		t.Errorf("retryDelay(2) should be 2000-2500ms, got %v", d2)
	}
}

func TestRetryDelay_CapsAtMaxDelay(t *testing.T) {
	d := retryDelay(100) // Very high attempt.
	// Should be capped at retryMaxDelay (10s) + up to 25% jitter.
	if d > 12500*time.Millisecond {
		t.Errorf("retryDelay(100) should be capped, got %v", d)
	}
}

// -------------------------------------------------------
// safeRDAPCache (concurrent access)
// -------------------------------------------------------

func TestSafeRDAPCache_ConcurrentAccess(t *testing.T) {
	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{},
		Path:    filepath.Join(t.TempDir(), "cache.json"),
	}
	sc := newSafeRDAPCache(cache)

	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			ip := strings.Replace("10.0.0.X", "X", strings.Repeat("1", id%9+1), 1)
			data := &models.ScannerData{
				IPOrCIDR: ip,
				RDAPName: "Worker" + strings.Repeat("1", id%9+1),
			}
			sc.updateCache(ip, data)
			sc.applyCache(ip, &models.ScannerData{IPOrCIDR: ip})
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// No race condition, no panic = success.
}

// -------------------------------------------------------
// enrichUsingCache (safeRDAPCache)
// -------------------------------------------------------

func TestEnrichUsingCache_SafeCache_CacheHit(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{
			"10.0.0.1": {
				RDAPName:    "SafeCachedName",
				CountryCode: "JP",
				CountryName: "Japan",
			},
		},
		Path: filepath.Join(dir, "cache.json"),
	}
	sc := newSafeRDAPCache(cache)

	data := &models.ScannerData{IPOrCIDR: "10.0.0.1"}
	err := ext.enrichUsingCache(data, sc)
	if err != nil {
		t.Fatalf("enrichUsingCache via safeRDAPCache: %v", err)
	}

	if data.RDAPName != "SafeCachedName" {
		t.Errorf("RDAPName: want %q, got %q", "SafeCachedName", data.RDAPName)
	}
	if data.CountryCode != "JP" {
		t.Errorf("CountryCode: want %q, got %q", "JP", data.CountryCode)
	}
}

// -------------------------------------------------------
// retryAfterDelay edge cases
// -------------------------------------------------------

func TestRetryAfterDelay_SecondsHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{"5"}},
	}
	d := retryAfterDelay(resp)
	if d != 5*time.Second {
		t.Errorf("Expected 5s, got %v", d)
	}
}

func TestRetryAfterDelay_HTTPDateHeader(t *testing.T) {
	futureTime := time.Now().Add(10 * time.Second)
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{futureTime.UTC().Format(http.TimeFormat)}},
	}
	d := retryAfterDelay(resp)
	// Should be roughly 10 seconds (allow tolerance).
	if d < 8*time.Second || d > 12*time.Second {
		t.Errorf("Expected ~10s, got %v", d)
	}
}

func TestRetryAfterDelay_EmptyHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{},
	}
	d := retryAfterDelay(resp)
	if d != 0 {
		t.Errorf("Expected 0 for empty header, got %v", d)
	}
}

func TestRetryAfterDelay_InvalidHeader(t *testing.T) {
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{"garbage"}},
	}
	d := retryAfterDelay(resp)
	if d != 0 {
		t.Errorf("Expected 0 for garbage header, got %v", d)
	}
}

func TestRetryAfterDelay_PastHTTPDate(t *testing.T) {
	pastTime := time.Now().Add(-10 * time.Second)
	resp := &http.Response{
		Header: http.Header{"Retry-After": []string{pastTime.UTC().Format(http.TimeFormat)}},
	}
	d := retryAfterDelay(resp)
	// Past date should return 0 (negative duration is not useful).
	if d != 0 {
		t.Errorf("Expected 0 for past date, got %v", d)
	}
}

// -------------------------------------------------------
// SaveToJSON / SaveToCSV error paths
// -------------------------------------------------------

func TestSaveToJSON_ErrorOnInvalidDir(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)
	ext.config.ResultsDir = "/nonexistent/deeply/nested/path"

	err := ext.SaveToJSON([]models.ScannerData{{IPOrCIDR: "1.2.3.4"}}, "test.json")
	if err == nil {
		t.Error("Expected error for invalid results directory")
	}
}

func TestSaveToCSV_ErrorOnInvalidDir(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)
	ext.config.ResultsDir = "/nonexistent/deeply/nested/path"

	err := ext.SaveToCSV([]models.ScannerData{{IPOrCIDR: "1.2.3.4"}}, "test.csv")
	if err == nil {
		t.Error("Expected error for invalid results directory")
	}
}

// -------------------------------------------------------
// LoadFromJSON error paths
// -------------------------------------------------------

func TestLoadFromJSON_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	// Write invalid JSON to results dir.
	resultsDir := filepath.Join(dir, "results")
	_ = os.MkdirAll(resultsDir, 0755)
	ext.config.ResultsDir = resultsDir
	if err := os.WriteFile(filepath.Join(resultsDir, "bad.json"), []byte("{bad json"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := ext.LoadFromJSON("bad.json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestLoadFromJSON_FileNotFound(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)
	ext.config.ResultsDir = filepath.Join(dir, "results")

	_, err := ext.LoadFromJSON("nonexistent.json")
	if err == nil {
		t.Error("Expected error for missing file")
	}
}

// -------------------------------------------------------
// EnrichRecordWithDelay
// -------------------------------------------------------

func TestEnrichRecordWithDelay_RestoresThrottle(t *testing.T) {
	// Mock RDAP + geo servers.
	rdapSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"TestNet","handle":"NET-1"}`)
	}))
	defer rdapSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"success","countryCode":"US","country":"United States","isp":"TestISP","as":"AS123","reverse":"test.com"}`)
	}))
	defer geoSrv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	cfg := models.DatabaseConfig{
		LocalPath:   dir,
		ResultsDir:  filepath.Join(dir, "results"),
		APIThrottle: 0.5,
	}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{rdapSrv.URL + "/ip/"}
	ext.geoBaseURL = geoSrv.URL + "/"
	ext.apiClient = &http.Client{Timeout: 2 * time.Second}

	data := &models.ScannerData{IPOrCIDR: "10.0.0.1"}
	err := ext.EnrichRecordWithDelay(data, 100)
	if err != nil {
		t.Fatalf("EnrichRecordWithDelay: %v", err)
	}

	// Verify throttle was restored.
	if ext.config.APIThrottle != 0.5 {
		t.Errorf("APIThrottle should be restored to 0.5, got %f", ext.config.APIThrottle)
	}

	// Verify some enrichment happened.
	if data.CountryCode != "US" {
		t.Errorf("CountryCode: want %q, got %q", "US", data.CountryCode)
	}
}

// -------------------------------------------------------
// enrichWithAPI (single-record, loads+saves cache)
// -------------------------------------------------------

func TestEnrichWithAPI_EnrichesAndPersistsCache(t *testing.T) {
	rdapSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"name":"APINet","handle":"API-1"}`)
	}))
	defer rdapSrv.Close()

	geoSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"success","countryCode":"DE","country":"Germany","isp":"TestISP","as":"AS456 TestAS","reverse":"api.example.com"}`)
	}))
	defer geoSrv.Close()

	dir := t.TempDir()
	log := logger.NewLogger()
	// Set up build/data dir for cache file.
	buildDataDir := filepath.Join(dir, "build", "data")
	_ = os.MkdirAll(buildDataDir, 0755)

	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := models.DatabaseConfig{
		LocalPath:  dir,
		ResultsDir: filepath.Join(dir, "results"),
	}
	ext := NewExtractor(cfg, log)
	ext.rdapEndpoints = []string{rdapSrv.URL + "/ip/"}
	ext.geoBaseURL = geoSrv.URL + "/"
	ext.apiClient = &http.Client{Timeout: 2 * time.Second}

	data := &models.ScannerData{IPOrCIDR: "10.0.0.1"}
	err := ext.enrichWithAPI(data)
	if err != nil {
		t.Fatalf("enrichWithAPI: %v", err)
	}

	if data.RDAPName != "APINet" {
		t.Errorf("RDAPName: want %q, got %q", "APINet", data.RDAPName)
	}
	if data.CountryCode != "DE" {
		t.Errorf("CountryCode: want %q, got %q", "DE", data.CountryCode)
	}
	if data.ASName != "TestAS" {
		t.Errorf("ASName: want %q, got %q", "TestAS", data.ASName)
	}

	// Verify cache was persisted to disk.
	cacheData, err := os.ReadFile(filepath.Join(buildDataDir, "rdap_cache.json"))
	if err != nil {
		t.Fatalf("Cache file should exist: %v", err)
	}
	if !strings.Contains(string(cacheData), "10.0.0.1") {
		t.Error("Cache file should contain entry for 10.0.0.1")
	}
}

// -------------------------------------------------------
// cacheAccessor interface compliance
// -------------------------------------------------------

func TestCacheAccessor_InterfaceCompliance(t *testing.T) {
	// Verify both types implement cacheAccessor.
	var _ cacheAccessor = &rdapCache{Entries: map[string]models.RDAPCacheEntry{}}
	var _ cacheAccessor = newSafeRDAPCache(&rdapCache{Entries: map[string]models.RDAPCacheEntry{}})
}
