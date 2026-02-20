# Usage

LiaCheckScanner Go is a GUI application. Launch it and interact through the tabbed interface.

## Launching the application

```bash
# From a built binary
./build/liacheckscanner

# Or directly with Go
go run ./cmd/liacheckscanner

# Or via the Makefile
make run
```

On startup the application:

1. Creates all required directories (`logs/`, `results/`, `data/`, `config/`, etc.)
2. Initializes the logger
3. Loads configuration from `config/config.json` (creates a default if missing)
4. Opens the GUI window
5. Attempts to load data from the most recent CSV in `results/`; if none is found, it automatically clones the scanner repository and runs extraction

## GUI Tabs

### Dashboard

The landing tab. It shows:

- **Real-time statistics** -- total records, unique IPs, countries, scanners, high-risk count, and last-updated timestamp.
- **Quick actions** -- buttons for Refresh Data, Export All, and Advanced Search.
- **System information** -- version, owner, platform details.

### Database

The main data view. Key features:

| Control                    | Description                                                                 |
|----------------------------|-----------------------------------------------------------------------------|
| Records per page           | Dropdown to select 25, 50, 100, 250, 500, 1000, or All                     |
| Page navigation            | First / Previous / Next / Last buttons, plus a "Go to page" field          |
| Mettre a jour              | Re-runs extraction (clone + parse + enrich) and reloads the table          |
| Associer RDAP (page)       | Enriches only the IPs visible on the current page via RDAP + geolocation   |
| Associer RDAP (tout)       | Enriches the entire dataset with RDAP data, using parallel workers         |
| Annuler                    | Cancels a running RDAP enrichment                                          |
| RDAP Details               | Shows full RDAP/JSON detail for the selected row                           |
| Geoloc                     | Displays continent-level distribution and allows importing IPs from a file |
| Export All / Export Selected | Saves data to a timestamped CSV in `results/`                             |

!!! info "Resume support"
    If an "Associer RDAP (tout)" operation is interrupted, the next run detects the saved progress file and offers to resume from where it stopped.

### Search

Advanced search and single-IP enrichment:

- **Search field** -- enter an IP, CIDR, scanner name, or country code.
- **Filters** -- narrow by country, scanner type, or risk level.
- **Perform Search** -- filters the loaded dataset.
- **Enrich IP Data** -- runs real-time RDAP + geolocation + reputation lookup for a single IP and displays results in the enrichment pane.
- **Export Results** -- saves current search results to CSV.

### Configuration

Edit application settings without touching JSON files directly:

- Results and logs directories
- Repository URL and local clone path
- RDAP/Geo throttle (in milliseconds)
- Parallelism (number of worker goroutines)
- RDAP registry selection (ARIN, RIPE, APNIC, LACNIC, AFRINIC)

Press **Save Configuration** to persist changes to `config/config.json`.

### Logs

View, filter, and export application logs:

- **Log level filter** -- All, INFO, WARNING, ERROR
- **Refresh Logs** -- reloads the display
- **Export Logs** -- saves logs to a text file
- **Export Logs (ZIP)** -- archives the entire `logs/` directory

## Makefile targets

The Makefile provides convenient shortcuts:

```bash
make help           # List all available targets
make build          # Compile the application
make run            # Build and run
make test           # Run all tests
make test-coverage  # Run tests with HTML coverage report
make lint           # Run golangci-lint
make fmt            # Format code with gofmt
make vet            # Run go vet
make clean          # Remove build artifacts
make deps           # Download and tidy dependencies
make build-all      # Cross-compile for Linux, Windows, macOS
make docker-build   # Build the Docker image
make docker-run     # Run the Docker container
make security       # Run gosec security analysis
make bench          # Run benchmarks
make release        # Clean + cross-compile
make setup          # Create required directories
make docs           # Serve documentation locally
make docs-build     # Build documentation site
```

## Development mode

If [Air](https://github.com/cosmtrek/air) is installed, the `dev` target enables hot reload:

```bash
make dev
```

Otherwise it falls back to `go run`.

## Example workflow

```bash
# 1. Clone and build
git clone https://github.com/mo0ogly/LiaCheckScanner_Go.git
cd LiaCheckScanner_Go
make build

# 2. Launch
make run

# 3. In the GUI:
#    - Go to the Database tab
#    - Click "Associer RDAP (tout)" to enrich all IPs
#    - Wait for completion (progress bar tracks status)
#    - Click "Export All" to save results

# 4. Find the exported CSV
ls results/
```
