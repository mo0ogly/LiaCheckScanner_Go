package extractor

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

// newSilentLogger creates a logger with CRITICAL level so that benchmark
// iterations do not spend time on console I/O.
func newSilentLogger() *logger.Logger {
	l := logger.NewLogger()
	l.SetLogLevel(models.LogLevelCritical)
	return l
}

// ---------------------------------------------------------------------------
// Helpers: generate test data programmatically for benchmarks.
// ---------------------------------------------------------------------------

// generateNFTFileContent builds a .nft file body containing n unique IPv4 addresses.
func generateNFTFileContent(n int) string {
	var b []byte
	b = append(b, "# auto-generated benchmark data\n"...)
	b = append(b, "set bench_v4 {\n    type ipv4_addr\n    flags interval\n    elements = {\n"...)
	for i := 0; i < n; i++ {
		a := (i / (256 * 256)) % 256
		bc := (i / 256) % 256
		d := i % 256
		line := fmt.Sprintf("        %d.%d.%d.%d,\n", a+1, bc, d, (i*7)%256)
		b = append(b, line...)
	}
	b = append(b, "    }\n}\n"...)
	return string(b)
}

// generateScannerData builds a slice of n ScannerData entries.
func generateScannerData(n int) []models.ScannerData {
	now := time.Now()
	data := make([]models.ScannerData, n)
	for i := 0; i < n; i++ {
		data[i] = models.ScannerData{
			ID:          fmt.Sprintf("bench_%d", i),
			IPOrCIDR:    fmt.Sprintf("%d.%d.%d.%d", (i/16777216)%256, (i/65536)%256, (i/256)%256, i%256),
			ScannerName: "bench_scanner",
			ScannerType: models.ScannerTypeShodan,
			SourceFile:  "bench.nft",
			CountryCode: "US",
			CountryName: "United States",
			ISP:         "BenchISP",
			Organization: "BenchOrg",
			RiskLevel:   "Medium",
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			CreatedAt:   now,
			UpdatedAt:   now,
			Tags:        []string{"benchmark", "test"},
		}
	}
	return data
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

// BenchmarkExtractIPsFromNFTFile benchmarks parsing a single .nft file with 1000 IPs.
func BenchmarkExtractIPsFromNFTFile(b *testing.B) {
	dir := b.TempDir()
	nftFile := filepath.Join(dir, "bench.nft")
	content := generateNFTFileContent(1000)
	if err := os.WriteFile(nftFile, []byte(content), 0644); err != nil {
		b.Fatalf("WriteFile: %v", err)
	}

	ext := newBenchExtractor(b, dir)
	ipv4Re := regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`)
	ipv6Re := regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ext.extractIPsFromNFTFile(nftFile, ipv4Re, ipv6Re)
		if err != nil {
			b.Fatalf("extractIPsFromNFTFile: %v", err)
		}
	}
}

// BenchmarkParseFilesForIPs benchmarks walking a directory with 10 .nft files.
func BenchmarkParseFilesForIPs(b *testing.B) {
	dir := b.TempDir()

	// Create 10 .nft files, each with 100 IPs.
	for j := 0; j < 10; j++ {
		name := filepath.Join(dir, fmt.Sprintf("scanner_%d.nft", j))
		content := generateNFTFileContent(100)
		if err := os.WriteFile(name, []byte(content), 0644); err != nil {
			b.Fatalf("WriteFile: %v", err)
		}
	}

	ext := newBenchExtractor(b, dir)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := ext.parseFilesForIPs(dir)
		if err != nil {
			b.Fatalf("parseFilesForIPs: %v", err)
		}
	}
}

// BenchmarkGetScannerType benchmarks looking up the scanner type by name.
func BenchmarkGetScannerType(b *testing.B) {
	dir := b.TempDir()
	ext := newBenchExtractor(b, dir)

	names := []string{"shodan", "censys", "binaryedge", "rapid7", "shadowserver", "custom", "unknown"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ext.getScannerType(names[i%len(names)])
	}
}

// BenchmarkGetCountryName benchmarks looking up the country name by code.
func BenchmarkGetCountryName(b *testing.B) {
	dir := b.TempDir()
	ext := newBenchExtractor(b, dir)

	codes := []string{"FR", "US", "DE", "GB", "CA", "AU", "JP", "BR", "IN", "RU", "ZZ"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ext.getCountryName(codes[i%len(codes)])
	}
}

// BenchmarkGetRiskLevel benchmarks calculating the risk level from a score.
func BenchmarkGetRiskLevel(b *testing.B) {
	dir := b.TempDir()
	ext := newBenchExtractor(b, dir)

	scores := []int{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ext.getRiskLevel(scores[i%len(scores)])
	}
}

// BenchmarkApplyCache benchmarks cache-hit performance.
func BenchmarkApplyCache(b *testing.B) {
	dir := b.TempDir()
	ext := newBenchExtractor(b, dir)

	// Pre-populate a cache with 1000 entries.
	cache := &rdapCache{
		Entries: make(map[string]models.RDAPCacheEntry, 1000),
	}
	for i := 0; i < 1000; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", (i/16777216)%256, (i/65536)%256, (i/256)%256, i%256)
		cache.Entries[ip] = models.RDAPCacheEntry{
			RDAPName:    fmt.Sprintf("Net-%d", i),
			RDAPHandle:  fmt.Sprintf("HANDLE-%d", i),
			RDAPCIDR:    fmt.Sprintf("%d.0.0.0/8", i%256),
			Registry:    "arin",
			CountryCode: "US",
			CountryName: "United States",
			ISP:         "BenchISP",
			ASN:         fmt.Sprintf("AS%d", 10000+i),
			AbuseEmail:  "abuse@bench.com",
			TechEmail:   "tech@bench.com",
			CachedAt:    time.Now().Format(time.RFC3339),
		}
	}

	data := &models.ScannerData{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ip := fmt.Sprintf("%d.%d.%d.%d", (i%1000/16777216)%256, (i%1000/65536)%256, (i%1000/256)%256, i%1000%256)
		ext.applyCache(ip, data, cache)
	}
}

// BenchmarkSaveToJSON benchmarks JSON serialization of 1000 entries.
func BenchmarkSaveToJSON(b *testing.B) {
	dir := b.TempDir()
	ext := newBenchExtractor(b, dir)
	data := generateScannerData(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("bench_%d.json", i)
		if err := ext.SaveToJSON(data, filename); err != nil {
			b.Fatalf("SaveToJSON: %v", err)
		}
	}
}

// BenchmarkSaveToCSV benchmarks CSV serialization of 1000 entries.
func BenchmarkSaveToCSV(b *testing.B) {
	dir := b.TempDir()
	ext := newBenchExtractor(b, dir)
	data := generateScannerData(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filename := fmt.Sprintf("bench_%d.csv", i)
		if err := ext.SaveToCSV(data, filename); err != nil {
			b.Fatalf("SaveToCSV: %v", err)
		}
	}
}

// ---------------------------------------------------------------------------
// Helper: create an Extractor for benchmarks (uses testing.TB for both T and B).
// ---------------------------------------------------------------------------

func newBenchExtractor(tb testing.TB, localPath string) *Extractor {
	tb.Helper()
	log := newSilentLogger()
	cfg := models.DatabaseConfig{
		RepoURL:    "https://example.com/repo",
		LocalPath:  localPath,
		ResultsDir: filepath.Join(localPath, "results"),
		LogsDir:    filepath.Join(localPath, "logs"),
	}
	return NewExtractor(cfg, log)
}
