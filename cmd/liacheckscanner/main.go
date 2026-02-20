package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/lia/liacheckscanner_go/internal/config"
	"github.com/lia/liacheckscanner_go/internal/extractor"
	"github.com/lia/liacheckscanner_go/internal/gui"
	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

const (
	// Version is the current version of the LiaCheckScanner application.
	Version = "1.0.0"
	// AppName is the display name of the application.
	AppName = "LiaCheckScanner"
	// Owner is the author and contact information for the application.
	Owner = "LIA - mo0ogly@proton.me"
)

// createRequiredDirectories creates all necessary directories for the application
func createRequiredDirectories() error {
	// List of required directories
	dirs := []string{
		"logs",
		"results",
		"data",
		"config",
		"assets/icons",
		"build",
		"build/data",
		"internet-scanners",
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	return nil
}

func main() {
	// ----- CLI flags -----
	cliMode := flag.Bool("cli", false, "Run in headless CLI mode (no GUI)")
	outputFile := flag.String("output", "", "Output file path (CLI mode); defaults to stdout")
	outputFormat := flag.String("format", "csv", "Output format: csv or json (CLI mode)")
	enableRDAP := flag.Bool("rdap", false, "Enable RDAP enrichment (CLI mode)")
	flag.Parse()

	// Create required directories first
	if err := createRequiredDirectories(); err != nil {
		os.Stderr.WriteString("Error creating required directories: " + err.Error() + "\n")
		os.Exit(1)
	}

	// Initialiser le logger
	log := logger.NewLogger()
	log.Info("Main", "Starting "+AppName+" v"+Version)
	log.Info("Main", "Owner: "+Owner)
	log.Info("Main", "Directories created successfully")

	// Charger la configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Error("Main", "Error loading configuration: "+err.Error())
		os.Exit(1)
	}
	log.Info("Main", "Configuration loaded successfully")

	// ----- CLI mode -----
	if *cliMode {
		runCLI(cfg, log, *outputFile, *outputFormat, *enableRDAP)
		return
	}

	// ----- GUI mode (default) -----
	app := gui.NewApp(cfg, log)
	app.Run()

	log.Info("Main", AppName+" closed successfully")
}

// runCLI executes the headless CLI workflow: extract IPs, optionally enrich
// with RDAP, and write results to stdout or to a file.
func runCLI(cfg *models.AppConfig, log *logger.Logger, outputFile, outputFormat string, enableRDAP bool) {
	log.Info("CLI", "Running in CLI (headless) mode")

	ext := extractor.NewExtractor(cfg.Database, log)

	// --- Extract IPs from the internet-scanners repository ---
	log.Info("CLI", "Extracting IPs from repository...")
	ips, err := ext.ExtractIPsOnly()
	if err != nil {
		log.Error("CLI", "Extraction failed: "+err.Error())
		os.Exit(1)
	}
	log.Info("CLI", fmt.Sprintf("Extracted %d unique IPs", len(ips)))

	// Build base ScannerData records
	data := ext.BuildBaseRecords(ips)

	// --- Optional RDAP enrichment ---
	if enableRDAP {
		log.Info("CLI", "RDAP enrichment enabled, enriching records...")
		for i := range data {
			if err := ext.EnrichRecordWithDelay(&data[i], 0); err != nil {
				log.Warning("CLI", fmt.Sprintf("Enrichment error for %s: %v", data[i].IPOrCIDR, err))
			}
		}
		log.Info("CLI", fmt.Sprintf("Enrichment complete: %d records", len(data)))
	}

	// --- Output ---
	format := strings.ToLower(outputFormat)
	if format != "csv" && format != "json" {
		log.Error("CLI", "Unsupported format: "+outputFormat+". Use csv or json.")
		os.Exit(1)
	}

	if outputFile != "" {
		if format == "json" {
			if err := ext.SaveToJSON(data, outputFile); err != nil {
				log.Error("CLI", "Failed to write JSON: "+err.Error())
				os.Exit(1)
			}
		} else {
			if err := ext.SaveToCSV(data, outputFile); err != nil {
				log.Error("CLI", "Failed to write CSV: "+err.Error())
				os.Exit(1)
			}
		}
		log.Info("CLI", "Results written to "+outputFile)
	} else {
		// Write to stdout
		if format == "json" {
			encoder := json.NewEncoder(os.Stdout)
			encoder.SetIndent("", "  ")
			if err := encoder.Encode(data); err != nil {
				log.Error("CLI", "Failed to encode JSON to stdout: "+err.Error())
				os.Exit(1)
			}
		} else {
			writeCSVToStdout(data)
		}
	}

	log.Info("CLI", "CLI mode completed successfully")
}

// writeCSVToStdout writes scanner data as CSV to standard output.
func writeCSVToStdout(data []models.ScannerData) {
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()

	_ = w.Write(models.CSVHeaders)

	for _, item := range data {
		_ = w.Write(models.ScannerDataToCSVRow(item))
	}
}
