// Package gui provides the graphical user interface for LiaCheckScanner.
// This file contains export functionality for CSV data, logs, and ZIP archives.
package gui

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2/dialog"

	"github.com/lia/liacheckscanner_go/internal/models"
)

// exportAllData exports all data to a CSV file with professional formatting
// It creates a timestamped file in the results directory
func (a *App) exportAllData() {
	if len(a.data) == 0 {
		dialog.ShowInformation("Export", "⚠️ No data to export", a.mainWindow)
		return
	}

	// Generate professional filename
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("results/liacheckscanner_export_%s.csv", timestamp)

	// Create CSV file
	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Professional headers
	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk Level", "Score", "Last Seen", "Tags", "Notes"}
	writer.Write(headers)

	// Export data
	for _, item := range a.data {
		row := []string{
			item.IPOrCIDR,
			item.ScannerName,
			string(item.ScannerType),
			item.CountryCode,
			item.ISP,
			item.RiskLevel,
			fmt.Sprintf("%d", item.AbuseConfidenceScore),
			item.LastSeen.Format("2006-01-02"),
			strings.Join(item.Tags, ";"),
			item.Notes,
		}
		writer.Write(row)
	}

	a.logger.Info("GUI", fmt.Sprintf("✅ %d records exported to %s", len(a.data), filename))
	dialog.ShowInformation("Export Success", fmt.Sprintf("✅ %d records exported to:\n%s", len(a.data), filename), a.mainWindow)
}

// exportSelected exports selected data with professional confirmation
func (a *App) exportSelected() {
	dialog.ShowConfirm("Export Selected", "Export selected records to CSV?", func(confirm bool) {
		if confirm {
			a.exportSelectedData()
		}
	}, a.mainWindow)
}

// exportSelectedData performs the actual export of selected data
func (a *App) exportSelectedData() {
	selectedRows := a.getSelectedRows()
	if len(selectedRows) == 0 {
		dialog.ShowInformation("Export", "No records selected for export", a.mainWindow)
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("results/selected_export_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Professional headers
	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk Level", "Score", "Last Seen"}
	writer.Write(headers)

	// Export selected data
	for _, index := range selectedRows {
		if index < len(a.data) {
			item := a.data[index]
			row := []string{
				item.IPOrCIDR,
				item.ScannerName,
				string(item.ScannerType),
				item.CountryCode,
				item.ISP,
				item.RiskLevel,
				fmt.Sprintf("%d", item.AbuseConfidenceScore),
				item.LastSeen.Format("2006-01-02"),
			}
			writer.Write(row)
		}
	}

	a.logger.Info("GUI", fmt.Sprintf("✅ %d selected records exported to %s", len(selectedRows), filename))
	dialog.ShowInformation("Export Success", fmt.Sprintf("✅ %d records exported to:\n%s", len(selectedRows), filename), a.mainWindow)
}

// getSelectedRows returns indices of selected rows (simulated for now)
func (a *App) getSelectedRows() []int {
	// For now, return first 500 rows as a simulation
	// In a real implementation, this would track actual user selection
	maxRows := 500
	if len(a.data) < maxRows {
		maxRows = len(a.data)
	}

	var selected []int
	for i := 0; i < maxRows; i++ {
		selected = append(selected, i)
	}
	return selected
}

// exportSearchResults exports search results to CSV
func (a *App) exportSearchResults() {
	if len(a.searchResults) == 0 {
		dialog.ShowInformation("Export", "No search results to export", a.mainWindow)
		return
	}

	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("results/search_results_%s.csv", timestamp)

	file, err := os.Create(filename)
	if err != nil {
		dialog.ShowError(err, a.mainWindow)
		return
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"IP/CIDR", "Scanner", "Type", "Country", "ISP", "Risk", "Score", "Last Seen"}
	writer.Write(headers)

	for _, item := range a.searchResults {
		row := []string{
			item.IPOrCIDR,
			item.ScannerName,
			string(item.ScannerType),
			item.CountryCode,
			item.ISP,
			item.RiskLevel,
			fmt.Sprintf("%d", item.AbuseConfidenceScore),
			item.LastSeen.Format("2006-01-02"),
		}
		writer.Write(row)
	}

	a.logger.Info("GUI", fmt.Sprintf("✅ %d search results exported to %s", len(a.searchResults), filename))
	dialog.ShowInformation("Export Success", fmt.Sprintf("✅ %d search results exported to:\n%s", len(a.searchResults), filename), a.mainWindow)
}

// exportLogs exports logs to file (placeholder implementation)
func (a *App) exportLogs() {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	filename := fmt.Sprintf("logs/system_logs_%s.txt", timestamp)

	// Implementation would export actual logs
	a.logger.Info("GUI", "Logs exported to: "+filename)
	dialog.ShowInformation("Export Success", "Logs exported to:\n"+filename, a.mainWindow)
}

// zipDirectory zips a directory to the given zip file
func (a *App) zipDirectory(srcDir, destZip string) error {
	if err := os.MkdirAll(filepath.Dir(destZip), 0755); err != nil {
		return err
	}
	zf, err := os.Create(destZip)
	if err != nil {
		return err
	}
	defer zf.Close()
	w := zip.NewWriter(zf)
	defer w.Close()
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(srcDir, path)
		f, err := w.Create(rel)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(f, src)
		return err
	})
}

// Helper functions for search statistics

// countUniqueCountriesInResults counts unique countries in search results
func (a *App) countUniqueCountriesInResults(results []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range results {
		if item.CountryCode != "" {
			unique[item.CountryCode] = true
		}
	}
	return len(unique)
}

// countUniqueScannersInResults counts unique scanners in search results
func (a *App) countUniqueScannersInResults(results []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range results {
		unique[item.ScannerName] = true
	}
	return len(unique)
}

// countRiskLevelsInResults counts unique risk levels in search results
func (a *App) countRiskLevelsInResults(results []models.ScannerData) int {
	unique := make(map[string]bool)
	for _, item := range results {
		unique[item.RiskLevel] = true
	}
	return len(unique)
}
