# LiaCheckScanner Go

> Scanner IP extractor and RDAP enrichment tool

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://github.com/mo0ogly/LiaCheckScanner_Go/blob/main/LICENSE)
[![CI](https://github.com/mo0ogly/LiaCheckScanner_Go/actions/workflows/ci.yml/badge.svg)](https://github.com/mo0ogly/LiaCheckScanner_Go/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/mo0ogly/LiaCheckScanner_Go/branch/main/graph/badge.svg)](https://codecov.io/gh/mo0ogly/LiaCheckScanner_Go)

**Owner:** mo0ogly@proton.me

---

## What it does

LiaCheckScanner Go extracts scanner IP addresses from the [MDMCK10/internet-scanners](https://github.com/MDMCK10/internet-scanners) repository and enriches them with RDAP/WHOIS data from all five major Regional Internet Registries.

### Basic functionality

- Clones the internet-scanners repository
- Parses `.nft` (Netfilter) files to extract IPv4/IPv6 addresses and CIDR blocks
- Enriches IPs with RDAP data from ARIN, RIPE, APNIC, LACNIC, and AFRINIC
- Displays results in a multi-tab GUI with pagination
- Exports enriched data to CSV

## Features

- **IP Extraction** -- Parses `.nft` files for IPv4 and IPv6 addresses, including CIDR notation
- **RDAP Enrichment** -- Adds WHOIS-style data (country, ISP, ASN, organization, abuse contacts)
- **GUI Interface** -- Fyne-based interface with Dashboard, Database, Search, Configuration, and Logs tabs
- **CSV Export** -- Full metadata export with timestamped filenames
- **Resume Support** -- Can resume interrupted RDAP enrichment operations using on-disk progress tracking
- **Parallel Processing** -- Configurable worker pools with token-bucket rate limiting
- **RDAP Caching** -- On-disk JSON cache to avoid redundant API calls

## Data Sources

### Scanner Repository

| Field     | Value                                                                      |
|-----------|----------------------------------------------------------------------------|
| Source    | [MDMCK10/internet-scanners](https://github.com/MDMCK10/internet-scanners) |
| Format    | Netfilter `.nft` files                                                     |
| Content   | IP addresses and CIDR blocks from 50+ internet scanners                    |

### RDAP Registries

| Registry | Region               |
|----------|----------------------|
| ARIN     | North America        |
| RIPE     | Europe               |
| APNIC    | Asia-Pacific         |
| LACNIC   | Latin America        |
| AFRINIC  | Africa               |

## Quick Start

```bash
git clone https://github.com/mo0ogly/LiaCheckScanner_Go.git
cd LiaCheckScanner_Go
go build -o build/liacheckscanner ./cmd/liacheckscanner
./build/liacheckscanner
```

See [Installation](installation.md) for detailed setup instructions and [Usage](usage.md) for a complete guide.

## License

MIT License -- see the [LICENSE](https://github.com/mo0ogly/LiaCheckScanner_Go/blob/main/LICENSE) file.

## Contact

mo0ogly@proton.me
