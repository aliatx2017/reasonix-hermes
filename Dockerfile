# Reasonix Hermes — multi-stage Docker build
# Stage 1: build all Hermes binaries from source
# Stage 2: minimal runtime image with just the binaries + ca-certificates

FROM golang:1.25-bookworm AS builder

WORKDIR /src

# Cache module downloads in a separate layer
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all Hermes binaries with symbol stripping
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/reasonix ./cmd/reasonix && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/reasonix-mcpbridge ./cmd/reasonix-mcpbridge && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/reasonix-memoryserver ./cmd/reasonix-memoryserver && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/reasonix-hooks ./cmd/reasonix-hooks && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/reasonix-bot ./bot

# --- Runtime image ---
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Copy binaries
COPY --from=builder /out/reasonix /usr/local/bin/reasonix
COPY --from=builder /out/reasonix-mcpbridge /usr/local/bin/reasonix-mcpbridge
COPY --from=builder /out/reasonix-memoryserver /usr/local/bin/reasonix-memoryserver
COPY --from=builder /out/reasonix-hooks /usr/local/bin/reasonix-hooks
COPY --from=builder /out/reasonix-bot /usr/local/bin/reasonix-bot

# Create workspace mount point
RUN mkdir -p /workspace && chown 65532:65532 /workspace
WORKDIR /workspace

# Default entrypoint: CLI. Override with --entrypoint for other binaries.
ENTRYPOINT ["/usr/local/bin/reasonix"]
CMD ["chat"]

# --- Alternative: slim image with shell (uncomment to use) ---
# FROM debian:bookworm-slim AS runtime-slim
# RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates bash git curl && \
#     rm -rf /var/lib/apt/lists/*
# COPY --from=builder /out/* /usr/local/bin/
# RUN useradd -m -u 1000 reasonix && mkdir -p /workspace && chown reasonix /workspace
# USER reasonix
# WORKDIR /workspace
# ENTRYPOINT ["/usr/local/bin/reasonix"]
# CMD ["chat"]
