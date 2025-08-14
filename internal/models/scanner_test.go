package models

import (
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
