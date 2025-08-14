# 📦 Installation Guide

Complete installation guide for **LiaCheckScanner Go** - Professional Internet Scanner Detection Tool.

## 🔧 System Requirements

### Minimum Requirements
- **OS**: Linux (Ubuntu 18.04+), Windows 10+, macOS 10.14+
- **RAM**: 512 MB
- **Storage**: 100 MB free space
- **Network**: Internet connection for data sources

### Recommended Requirements
- **OS**: Linux (Ubuntu 20.04+), Windows 11, macOS 12+
- **RAM**: 2 GB
- **Storage**: 1 GB free space
- **CPU**: Multi-core processor for parallel processing

## 🚀 Quick Installation

### Option 1: Automated Installation (Recommended)

```bash
# Clone the repository
git clone https://github.com/LIA/liacheckscanner_go.git
cd liacheckscanner_go

# Make installation script executable
chmod +x install.sh

# Run automated installation
./install.sh

# Launch the application
./run.sh
```

### Option 2: Manual Installation

```bash
# Prerequisites: Go 1.19+ and Git
go version  # Should be 1.19 or higher
git --version

# Clone and build
git clone https://github.com/LIA/liacheckscanner_go.git
cd liacheckscanner_go

# Install dependencies
go mod download
go mod tidy

# Build the application
go build -o build/liacheckscanner ./cmd/liacheckscanner

# Run the application
./build/liacheckscanner
```

## 🐹 Go Installation

### Linux (Ubuntu/Debian)
```bash
# Remove old Go installation (if any)
sudo rm -rf /usr/local/go

# Download and install Go 1.21
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### macOS
```bash
# Using Homebrew (recommended)
brew install go

# Or download from official site
# https://golang.org/dl/
```

### Windows
1. Download installer from [https://golang.org/dl/](https://golang.org/dl/)
2. Run the installer
3. Restart your terminal
4. Verify: `go version`

## 📋 Platform-Specific Instructions

### Linux

#### Ubuntu/Debian
```bash
# Install prerequisites
sudo apt update
sudo apt install git wget curl

# Install Go (if not installed)
./scripts/install-go-linux.sh

# Build and install
make install
```

#### CentOS/RHEL/Fedora
```bash
# Install prerequisites
sudo yum install git wget curl  # CentOS 7
sudo dnf install git wget curl  # CentOS 8+/Fedora

# Install Go
./scripts/install-go-linux.sh

# Build and install
make install
```

### Windows

#### Using PowerShell
```powershell
# Install Git (if not installed)
winget install Git.Git

# Install Go (if not installed)
winget install GoLang.Go

# Clone and build
git clone https://github.com/LIA/liacheckscanner_go.git
cd liacheckscanner_go
go build -o build/liacheckscanner.exe ./cmd/liacheckscanner

# Run
.\build\liacheckscanner.exe
```

#### Using Command Prompt
```cmd
REM Clone repository
git clone https://github.com/LIA/liacheckscanner_go.git
cd liacheckscanner_go

REM Build application
go build -o build\liacheckscanner.exe .\cmd\liacheckscanner

REM Run application
build\liacheckscanner.exe
```

### macOS

#### Using Terminal
```bash
# Install prerequisites (if using Homebrew)
brew install git go

# Clone and build
git clone https://github.com/LIA/liacheckscanner_go.git
cd liacheckscanner_go

# Build
go build -o build/liacheckscanner ./cmd/liacheckscanner

# Run
./build/liacheckscanner
```

## 🔧 Build Options

### Development Build
```bash
# Standard build
go build -o build/liacheckscanner ./cmd/liacheckscanner

# Build with race detection
go build -race -o build/liacheckscanner-race ./cmd/liacheckscanner

# Build with debug symbols
go build -gcflags="all=-N -l" -o build/liacheckscanner-debug ./cmd/liacheckscanner
```

### Production Build
```bash
# Optimized build (smaller binary)
go build -ldflags="-s -w" -o build/liacheckscanner ./cmd/liacheckscanner

# Build with version information
VERSION=$(git describe --tags --always)
go build -ldflags="-s -w -X main.Version=$VERSION" -o build/liacheckscanner ./cmd/liacheckscanner
```

### Cross-Platform Builds
```bash
# Windows (from Linux/macOS)
GOOS=windows GOARCH=amd64 go build -o build/liacheckscanner.exe ./cmd/liacheckscanner

# macOS (from Linux/Windows)
GOOS=darwin GOARCH=amd64 go build -o build/liacheckscanner-mac ./cmd/liacheckscanner

# Linux (from Windows/macOS)
GOOS=linux GOARCH=amd64 go build -o build/liacheckscanner-linux ./cmd/liacheckscanner

# ARM64 builds
GOOS=linux GOARCH=arm64 go build -o build/liacheckscanner-arm64 ./cmd/liacheckscanner
GOOS=darwin GOARCH=arm64 go build -o build/liacheckscanner-mac-arm64 ./cmd/liacheckscanner
```

## 📁 Directory Structure After Installation

```
liacheckscanner_go/
├── build/
│   └── liacheckscanner*          # Compiled binary
├── config/
│   └── config.json               # Application configuration
├── logs/                         # Application logs (created at runtime)
├── results/                      # Exported data files
├── data/                         # Cache and temporary data
└── internet-scanners/            # Scanner database (auto-downloaded)
```

## ⚙️ Configuration

### Initial Configuration
The application creates a default configuration file on first run:

```json
{
  "app_name": "LiaCheckScanner",
  "version": "1.0.0",
  "theme": "dark",
  "language": "en",
  "database": {
    "repo_url": "https://github.com/MDMCK10/internet-scanners",
    "local_path": "./internet-scanners",
    "results_dir": "./results",
    "logs_dir": "./logs",
    "api_throttle": 1.0,
    "parallelism": 4
  }
}
```

### Custom Configuration
Create a custom configuration:

```bash
# Copy default configuration
cp config/config.json config/my-config.json

# Edit configuration
# Set custom paths, API settings, etc.

# Run with custom config
./build/liacheckscanner -config config/my-config.json
```

## 🧪 Verification

### Test Installation
```bash
# Check if binary runs
./build/liacheckscanner --version

# Run basic tests
go test ./...

# Test specific components
go test ./internal/extractor
go test ./internal/gui
```

### Health Check
```bash
# Check dependencies
go mod verify

# Check for updates
go list -u -m all

# Security audit
go list -json -deps | nancy sleuth
```

## 🐛 Troubleshooting

### Common Issues

#### "go: command not found"
```bash
# Check if Go is in PATH
echo $PATH | grep go

# Add Go to PATH (Linux/macOS)
export PATH=$PATH:/usr/local/go/bin

# Verify Go installation
go version
```

#### "permission denied" (Linux/macOS)
```bash
# Make binary executable
chmod +x build/liacheckscanner

# Or run with bash
bash -c './build/liacheckscanner'
```

#### Build Errors
```bash
# Clean module cache
go clean -modcache

# Re-download dependencies
go mod download

# Rebuild
go build -o build/liacheckscanner ./cmd/liacheckscanner
```

#### GUI Not Showing (Linux)
```bash
# Install required libraries
sudo apt install libgl1-mesa-dev xorg-dev

# For Wayland
export DISPLAY=:0
```

### Log Files
Check log files for detailed error information:
```bash
# View latest logs
tail -f logs/liacheckscanner_$(date +%Y-%m-%d).log

# Search for errors
grep -i error logs/*.log
```

## 🔄 Updates

### Manual Update
```bash
# Pull latest changes
git pull origin main

# Rebuild
go build -o build/liacheckscanner ./cmd/liacheckscanner
```

### Automated Update (planned)
```bash
# Check for updates
./build/liacheckscanner --check-update

# Download and install updates
./build/liacheckscanner --update
```

## 🗑️ Uninstallation

```bash
# Remove application directory
rm -rf liacheckscanner_go/

# Remove configuration (optional)
rm -rf ~/.config/liacheckscanner/

# Remove cached data (optional)
rm -rf ~/.cache/liacheckscanner/
```

## 📞 Support

If you encounter any issues during installation:

1. **Check the logs**: `logs/liacheckscanner_*.log`
2. **Verify requirements**: Go 1.19+, Git, Internet connection
3. **Search existing issues**: [GitHub Issues](https://github.com/LIA/liacheckscanner_go/issues)
4. **Create a new issue**: Include OS, Go version, and error logs
5. **Contact support**: mo0ogly@proton.me

---

**Ready to scan? 🚀** Start with `./build/liacheckscanner` and explore the professional scanner detection capabilities! 