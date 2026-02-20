package extractor

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// ScannerInfo holds the name, type, and source file of a scanner associated with an IP.
type ScannerInfo struct {
	Name       string
	Type       models.ScannerType
	SourceFile string
}

// mapIPsToScanners maps IPs to their scanner information based on .nft files.
func (e *Extractor) mapIPsToScanners(ips []string) map[string]ScannerInfo {
	ipToScanner := make(map[string]ScannerInfo)

	err := filepath.Walk(e.config.LocalPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".nft") {
			return nil
		}

		fileName := filepath.Base(path)
		scannerName := strings.TrimSuffix(fileName, ".nft")
		scannerType := e.getScannerType(scannerName)

		fileIPs, err := e.extractIPsFromNFTFile(path,
			regexp.MustCompile(`\b(?:[0-9]{1,3}\.){3}[0-9]{1,3}(?:/\d{1,2})?\b`),
			regexp.MustCompile(`(?:[a-fA-F0-9]{0,4}:){2,7}[a-fA-F0-9]{0,4}(?:/\d{1,3})?`))
		if err != nil {
			return nil
		}

		for _, ip := range fileIPs {
			ipToScanner[ip] = ScannerInfo{
				Name:       scannerName,
				Type:       scannerType,
				SourceFile: fileName,
			}
		}

		return nil
	})

	if err != nil {
		e.logger.Warning("Extractor", "Erreur lors du mapping des scanners: "+err.Error())
	}

	return ipToScanner
}

// getScannerType determines the scanner type based on the scanner name.
func (e *Extractor) getScannerType(scannerName string) models.ScannerType {
	switch strings.ToLower(scannerName) {
	case "shodan":
		return models.ScannerTypeShodan
	case "censys":
		return models.ScannerTypeCensys
	case "binaryedge":
		return models.ScannerTypeBinaryEdge
	case "rapid7":
		return models.ScannerTypeRapid7
	case "shadowserver":
		return models.ScannerTypeShadowServer
	default:
		return models.ScannerTypeOther
	}
}

// getCountryName returns the country name from a country code.
func (e *Extractor) getCountryName(code string) string {
	countries := map[string]string{
		"FR": "France",
		"US": "United States",
		"DE": "Germany",
		"GB": "United Kingdom",
		"CA": "Canada",
		"AU": "Australia",
		"JP": "Japan",
		"BR": "Brazil",
		"IN": "India",
		"RU": "Russia",
	}

	if name, exists := countries[code]; exists {
		return name
	}
	return "Unknown"
}

// getRiskLevel returns the risk level based on the abuse confidence score.
func (e *Extractor) getRiskLevel(score int) string {
	switch {
	case score >= 80:
		return "Critical"
	case score >= 60:
		return "High"
	case score >= 40:
		return "Medium"
	case score >= 20:
		return "Low"
	default:
		return "Very Low"
	}
}
