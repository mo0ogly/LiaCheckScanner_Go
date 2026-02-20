package extractor

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/lia/liacheckscanner_go/internal/logger"
	"github.com/lia/liacheckscanner_go/internal/models"
)

// Extractor handles data extraction from scanner repositories and enrichment via RDAP and geolocation APIs.
type Extractor struct {
	logger      *logger.Logger
	config      models.DatabaseConfig
	apiClient   *http.Client
	rateLimiter *RateLimiter

	// rdapEndpoints overrides the default RDAP registry URLs (for testing).
	rdapEndpoints []string
	// geoBaseURL overrides the default ip-api.com base URL (for testing).
	geoBaseURL string
}

// NewExtractor creates a new Extractor with the given database configuration and logger.
func NewExtractor(config models.DatabaseConfig, logger *logger.Logger) *Extractor {
	// Build a rate limiter from APIThrottle.  APIThrottle is expressed as
	// seconds between requests (e.g. 1 means 1 req/s, 0.5 means 2 req/s).
	var rps float64
	if config.APIThrottle > 0 {
		rps = 1.0 / config.APIThrottle
	}
	return &Extractor{
		logger: logger,
		config: config,
		apiClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		rateLimiter: NewRateLimiter(rps),
	}
}

// ExtractData clones or updates the configured repository, parses .nft files for IPs, enriches the results, and saves them to CSV.
func (e *Extractor) ExtractData() ([]models.ScannerData, error) {
	e.logger.Info("Extractor", "Debut de l'extraction des donnees")

	if err := e.cloneOrUpdateRepo(); err != nil {
		return nil, err
	}

	localPath := e.config.LocalPath
	if localPath == "" {
		localPath = "./data/internet-scanners"
	}

	scanners, err := e.parseFilesForIPs(localPath)
	if err != nil {
		e.logger.Error("Extractor", "Erreur lors du parsing: "+err.Error())
		return nil, fmt.Errorf("parse failed: %w", err)
	}

	if len(scanners) == 0 {
		e.logger.Error("Extractor", "Aucune IP trouvee")
		return nil, fmt.Errorf("no IPs found in repository")
	}

	e.logger.Info("Extractor", fmt.Sprintf("%d IPs uniques extraites au total", len(scanners)))

	e.logger.Info("Extractor", "Enrichissement des donnees...")
	enrichedData, err := e.enrichData(scanners)
	if err != nil {
		e.logger.Error("Extractor", "Erreur lors de l'enrichissement: "+err.Error())
		return nil, fmt.Errorf("enrichment failed: %w", err)
	}
	e.logger.Info("Extractor", fmt.Sprintf("%d enregistrements enrichis", len(enrichedData)))

	ts := time.Now().Format("2006-01-02_15-04-05")
	csvName := fmt.Sprintf("%s_liacheckscanner.csv", ts)
	if err := e.SaveToCSV(enrichedData, csvName); err != nil {
		e.logger.Warning("Extractor", "Erreur lors de la sauvegarde CSV: "+err.Error())
	} else {
		e.logger.Info("Extractor", "Sauvegarde en CSV...")
	}

	e.logger.Info("Extractor", fmt.Sprintf("Extraction terminee: %d enregistrements", len(enrichedData)))
	return enrichedData, nil
}

// ExtractIPsOnly clones or updates the repository and parses .nft files,
// returning only the unique IP list without performing any enrichment.
func (e *Extractor) ExtractIPsOnly() ([]string, error) {
	if err := e.cloneOrUpdateRepo(); err != nil {
		return nil, err
	}

	localPath := e.config.LocalPath
	if localPath == "" {
		localPath = "./data/internet-scanners"
	}

	return e.parseFilesForIPs(localPath)
}

// BuildBaseRecords creates ScannerData records from the given IP list,
// mapping each IP to its scanner source but without RDAP enrichment.
func (e *Extractor) BuildBaseRecords(ips []string) []models.ScannerData {
	ipToScanner := e.mapIPsToScanners(ips)
	now := time.Now()
	var records []models.ScannerData
	for i, ip := range ips {
		info := ipToScanner[ip]
		records = append(records, models.ScannerData{
			ID:          fmt.Sprintf("scanner_%d", i+1),
			IPOrCIDR:    ip,
			ScannerName: info.Name,
			ScannerType: info.Type,
			SourceFile:  info.SourceFile,
			LastSeen:    now,
			FirstSeen:   now,
			ExportDate:  now,
			CreatedAt:   now,
			UpdatedAt:   now,
			Tags:        []string{"extracted", info.Name},
			RiskLevel:   "unknown",
		})
	}
	return records
}

// cloneOrUpdateRepo clones or updates the configured repository.
func (e *Extractor) cloneOrUpdateRepo() error {
	repoURL := e.config.RepoURL
	if repoURL == "" {
		repoURL = "https://github.com/MDMCK10/internet-scanners"
	}
	localPath := e.config.LocalPath
	if localPath == "" {
		localPath = "./data/internet-scanners"
	}

	e.logger.Info("Extractor", "Clonage/mise a jour du repository...")
	e.logger.Info("Extractor", "Repository: "+repoURL)
	e.logger.Info("Extractor", "Local Path: "+localPath)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		parentDir := filepath.Dir(localPath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("cloneOrUpdateRepo: creating parent directory: %w", err)
		}
		e.logger.Info("Extractor", "Clonage du repository depuis "+repoURL)
		cmd := exec.Command("git", "clone", repoURL, localPath)
		if err := cmd.Run(); err != nil {
			e.logger.Error("Extractor", "Erreur lors du clonage: "+err.Error())
			return fmt.Errorf("git clone failed: %w", err)
		}
	} else {
		e.logger.Info("Extractor", "Repository local trouve, mise a jour...")
		cmd := exec.Command("git", "-C", localPath, "pull")
		if err := cmd.Run(); err != nil {
			e.logger.Error("Extractor", "Erreur lors de la mise a jour: "+err.Error())
			return fmt.Errorf("git pull failed: %w", err)
		}
	}

	e.logger.Info("Extractor", "Repository synchronise")
	return nil
}

// EnrichRecordWithDelay enriches a single scanner record, applying the specified delay in milliseconds.
func (e *Extractor) EnrichRecordWithDelay(data *models.ScannerData, delayMs int) error {
	prev := e.config.APIThrottle
	if delayMs >= 0 {
		e.config.APIThrottle = float64(delayMs) / 1000.0
	}
	defer func() { e.config.APIThrottle = prev }()
	return e.enrichWithAPI(data)
}
