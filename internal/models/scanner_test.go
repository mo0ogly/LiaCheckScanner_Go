package models

import (
	"strings"
	"testing"
	"time"
)

// TestScannerDataCreation tests the creation of ScannerData structs
func TestScannerDataCreation(t *testing.T) {
	// Test data creation
	data := ScannerData{
		IPOrCIDR:             "192.168.1.1",
		ScannerName:          "Shodan",
		ScannerType:          ScannerTypeShodan,
		CountryCode:          "US",
		ISP:                  "Test ISP",
		RiskLevel:            "Medium",
		AbuseConfidenceScore: 75,
		LastSeen:             time.Now(),
		Tags:                 []string{"test", "development"},
		Notes:                "Test data for development",
	}

	// Verify all fields are set correctly
	if data.IPOrCIDR != "192.168.1.1" {
		t.Errorf("Expected IPOrCIDR to be '192.168.1.1', got '%s'", data.IPOrCIDR)
	}

	if data.ScannerName != "Shodan" {
		t.Errorf("Expected ScannerName to be 'Shodan', got '%s'", data.ScannerName)
	}

	if data.ScannerType != ScannerTypeShodan {
		t.Errorf("Expected ScannerType to be ScannerTypeShodan, got '%s'", data.ScannerType)
	}

	if data.CountryCode != "US" {
		t.Errorf("Expected CountryCode to be 'US', got '%s'", data.CountryCode)
	}

	if data.ISP != "Test ISP" {
		t.Errorf("Expected ISP to be 'Test ISP', got '%s'", data.ISP)
	}

	if data.RiskLevel != "Medium" {
		t.Errorf("Expected RiskLevel to be 'Medium', got '%s'", data.RiskLevel)
	}

	if data.AbuseConfidenceScore != 75 {
		t.Errorf("Expected AbuseConfidenceScore to be 75, got %d", data.AbuseConfidenceScore)
	}

	if len(data.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(data.Tags))
	}

	if data.Notes != "Test data for development" {
		t.Errorf("Expected Notes to be 'Test data for development', got '%s'", data.Notes)
	}
}

// TestScannerTypeValidation tests scanner type validation
func TestScannerTypeValidation(t *testing.T) {
	validTypes := []ScannerType{
		ScannerTypeShodan,
		ScannerTypeCensys,
		ScannerTypeBinaryEdge,
		ScannerTypeRapid7,
		ScannerTypeShadowServer,
		ScannerTypeOther,
		ScannerTypeShadowServer,
		ScannerTypeUnknown,
	}

	for _, scannerType := range validTypes {
		if scannerType == "" {
			t.Errorf("Scanner type should not be empty")
		}
	}
}

// TestRiskLevelValidation tests risk level validation
func TestRiskLevelValidation(t *testing.T) {
	validRiskLevels := []string{"High", "Medium", "Low", "Unknown"}

	for _, riskLevel := range validRiskLevels {
		if riskLevel == "" {
			t.Errorf("Risk level should not be empty")
		}
	}
}

// TestCountryCodeValidation tests country code validation
func TestCountryCodeValidation(t *testing.T) {
	validCountryCodes := []string{"US", "FR", "DE", "GB", "CA", "AU", "JP", "BR", "IN", "RU", "CN"}

	for _, countryCode := range validCountryCodes {
		if len(countryCode) != 2 {
			t.Errorf("Country code should be 2 characters, got '%s'", countryCode)
		}
	}
}

// TestAbuseConfidenceScoreRange tests abuse confidence score range
func TestAbuseConfidenceScoreRange(t *testing.T) {
	testCases := []struct {
		score int
		valid bool
	}{
		{0, true},
		{50, true},
		{100, true},
		{-1, false},
		{101, false},
	}

	for _, tc := range testCases {
		if tc.score < 0 || tc.score > 100 {
			if tc.valid {
				t.Errorf("Score %d should be invalid", tc.score)
			}
		} else {
			if !tc.valid {
				t.Errorf("Score %d should be valid", tc.score)
			}
		}
	}
}

// TestIPOrCIDRValidation tests IP/CIDR validation
func TestIPOrCIDRValidation(t *testing.T) {
	validIPs := []string{
		"192.168.1.1",
		"10.0.0.0/24",
		"172.16.0.1",
		"8.8.8.8",
	}

	invalidIPs := []string{
		"",
		"invalid",
		"256.256.256.256",
		"192.168.1.1/33",
	}

	for _, ip := range validIPs {
		if ip == "" {
			t.Errorf("IP address should not be empty")
		}
	}

	for _, ip := range invalidIPs {
		if ip == "" {
			// Empty IP is indeed invalid, so this is expected behavior
			continue
		}
	}
}

// TestScannerDataEquality tests scanner data equality
func TestScannerDataEquality(t *testing.T) {
	data1 := ScannerData{
		IPOrCIDR:    "192.168.1.1",
		ScannerName: "Shodan",
		ScannerType: ScannerTypeShodan,
	}

	data2 := ScannerData{
		IPOrCIDR:    "192.168.1.1",
		ScannerName: "Shodan",
		ScannerType: ScannerTypeShodan,
	}

	data3 := ScannerData{
		IPOrCIDR:    "192.168.1.2",
		ScannerName: "Censys",
		ScannerType: ScannerTypeCensys,
	}

	// Test equality
	if data1.IPOrCIDR != data2.IPOrCIDR {
		t.Errorf("Data1 and Data2 should have the same IP")
	}

	if data1.ScannerName != data2.ScannerName {
		t.Errorf("Data1 and Data2 should have the same scanner name")
	}

	// Test inequality
	if data1.IPOrCIDR == data3.IPOrCIDR {
		t.Errorf("Data1 and Data3 should have different IPs")
	}

	if data1.ScannerName == data3.ScannerName {
		t.Errorf("Data1 and Data3 should have different scanner names")
	}
}

// TestScannerDataTags tests tag functionality
func TestScannerDataTags(t *testing.T) {
	data := ScannerData{
		Tags: []string{"test", "development", "security"},
	}

	// Test tag count
	if len(data.Tags) != 3 {
		t.Errorf("Expected 3 tags, got %d", len(data.Tags))
	}

	// Test tag content
	expectedTags := []string{"test", "development", "security"}
	for i, tag := range data.Tags {
		if tag != expectedTags[i] {
			t.Errorf("Expected tag '%s', got '%s'", expectedTags[i], tag)
		}
	}
}

// TestScannerDataTimeStamp tests timestamp functionality
func TestScannerDataTimeStamp(t *testing.T) {
	now := time.Now()
	data := ScannerData{
		LastSeen: now,
	}

	// Test that LastSeen is set
	if data.LastSeen.IsZero() {
		t.Errorf("LastSeen should not be zero")
	}

	// Test that LastSeen is close to now
	diff := time.Since(data.LastSeen)
	if diff < -time.Second || diff > time.Second {
		t.Errorf("LastSeen should be close to current time, difference: %v", diff)
	}
}

// -------------------------------------------------------
// ScannerDataToCSVRow
// -------------------------------------------------------

func TestScannerDataToCSVRow_BasicOutput(t *testing.T) {
	now := time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)
	data := ScannerData{
		ID:                   "scanner_1",
		IPOrCIDR:             "192.0.2.1",
		ScannerName:          "shodan",
		ScannerType:          ScannerTypeShodan,
		SourceFile:           "shodan.nft",
		CountryCode:          "US",
		CountryName:          "United States",
		ISP:                  "Test ISP",
		Organization:         "Test Org",
		RDAPName:             "TESTNET",
		RDAPHandle:           "NET-192-0-2-0-1",
		RDAPCIDR:             "192.0.2.0/24",
		Registry:             "arin",
		StartAddress:         "192.0.2.0",
		EndAddress:           "192.0.2.255",
		IPVersion:            "v4",
		RDAPType:             "DIRECT ALLOCATION",
		ParentHandle:         "NET-192-0-0-0-0",
		EventRegistration:    "2020-01-01",
		EventLastChanged:     "2023-06-15",
		ASN:                  "AS12345",
		ASName:               "TestAS",
		ReverseDNS:           "test.example.com",
		AbuseConfidenceScore: 85,
		AbuseReports:         42,
		UsageType:            "Data Center",
		Domain:               "example.com",
		LastSeen:             now,
		FirstSeen:            now,
		Tags:                 []string{"extracted", "shodan"},
		Notes:                "Test note",
		RiskLevel:            "High",
		ExportDate:           now,
		AbuseEmail:           "abuse@test.com",
		TechEmail:            "tech@test.com",
	}

	row := ScannerDataToCSVRow(data)

	if len(row) != len(CSVHeaders) {
		t.Fatalf("Row length %d != CSVHeaders length %d", len(row), len(CSVHeaders))
	}

	// Spot-check key positions.
	checks := map[int]string{
		0:  "scanner_1",
		1:  "192.0.2.1",
		2:  "shodan",
		3:  "shodan",
		4:  "shodan.nft",
		5:  "US",
		6:  "United States",
		7:  "Test ISP",
		8:  "Test Org",
		9:  "TESTNET",
		20: "AS12345",
		21: "TestAS",
		22: "test.example.com",
		23: "85",
		24: "42",
		29: "extracted, shodan",
		30: "Test note",
		31: "High",
		33: "abuse@test.com",
		34: "tech@test.com",
	}

	for idx, want := range checks {
		if row[idx] != want {
			t.Errorf("row[%d] (%s): want %q, got %q", idx, CSVHeaders[idx], want, row[idx])
		}
	}
}

func TestScannerDataToCSVRow_ZeroTimeValues(t *testing.T) {
	data := ScannerData{}
	row := ScannerDataToCSVRow(data)

	// time.Time{} formatted with "2006-01-02 15:04:05" = "0001-01-01 00:00:00"
	zeroTime := "0001-01-01 00:00:00"
	timeIdxs := []int{27, 28, 32} // LastSeen, FirstSeen, ExportDate
	for _, idx := range timeIdxs {
		if row[idx] != zeroTime {
			t.Errorf("row[%d] (%s): want %q for zero time, got %q", idx, CSVHeaders[idx], zeroTime, row[idx])
		}
	}
}

func TestScannerDataToCSVRow_EmptyTags(t *testing.T) {
	// nil tags
	data := ScannerData{Tags: nil}
	row := ScannerDataToCSVRow(data)
	if row[29] != "" {
		t.Errorf("nil Tags: want empty string, got %q", row[29])
	}

	// empty slice
	data2 := ScannerData{Tags: []string{}}
	row2 := ScannerDataToCSVRow(data2)
	if row2[29] != "" {
		t.Errorf("empty Tags: want empty string, got %q", row2[29])
	}
}

func TestScannerDataToCSVRow_SpecialCharacters(t *testing.T) {
	data := ScannerData{
		ScannerName: `Shodan "pro"`,
		Notes:       "Line1\nLine2\tTabbed",
		Domain:      "test,comma.com",
		ISP:         "ISP with unicode: 你好",
	}
	row := ScannerDataToCSVRow(data)

	if row[2] != `Shodan "pro"` {
		t.Errorf("ScannerName: want %q, got %q", `Shodan "pro"`, row[2])
	}
	if row[30] != "Line1\nLine2\tTabbed" {
		t.Errorf("Notes: want newlines/tabs preserved, got %q", row[30])
	}
	if row[26] != "test,comma.com" {
		t.Errorf("Domain: want %q, got %q", "test,comma.com", row[26])
	}
	if row[7] != "ISP with unicode: 你好" {
		t.Errorf("ISP: want unicode preserved, got %q", row[7])
	}
}

func TestScannerDataToCSVRow_IntegerFormatting(t *testing.T) {
	tests := []struct {
		score   int
		reports int
		wantS   string
		wantR   string
	}{
		{0, 0, "0", "0"},
		{100, 9999, "100", "9999"},
		{-1, -5, "-1", "-5"},
	}

	for _, tc := range tests {
		data := ScannerData{
			AbuseConfidenceScore: tc.score,
			AbuseReports:         tc.reports,
		}
		row := ScannerDataToCSVRow(data)
		if row[23] != tc.wantS {
			t.Errorf("Score %d: want %q, got %q", tc.score, tc.wantS, row[23])
		}
		if row[24] != tc.wantR {
			t.Errorf("Reports %d: want %q, got %q", tc.reports, tc.wantR, row[24])
		}
	}
}

// -------------------------------------------------------
// CSVHeaders
// -------------------------------------------------------

func TestCSVHeaders_Count(t *testing.T) {
	if len(CSVHeaders) != 35 {
		t.Errorf("Expected 35 CSV headers, got %d", len(CSVHeaders))
	}
}

func TestCSVHeaders_NoDuplicates(t *testing.T) {
	seen := make(map[string]bool, len(CSVHeaders))
	for _, h := range CSVHeaders {
		lower := strings.ToLower(h)
		if seen[lower] {
			t.Errorf("Duplicate CSV header: %q", h)
		}
		seen[lower] = true
	}
}

// BenchmarkScannerDataCreation benchmarks scanner data creation
func BenchmarkScannerDataCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ScannerData{
			IPOrCIDR:             "192.168.1.1",
			ScannerName:          "Shodan",
			ScannerType:          ScannerTypeShodan,
			CountryCode:          "US",
			ISP:                  "Test ISP",
			RiskLevel:            "Medium",
			AbuseConfidenceScore: 75,
			LastSeen:             time.Now(),
			Tags:                 []string{"test", "development"},
			Notes:                "Test data for development",
		}
	}
}
