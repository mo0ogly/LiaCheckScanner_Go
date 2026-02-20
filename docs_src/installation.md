# Installation

LiaCheckScanner Go can be installed from source, via `go install`, or run inside a Docker container.

## Requirements

- **Go 1.21** or later
- **Git** (used at runtime to clone/update the scanner repository)
- An internet connection for cloning repositories and querying RDAP endpoints

### Platform-specific dependencies (Fyne / OpenGL)

LiaCheckScanner uses the [Fyne](https://fyne.io/) GUI toolkit, which requires CGO and OpenGL libraries at compile time. Install the platform-specific packages listed below before building.

=== "Ubuntu / Debian"

    ```bash
    sudo apt-get update
    sudo apt-get install -y \
        libgl1-mesa-dev \
        xorg-dev \
        libxcursor-dev \
        libxrandr-dev \
        libxinerama-dev \
        libxi-dev
    ```

=== "Fedora / RHEL"

    ```bash
    sudo dnf install -y \
        mesa-libGL-devel \
        libXcursor-devel \
        libXrandr-devel \
        libXinerama-devel \
        libXi-devel
    ```

=== "macOS"

    Xcode Command Line Tools provide everything Fyne needs:

    ```bash
    xcode-select --install
    ```

=== "Windows"

    Install a C compiler toolchain. The easiest option is [MSYS2](https://www.msys2.org/) with MinGW-w64:

    ```bash
    pacman -S mingw-w64-x86_64-gcc
    ```

    Make sure `gcc` is on your `PATH` before running `go build`.

---

## Install from source (recommended)

```bash
# Clone the repository
git clone https://github.com/mo0ogly/LiaCheckScanner_Go.git
cd LiaCheckScanner_Go

# Download dependencies
go mod download

# Build the binary
go build -ldflags="-s -w" -o build/liacheckscanner ./cmd/liacheckscanner

# Run
./build/liacheckscanner
```

!!! tip
    The Makefile wraps these steps. Run `make build` to compile and `make run` to launch in one command.

## Install with `go install`

If your `GOPATH/bin` is on your `PATH`:

```bash
go install github.com/lia/liacheckscanner_go/cmd/liacheckscanner@latest
liacheckscanner
```

Or use the Makefile target:

```bash
make install
```

## Docker

### Build the image

```bash
docker build -t liacheckscanner:1.0.0 .
```

Or with the Makefile:

```bash
make docker-build
```

### Run the container

```bash
docker run -it --rm liacheckscanner:1.0.0
```

!!! note
    The Docker image includes all OpenGL runtime libraries so the GUI can start inside the container. If you need to display the window on a host X server, pass the appropriate `DISPLAY` and X11 socket volume mounts.

## Cross-compilation

The Makefile provides targets for building platform-specific binaries:

```bash
# Build for all platforms
make build-all

# Or individually
make build-linux    # linux/amd64 + linux/arm64
make build-windows  # windows/amd64
make build-darwin   # darwin/amd64 + darwin/arm64
```

Resulting binaries are placed in the `build/` directory.

## Initial setup

On first run the application creates the following directories automatically:

| Directory              | Purpose                         |
|------------------------|---------------------------------|
| `logs/`                | Log files                       |
| `results/`             | CSV export output               |
| `data/`                | General data storage            |
| `config/`              | Configuration files             |
| `assets/icons/`        | Application icons               |
| `build/`               | Build artifacts                 |
| `build/data/`          | RDAP cache and progress files   |
| `internet-scanners/`   | Cloned scanner repository       |

If the `config/config.json` file does not exist, a default configuration is generated automatically. See [Configuration](configuration.md) for details.

## Verifying the installation

After building, confirm everything works:

```bash
./build/liacheckscanner
```

The GUI window should appear and the application will begin loading data (either from an existing CSV or by running a fresh extraction).
