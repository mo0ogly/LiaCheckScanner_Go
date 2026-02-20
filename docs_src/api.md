# API Reference

This page documents the exported types and functions from each Go package. Since all application packages live under `internal/`, they are not importable by external modules, but this reference is useful for contributors working on the codebase.

---

## Package `models`

**Import path:** `github.com/lia/liacheckscanner_go/internal/models`

Defines all shared data structures used across the application.

### Types

#### `ScannerType`

```go
type ScannerType string
```

String enumeration for scanner classification.

| Constant                  | Value            |
|---------------------------|------------------|
| `ScannerTypeUnknown`      | `"unknown"`      |
| `ScannerTypeShodan`       | `"shodan"`       |
| `ScannerTypeCensys`       | `"censys"`       |
| `ScannerTypeBinaryEdge`   | `"binaryedge"`   |
| `ScannerTypeRapid7`       | `"rapid7"`       |
| `ScannerTypeShadowServer` | `"shadowserver"` |
| `ScannerTypeOther`        | `"other"`        |

#### `ScannerData`

```go
type ScannerData struct {
    ID                   string      `json:"id"`
    IPOrCIDR             string      `json:"ip_or_cidr"`
    ScannerName          string      `json:"scanner_name"`
    ScannerType          ScannerType `json:"scanner_type"`
    SourceFile           string      `json:"source_file"`
    CountryCode          string      `json:"country_code"`
    CountryName          string      `json:"country_name"`
    ISP                  string      `json:"isp"`
    Organization         string      `json:"organization"`
    AbuseConfidenceScore int         `json:"abuse_confidence_score"`
    AbuseReports         int         `json:"abuse_reports"`
    UsageType            string      `json:"usage_type"`
    Domain               string      `json:"domain"`
    RDAPName             string      `json:"rdap_name"`
    RDAPHandle           string      `json:"rdap_handle"`
    RDAPCIDR             string      `json:"rdap_cidr"`
    Registry             string      `json:"registry"`
    StartAddress         string      `json:"start_address"`
    EndAddress           string      `json:"end_address"`
    IPVersion            string      `json:"ip_version"`
    RDAPType             string      `json:"rdap_type"`
    ParentHandle         string      `json:"parent_handle"`
    EventRegistration    string      `json:"event_registration"`
    EventLastChanged     string      `json:"event_last_changed"`
    ASN                  string      `json:"asn"`
    ASName               string      `json:"as_name"`
    ReverseDNS           string      `json:"reverse_dns"`
    AbuseEmail           string      `json:"abuse_email"`
    TechEmail            string      `json:"tech_email"`
    LastSeen             time.Time   `json:"last_seen"`
    FirstSeen            time.Time   `json:"first_seen"`
    Tags                 []string    `json:"tags"`
    Notes                string      `json:"notes"`
    RiskLevel            string      `json:"risk_level"`
    ExportDate           time.Time   `json:"export_date"`
    CreatedAt            time.Time   `json:"created_at"`
    UpdatedAt            time.Time   `json:"updated_at"`
}
```

Primary data record representing a single enriched scanner IP. Each row in the CSV export and GUI table corresponds to one `ScannerData` instance.

#### `RDAPCacheEntry`

```go
type RDAPCacheEntry struct {
    RDAPName          string `json:"rdap_name"`
    RDAPHandle        string `json:"rdap_handle"`
    RDAPCIDR          string `json:"rdap_cidr"`
    Registry          string `json:"registry"`
    StartAddress      string `json:"start_address"`
    EndAddress        string `json:"end_address"`
    IPVersion         string `json:"ip_version"`
    RDAPType          string `json:"rdap_type"`
    ParentHandle      string `json:"parent_handle"`
    EventRegistration string `json:"event_registration"`
    EventLastChanged  string `json:"event_last_changed"`
    ASN               string `json:"asn"`
    ASName            string `json:"as_name"`
    ReverseDNS        string `json:"reverse_dns"`
    CountryCode       string `json:"country_code"`
    CountryName       string `json:"country_name"`
    ISP               string `json:"isp"`
    Organization      string `json:"organization"`
    AbuseEmail        string `json:"abuse_email"`
    TechEmail         string `json:"tech_email"`
    CachedAt          string `json:"cached_at"`
}
```

Persisted RDAP and geolocation results for a single IP. Stored in `build/data/rdap_cache.json`.

#### `RDAPProgressTracker`

```go
type RDAPProgressTracker struct {
    TotalRecords     int      `json:"total_records"`
    ProcessedRecords int      `json:"processed_records"`
    ProcessedIPs     []string `json:"processed_ips"`
    StartedAt        string   `json:"started_at"`
    LastUpdatedAt    string   `json:"last_updated_at"`
    Workers          int      `json:"workers"`
    Throttle         float64  `json:"throttle"`
    Completed        bool     `json:"completed"`
}
```

Tracks the state of a bulk RDAP enrichment operation. Saved to `build/data/rdap_progress.json` every 10 records.

#### `DatabaseConfig`

```go
type DatabaseConfig struct {
    RepoURL        string   `json:"repo_url"`
    LocalPath      string   `json:"local_path"`
    ResultsDir     string   `json:"results_dir"`
    LogsDir        string   `json:"logs_dir"`
    APIKey         string   `json:"api_key"`
    EnableAPI      bool     `json:"enable_api"`
    APIThrottle    float64  `json:"api_throttle"`
    Parallelism    int      `json:"parallelism"`
    Registries     []string `json:"registries"`
    AutoUpdate     bool     `json:"auto_update"`
    UpdateInterval int      `json:"update_interval"`
}
```

Configuration for repository access, export paths, and RDAP enrichment behavior.

#### `AppConfig`

```go
type AppConfig struct {
    AppName    string         `json:"app_name"`
    Version    string         `json:"version"`
    Owner      string         `json:"owner"`
    Theme      string         `json:"theme"`
    Language   string         `json:"language"`
    LogLevel   string         `json:"log_level"`
    MaxLogSize int            `json:"max_log_size"`
    LogBackups int            `json:"log_backups"`
    Database   DatabaseConfig `json:"database"`
}
```

Top-level application configuration. Serialized to/from `config/config.json`.

#### `SearchFilter`

```go
type SearchFilter struct {
    Query       string      `json:"query"`
    Type        string      `json:"type"`
    ScannerType ScannerType `json:"scanner_type"`
    Country     string      `json:"country"`
    ISP         string      `json:"isp"`
    RiskLevel   string      `json:"risk_level"`
    DateFrom    time.Time   `json:"date_from"`
    DateTo      time.Time   `json:"date_to"`
}
```

Criteria for advanced search filtering in the GUI.

#### `LogLevel`

```go
type LogLevel string
```

| Constant           | Value        |
|--------------------|--------------|
| `LogLevelDebug`    | `"DEBUG"`    |
| `LogLevelInfo`     | `"INFO"`     |
| `LogLevelWarning`  | `"WARNING"`  |
| `LogLevelError`    | `"ERROR"`    |
| `LogLevelCritical` | `"CRITICAL"` |

#### `LogEntry`

```go
type LogEntry struct {
    Timestamp time.Time              `json:"timestamp"`
    Level     LogLevel               `json:"level"`
    Component string                 `json:"component"`
    Message   string                 `json:"message"`
    Data      map[string]interface{} `json:"data,omitempty"`
}
```

A single structured log record.

---

## Package `config`

**Import path:** `github.com/lia/liacheckscanner_go/internal/config`

### Functions

#### `LoadConfig`

```go
func LoadConfig() (*models.AppConfig, error)
```

Convenience function that creates a `ConfigManager` and loads the configuration. Returns the loaded `AppConfig` or an error.

#### `NewConfigManager`

```go
func NewConfigManager() *ConfigManager
```

Creates a new `ConfigManager` pointing to `./config/config.json`. Creates the `config/` directory if it does not exist.

### Type `ConfigManager`

```go
type ConfigManager struct { /* unexported fields */ }
```

#### Methods

| Method                                                          | Description                                                        |
|-----------------------------------------------------------------|--------------------------------------------------------------------|
| `Load() (*models.AppConfig, error)`                             | Reads and parses the config file. Creates a default if missing.    |
| `Save(config *models.AppConfig) error`                          | Serializes the config to JSON and writes it to disk.               |
| `GetConfig() *models.AppConfig`                                 | Returns the currently loaded config (may be nil).                  |
| `UpdateDatabaseConfig(dbConfig models.DatabaseConfig) error`    | Updates the database section and saves.                            |
| `GetDatabaseConfig() models.DatabaseConfig`                     | Returns the database configuration section.                       |
| `SetAPIKey(apiKey string) error`                                | Sets the API key and saves.                                        |
| `GetAPIKey() string`                                            | Returns the current API key.                                       |
| `IsAPIEnabled() bool`                                           | Reports whether the API is enabled.                                |
| `EnableAPI() error`                                             | Enables the API and saves.                                         |
| `DisableAPI() error`                                            | Disables the API and saves.                                        |

---

## Package `extractor`

**Import path:** `github.com/lia/liacheckscanner_go/internal/extractor`

### Functions

#### `NewExtractor`

```go
func NewExtractor(config models.DatabaseConfig, logger *logger.Logger) *Extractor
```

Creates a new `Extractor` with the given configuration and logger. Initializes an HTTP client with a 30-second timeout.

### Type `Extractor`

```go
type Extractor struct { /* unexported fields */ }
```

#### Methods

| Method                                                                   | Description                                                                                            |
|--------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------|
| `ExtractData() ([]models.ScannerData, error)`                           | Full pipeline: clone/update repo, parse `.nft` files, enrich IPs, save to CSV. Returns all records.   |
| `SaveToJSON(data []models.ScannerData, filename string) error`           | Writes records to a JSON file in the results directory.                                                |
| `SaveToCSV(data []models.ScannerData, filename string) error`            | Writes records to a CSV file in the results directory.                                                 |
| `LoadFromJSON(filename string) ([]models.ScannerData, error)`            | Reads records from a JSON file in results or data directories.                                         |
| `EnrichRecordWithDelay(data *models.ScannerData, delayMs int) error`     | Enriches a single record via RDAP and geolocation with a custom delay.                                 |
| `GeoLookupContinent(ip string) (string, string, string, string, error)`  | Returns continent, continent code, country, and country code for an IP.                                |
| `LoadProgressTracker() *models.RDAPProgressTracker`                      | Loads the RDAP progress file from disk (returns empty tracker if missing).                             |
| `SaveProgressTracker(tracker *models.RDAPProgressTracker) error`         | Saves the progress tracker to disk.                                                                    |
| `IsIPProcessed(ip string, tracker *models.RDAPProgressTracker) bool`     | Checks whether an IP has already been processed in the given tracker.                                  |
| `ClearProgressTracker() error`                                           | Deletes the progress file from disk.                                                                   |

### Type `ScannerInfo`

```go
type ScannerInfo struct {
    Name       string
    Type       models.ScannerType
    SourceFile string
}
```

Associates an IP address with its scanner source file metadata.

---

## Package `logger`

**Import path:** `github.com/lia/liacheckscanner_go/internal/logger`

### Functions

#### `NewLogger`

```go
func NewLogger() *Logger
```

Creates a new logger. Initializes the `logs/` directory and opens a daily log file.

### Type `Logger`

```go
type Logger struct { /* unexported fields */ }
```

Thread-safe, leveled logger with file rotation.

#### Methods

| Method                                                              | Description                                                      |
|---------------------------------------------------------------------|------------------------------------------------------------------|
| `SetLogLevel(level models.LogLevel)`                                | Sets the minimum log level.                                      |
| `GetLogLevel() models.LogLevel`                                     | Returns the current minimum log level.                           |
| `Debug(component, message string, data ...map[string]interface{})`  | Logs at DEBUG level.                                             |
| `Info(component, message string, data ...map[string]interface{})`   | Logs at INFO level.                                              |
| `Warning(component, message string, data ...map[string]interface{})` | Logs at WARNING level.                                          |
| `Error(component, message string, data ...map[string]interface{})`  | Logs at ERROR level.                                             |
| `Critical(component, message string, data ...map[string]interface{})` | Logs at CRITICAL level.                                        |
| `GetEntries() []models.LogEntry`                                    | Returns a copy of all in-memory log entries.                     |
| `GetRecentEntries(count int) []models.LogEntry`                     | Returns the last `count` entries.                                |
| `ClearEntries()`                                                    | Clears the in-memory entry buffer.                               |
| `Close() error`                                                     | Closes the log file.                                             |

---

## Package `gui`

**Import path:** `github.com/lia/liacheckscanner_go/internal/gui`

### Functions

#### `NewApp`

```go
func NewApp(config *models.AppConfig, logger *logger.Logger) *App
```

Creates and initializes the Fyne application, sets up the window (1600x1000), creates the extractor, and builds the full UI.

### Type `App`

```go
type App struct { /* unexported fields */ }
```

Main application struct. Owns the Fyne app, all widgets, data, and pagination state.

#### Methods

| Method       | Description                                              |
|--------------|----------------------------------------------------------|
| `Run()`      | Enters the Fyne main event loop (blocks until quit).     |
| `Shutdown()` | Logs shutdown and calls `fyne.App.Quit()`.               |

---

## Package `main`

**Import path:** `github.com/lia/liacheckscanner_go/cmd/liacheckscanner`

### Constants

| Constant  | Value                        |
|-----------|------------------------------|
| `Version` | `"1.0.0"`                    |
| `AppName` | `"LiaCheckScanner"`          |
| `Owner`   | `"LIA - mo0ogly@proton.me"`  |
