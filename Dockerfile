# Stage 1: Build
FROM golang:1.21-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgl1-mesa-dev xorg-dev \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build \
    -ldflags="-s -w" \
    -o /bin/liacheckscanner \
    ./cmd/liacheckscanner

# Stage 2: Runtime
FROM debian:bookworm-slim

LABEL org.opencontainers.image.title="LiaCheckScanner"
LABEL org.opencontainers.image.description="Scanner IP extractor and RDAP enrichment tool"
LABEL org.opencontainers.image.licenses="MIT"
LABEL org.opencontainers.image.source="https://github.com/mo0ogly/LiaCheckScanner_Go"

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates git libgl1-mesa-glx libx11-6 libxcursor1 libxrandr2 libxinerama1 libxi6 \
    && rm -rf /var/lib/apt/lists/*

RUN groupadd -r appuser && useradd -r -g appuser -m appuser

WORKDIR /app
COPY --from=builder /bin/liacheckscanner .
COPY config/config.json config/config.json
COPY data/ data/

RUN mkdir -p logs results data/internet-scanners && \
    chown -R appuser:appuser /app

USER appuser
ENTRYPOINT ["./liacheckscanner"]
