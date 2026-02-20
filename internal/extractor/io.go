package extractor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// SaveToJSON writes the scanner data to a JSON file in the configured results directory.
func (e *Extractor) SaveToJSON(data []models.ScannerData, filename string) error {
	e.logger.Info("Extractor", "Sauvegarde en JSON...")

	if err := os.MkdirAll(e.config.ResultsDir, 0755); err != nil {
		return fmt.Errorf("creating results directory: %w", err)
	}

	filePath := filepath.Join(e.config.ResultsDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating JSON file %s: %w", filePath, err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("encoding JSON data: %w", err)
	}

	e.logger.Info("Extractor", fmt.Sprintf("Donnees sauvegardees: %s", filePath))
	return nil
}

// SaveToCSV writes the scanner data to a CSV file in the configured results directory.
func (e *Extractor) SaveToCSV(data []models.ScannerData, filename string) error {
	e.logger.Info("Extractor", "Sauvegarde en CSV...")

	if err := os.MkdirAll(e.config.ResultsDir, 0755); err != nil {
		return fmt.Errorf("creating results directory: %w", err)
	}

	filePath := filepath.Join(e.config.ResultsDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("creating CSV file %s: %w", filePath, err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{
		"ID", "IP/CIDR", "Scanner Name", "Scanner Type", "Source File",
		"Country Code", "Country Name", "ISP", "Organization",
		"RDAP Name", "RDAP Handle", "RDAP CIDR", "RDAP Registry",
		"Start Address", "End Address", "IP Version", "RDAP Type", "Parent Handle",
		"Event Registration", "Event Last Changed",
		"ASN", "AS Name", "Reverse DNS",
		"Abuse Confidence Score", "Abuse Reports", "Usage Type",
		"Domain", "Last Seen", "First Seen", "Tags", "Notes",
		"Risk Level", "Export Date", "Abuse Email", "Tech Email",
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("writing CSV headers: %w", err)
	}

	for _, item := range data {
		row := []string{
			item.ID,
			item.IPOrCIDR,
			item.ScannerName,
			string(item.ScannerType),
			item.SourceFile,
			item.CountryCode,
			item.CountryName,
			item.ISP,
			item.Organization,
			item.RDAPName,
			item.RDAPHandle,
			item.RDAPCIDR,
			item.Registry,
			item.StartAddress,
			item.EndAddress,
			item.IPVersion,
			item.RDAPType,
			item.ParentHandle,
			item.EventRegistration,
			item.EventLastChanged,
			item.ASN,
			item.ASName,
			item.ReverseDNS,
			fmt.Sprintf("%d", item.AbuseConfidenceScore),
			fmt.Sprintf("%d", item.AbuseReports),
			item.UsageType,
			item.Domain,
			item.LastSeen.Format("2006-01-02 15:04:05"),
			item.FirstSeen.Format("2006-01-02 15:04:05"),
			strings.Join(item.Tags, ", "),
			item.Notes,
			item.RiskLevel,
			item.ExportDate.Format("2006-01-02 15:04:05"),
			item.AbuseEmail,
			item.TechEmail,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("writing CSV row for %s: %w", item.ID, err)
		}
	}

	e.logger.Info("Extractor", fmt.Sprintf("Donnees sauvegardees: %s", filePath))
	return nil
}

// LoadFromJSON reads scanner data from a JSON file, searching in the results and data directories.
func (e *Extractor) LoadFromJSON(filename string) ([]models.ScannerData, error) {
	filePath := filepath.Join(e.config.ResultsDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		filePath = filepath.Join("data", filename)
		file, err = os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("aucun fichier de donnees trouve")
		}
	}
	defer file.Close()

	var data []models.ScannerData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		e.logger.Warning("Extractor", "Erreur lors du decodage JSON")
		return nil, fmt.Errorf("erreur lors du decodage JSON: %w", err)
	}

	return data, nil
}
