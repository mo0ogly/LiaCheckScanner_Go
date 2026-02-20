// Package gui provides the graphical user interface for LiaCheckScanner.
// This file contains pure logic functions (no Fyne dependency) that can be
// unit-tested without a display server.
package gui

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// CountUniqueIPs returns the number of distinct IP/CIDR values in data.
func CountUniqueIPs(data []models.ScannerData) int {
	unique := make(map[string]bool, len(data))
	for _, item := range data {
		unique[item.IPOrCIDR] = true
	}
	return len(unique)
}

// CountUniqueCountries returns the number of distinct non-empty country codes.
func CountUniqueCountries(data []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range data {
		if item.CountryCode != "" {
			unique[item.CountryCode] = true
		}
	}
	return len(unique)
}

// CountUniqueScanners returns the number of distinct scanner names.
func CountUniqueScanners(data []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range data {
		unique[item.ScannerName] = true
	}
	return len(unique)
}

// CountHighRisk returns the number of entries with RiskLevel == "High".
func CountHighRisk(data []models.ScannerData) int {
	count := 0
	for _, item := range data {
		if item.RiskLevel == "High" {
			count++
		}
	}
	return count
}

// CountRiskLevels returns the number of distinct risk level values.
func CountRiskLevels(data []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range data {
		unique[item.RiskLevel] = true
	}
	return len(unique)
}

// FilterAdvancedSearch filters data by query string, country, scanner, and risk level.
// Filter values "All Countries", "All Scanners", "All Risk Levels" match everything.
// The query is matched case-insensitively against IPOrCIDR and ScannerName.
func FilterAdvancedSearch(data []models.ScannerData, query, country, scanner, risk string) []models.ScannerData {
	var results []models.ScannerData
	queryLower := strings.ToLower(query)

	for _, item := range data {
		matchesQuery := query == "" ||
			strings.Contains(strings.ToLower(item.IPOrCIDR), queryLower) ||
			strings.Contains(strings.ToLower(item.ScannerName), queryLower)

		matchesCountry := country == "All Countries" || item.CountryCode == country
		matchesScanner := scanner == "All Scanners" || item.ScannerName == scanner
		matchesRisk := risk == "All Risk Levels" || item.RiskLevel == risk

		if matchesQuery && matchesCountry && matchesScanner && matchesRisk {
			results = append(results, item)
		}
	}
	return results
}

// CalculatePagination computes pagination values from data length, items per page,
// and the requested current page. It returns totalPages, the clamped validPage,
// startIdx, and endIdx (exclusive).
func CalculatePagination(dataLen, itemsPerPage, currentPage int) (totalPages, validPage, startIdx, endIdx int) {
	if itemsPerPage <= 0 {
		itemsPerPage = 1
	}

	totalPages = (dataLen + itemsPerPage - 1) / itemsPerPage
	if totalPages == 0 {
		totalPages = 1
	}

	validPage = currentPage
	if validPage > totalPages {
		validPage = totalPages
	}
	if validPage < 1 {
		validPage = 1
	}

	startIdx = (validPage - 1) * itemsPerPage
	endIdx = startIdx + itemsPerPage
	if endIdx > dataLen {
		endIdx = dataLen
	}
	return
}

// LoadCSVData reads a CSV file with header-based column mapping and returns
// a slice of ScannerData. Returns an error if the file cannot be opened,
// parsed, or contains fewer than 2 rows (header + at least one data row).
func LoadCSVData(filename string) ([]models.ScannerData, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("insufficient data in CSV file")
	}

	// Build header index map
	headers := records[0]
	index := func(name string) int {
		for i, h := range headers {
			if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(name)) {
				return i
			}
		}
		return -1
	}

	ipIdx := index("IP/CIDR")
	scannerNameIdx := index("Scanner Name")
	scannerTypeIdx := index("Scanner Type")
	countryCodeIdx := index("Country Code")
	ispIdx := index("ISP")
	orgIdx := index("Organization")
	rdapNameIdx := index("RDAP Name")
	rdapHandleIdx := index("RDAP Handle")
	rdapCIDRIdx := index("RDAP CIDR")
	registryIdx := index("RDAP Registry")
	asnIdx := index("ASN")
	asNameIdx := index("AS Name")
	reverseIdx := index("Reverse DNS")
	riskIdx := index("Risk Level")
	scoreIdx := index("Abuse Confidence Score")
	domainIdx := index("Domain")
	lastSeenIdx := index("Last Seen")
	tagsIdx := index("Tags")
	notesIdx := index("Notes")
	parentHandleIdx := index("Parent Handle")
	eventRegIdx := index("Event Registration")
	eventChangedIdx := index("Event Last Changed")
	startAddrIdx := index("Start Address")
	endAddrIdx := index("End Address")
	ipVersionIdx := index("IP Version")
	rdapTypeIdx := index("RDAP Type")
	abuseEmailIdx := index("Abuse Email")
	techEmailIdx := index("Tech Email")

	var data []models.ScannerData
	for _, record := range records[1:] {
		item := models.ScannerData{}
		get := func(idx int) string {
			if idx >= 0 && idx < len(record) {
				return record[idx]
			}
			return ""
		}

		item.IPOrCIDR = get(ipIdx)
		item.ScannerName = get(scannerNameIdx)
		if v := get(scannerTypeIdx); v != "" {
			item.ScannerType = models.ScannerType(v)
		}
		item.CountryCode = get(countryCodeIdx)
		item.ISP = get(ispIdx)
		item.Organization = get(orgIdx)
		item.RDAPName = get(rdapNameIdx)
		item.RDAPHandle = get(rdapHandleIdx)
		item.RDAPCIDR = get(rdapCIDRIdx)
		item.Registry = get(registryIdx)
		item.StartAddress = get(startAddrIdx)
		item.EndAddress = get(endAddrIdx)
		item.IPVersion = get(ipVersionIdx)
		item.RDAPType = get(rdapTypeIdx)
		item.ParentHandle = get(parentHandleIdx)
		item.EventRegistration = get(eventRegIdx)
		item.EventLastChanged = get(eventChangedIdx)
		item.ASN = get(asnIdx)
		item.ASName = get(asNameIdx)
		item.ReverseDNS = get(reverseIdx)
		item.RiskLevel = get(riskIdx)
		if v := get(scoreIdx); v != "" {
			if score, err := strconv.Atoi(v); err == nil {
				item.AbuseConfidenceScore = score
			}
		}
		item.Domain = get(domainIdx)
		if v := get(lastSeenIdx); v != "" {
			if t, err := time.Parse("2006-01-02 15:04:05", v); err == nil {
				item.LastSeen = t
			} else {
				item.LastSeen = time.Now()
			}
		} else {
			item.LastSeen = time.Now()
		}
		if v := get(tagsIdx); v != "" {
			if ts := strings.TrimSpace(v); ts != "" {
				item.Tags = strings.Split(ts, ",")
			}
		}
		item.Notes = get(notesIdx)
		item.AbuseEmail = get(abuseEmailIdx)
		item.TechEmail = get(techEmailIdx)

		data = append(data, item)
	}

	return data, nil
}
