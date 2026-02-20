# Architecture

## Project structure

```
LiaCheckScanner_Go/
├── cmd/
│   └── liacheckscanner/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   ├── config.go            # Configuration loading, saving, and management
│   │   └── config_test.go
│   ├── extractor/
│   │   ├── extractor.go         # IP extraction, RDAP enrichment, CSV/JSON I/O
│   │   └── extractor_test.go
│   ├── gui/
│   │   └── app.go               # Fyne GUI: tabs, table, pagination, search
│   ├── logger/
│   │   ├── logger.go            # Structured logging with rotation
│   │   └── logger_test.go
│   └── models/
│       ├── scanner.go           # Data types: ScannerData, AppConfig, etc.
│       └── scanner_test.go
├── config/
│   └── config.json              # Runtime configuration (auto-generated)
├── build/                       # Compiled binaries and cache files
├── results/                     # CSV exports
├── logs/                        # Application logs
├── data/                        # Cloned repositories and working data
├── docs_src/                    # MkDocs documentation source
├── Dockerfile                   # Multi-stage Docker build
├── Makefile                     # Build, test, and utility targets
├── mkdocs.yml                   # MkDocs configuration
├── go.mod
├── go.sum
├── README.md
├── CONTRIBUTING.md
├── CHANGELOG.md
└── LICENSE
```

## Package descriptions

### `cmd/liacheckscanner`

The `main` package. It performs three tasks:

1. Creates required directories on disk.
2. Initializes the logger and loads configuration via the `config` package.
3. Creates and runs the GUI via the `gui` package.

### `internal/config`

Manages the `config/config.json` file. The `ConfigManager` type provides methods to load, save, and update individual configuration sections (database settings, API key, etc.). If no configuration file exists, a default one is written on first load.

### `internal/extractor`

The core data-processing package. Responsibilities:

- **Repository management** -- clones or pulls the internet-scanners Git repository.
- **IP parsing** -- walks `.nft` files, extracts IPv4 and IPv6 addresses using regular expressions, and deduplicates them.
- **RDAP enrichment** -- queries all five Regional Internet Registries (ARIN, RIPE, APNIC, LACNIC, AFRINIC) for network, entity, and contact information.
- **Geolocation** -- calls ip-api.com for country, ISP, ASN, and reverse DNS data.
- **Caching** -- stores RDAP/geo results in `build/data/rdap_cache.json` to avoid repeated lookups.
- **Progress tracking** -- saves enrichment progress to `build/data/rdap_progress.json` so interrupted runs can be resumed.
- **Export** -- writes results to CSV and JSON files.

### `internal/gui`

Builds the Fyne-based graphical interface. The `App` struct owns the Fyne application, all UI widgets, data state, and pagination logic. It exposes five tabs:

| Tab           | Purpose                                              |
|---------------|------------------------------------------------------|
| Dashboard     | Statistics overview and quick actions                |
| Database      | Paginated data table with RDAP enrichment controls   |
| Search        | Advanced filtering and single-IP enrichment          |
| Configuration | Edit and save application settings                   |
| Logs          | View, filter, and export application logs            |

### `internal/logger`

Provides a thread-safe, leveled logging system. Log entries are:

- Printed to stdout with emoji-prefixed formatting.
- Written as JSON lines to a daily log file.
- Stored in memory (up to 1000 entries) for the Logs tab.
- Subject to file-size rotation with configurable limits.

### `internal/models`

Defines all shared data types:

- `ScannerData` -- a single enriched IP record with 30+ fields.
- `ScannerType` -- string enum for scanner classification.
- `AppConfig` / `DatabaseConfig` -- configuration structures.
- `RDAPCacheEntry` -- cached RDAP/geo result for one IP.
- `RDAPProgressTracker` -- progress state for bulk enrichment.
- `SearchFilter` -- criteria for advanced search.
- `LogLevel` / `LogEntry` -- logging types.

## Data flow

```
┌────────────────────────────────┐
│  internet-scanners repository  │
│  (GitHub, .nft files)          │
└──────────────┬─────────────────┘
               │ git clone / pull
               v
┌────────────────────────────────┐
│  extractor.parseFilesForIPs()  │
│  - Walk .nft files             │
│  - Regex match IPv4/IPv6       │
│  - Deduplicate                 │
└──────────────┬─────────────────┘
               │ []string (unique IPs)
               v
┌────────────────────────────────┐
│  extractor.enrichData()        │
│  - Map IPs to scanner source   │
│  - Create ScannerData records  │
│  - For each IP:                │
│    ├─ Check RDAP cache         │
│    ├─ Query RDAP registries    │
│    ├─ Query ip-api.com         │
│    ├─ Resolve reverse DNS      │
│    └─ Update cache             │
└──────────────┬─────────────────┘
               │ []ScannerData
               v
┌────────────────────────────────┐
│  extractor.SaveToCSV()         │
│  - Write timestamped CSV       │
└──────────────┬─────────────────┘
               │
               v
┌────────────────────────────────┐
│  gui.App                       │
│  - Load CSV into data table    │
│  - Display with pagination     │
│  - Allow further RDAP enrich   │
│  - Search and filter           │
│  - Export results               │
└────────────────────────────────┘
```

## Concurrency model

Bulk RDAP enrichment uses a worker pool pattern:

1. All unprocessed IP indices are pushed into a buffered channel.
2. `N` goroutines (controlled by `parallelism`) consume from the channel.
3. A shared `time.Ticker` (controlled by `api_throttle`) acts as a token bucket, ensuring each worker waits for a tick before making an API call.
4. Progress is saved to disk every 10 records.
5. A cancel flag allows the user to stop enrichment from the GUI.

## External dependencies

| Dependency       | Version | Purpose                             |
|------------------|---------|-------------------------------------|
| `fyne.io/fyne/v2` | 2.4.1 | Cross-platform GUI toolkit          |
| Go standard library | --   | HTTP client, JSON, CSV, regex, etc. |
