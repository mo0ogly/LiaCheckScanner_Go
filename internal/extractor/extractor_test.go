package extractor

import (
	"encoding/json"
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
// isTextFile
// -------------------------------------------------------

func TestIsTextFile(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	textPaths := []string{
		"file.txt", "README.md", "data.json", "config.yaml", "config.yml",
		"doc.xml", "results.csv", "settings.conf", "app.cfg", "setup.ini",
		"run.sh", "script.py", "app.js", "page.html", "style.css",
		"noextension", // no extension -> true
	}

	for _, p := range textPaths {
		if !ext.isTextFile(p) {
			t.Errorf("isTextFile(%q) should be true", p)
		}
	}

	binaryPaths := []string{
		"image.png", "photo.jpg", "archive.tar.gz", "binary.exe", "data.bin",
	}

	for _, p := range binaryPaths {
		if ext.isTextFile(p) {
			t.Errorf("isTextFile(%q) should be false", p)
		}
	}
}

// -------------------------------------------------------
// applyCache / updateCache
// -------------------------------------------------------

func TestApplyCache_Miss(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{},
	}
	data := &models.ScannerData{IPOrCIDR: "1.2.3.4"}

	applied := ext.applyCache("1.2.3.4", data, cache)
	if applied {
		t.Error("applyCache should return false for cache miss")
	}
}

func TestApplyCache_Hit(t *testing.T) {
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

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
	applied := ext.applyCache("1.2.3.4", data, cache)
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
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

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

	ext.updateCache("5.6.7.8", data, cache)

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
	dir := t.TempDir()
	ext := newTestExtractor(t, dir)

	cache := &rdapCache{
		Entries: map[string]models.RDAPCacheEntry{
			"1.1.1.1": {RDAPName: "OldName"},
		},
	}

	data := &models.ScannerData{
		IPOrCIDR: "1.1.1.1",
		RDAPName: "NewName",
	}

	ext.updateCache("1.1.1.1", data, cache)

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
