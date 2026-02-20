package extractor

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	if err := writer.Write(models.CSVHeaders); err != nil {
		return fmt.Errorf("writing CSV headers: %w", err)
	}

	for _, item := range data {
		if err := writer.Write(models.ScannerDataToCSVRow(item)); err != nil {
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
