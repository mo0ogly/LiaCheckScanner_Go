# LiaCheckScanner Go

> Scanner IP extractor and RDAP enrichment tool

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/build-passing-brightgreen.svg)](https://github.com/mo0ogly/LiaCheckScanner_Go)

**Owner:** mo0ogly@proton.me

---

## What it does

This tool extracts scanner IP addresses from the [MDMCK10/internet-scanners](https://github.com/MDMCK10/internet-scanners) repository and enriches them with RDAP/WHOIS data.

### Basic functionality
- Clones internet-scanners repository
- Parses .nft files to extract IPv4/IPv6 addresses
- Enriches IPs with RDAP data from major registries
- Displays results in a GUI table
- Exports to CSV

## Quick Start

```bash
git clone https://github.com/mo0ogly/LiaCheckScanner_Go.git
cd LiaCheckScanner_Go
go build -o build/liacheckscanner ./cmd/liacheckscanner
./build/liacheckscanner
```

## Data Sources

### Scanner Repository
- **Source**: [MDMCK10/internet-scanners](https://github.com/MDMCK10/internet-scanners)
- **Format**: Netfilter .nft files
- **Content**: IP addresses and CIDR blocks from various internet scanners

### RDAP Registries
- ARIN (North America)
- RIPE (Europe) 
- APNIC (Asia-Pacific)
- LACNIC (Latin America)
- AFRINIC (Africa)

## Features

- **IP Extraction**: Parses .nft files for IPv4/IPv6 addresses
- **RDAP Enrichment**: Adds WHOIS data (country, ISP, ASN)
- **GUI Interface**: Simple table view with pagination
- **CSV Export**: Basic export functionality
- **Resume Support**: Can resume interrupted RDAP operations

## Installation

### Requirements
- Go 1.19 or later
- Internet connection for cloning repos and RDAP queries

### Build
```bash
# Download dependencies
go mod download

# Build binary
go build -o build/liacheckscanner ./cmd/liacheckscanner

# Run
./build/liacheckscanner
```

## Usage

1. Launch the application
2. It automatically extracts IPs from the internet-scanners repo
3. Use "Associer RDAP" buttons to enrich data
4. Export results as CSV

## Configuration

Edit `config/config.json`:
```json
{
  "database": {
    "repo_url": "https://github.com/MDMCK10/internet-scanners",
    "api_throttle": 1.0,
    "parallelism": 4
  }
}
```

## Architecture

```
cmd/liacheckscanner/     # Main entry point
internal/
├── config/              # Configuration
├── extractor/           # IP extraction and RDAP
├── gui/                 # Fyne GUI
├── logger/              # Logging
└── models/              # Data structures
```

## Development

```bash
# Run tests
go test ./...

# Build for different platforms
GOOS=windows go build -o liacheckscanner.exe ./cmd/liacheckscanner
GOOS=darwin go build -o liacheckscanner-mac ./cmd/liacheckscanner
```

## License

MIT License - see [LICENSE](LICENSE) file.

## Contact

mo0ogly@proton.me 