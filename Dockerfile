# Reasonix Hermes — multi-stage Docker build
# Stage 1: build all Hermes binaries from source
# Stage 2: minimal runtime image with just the binaries + ca-certificates

FROM --platform=$BUILDPLATFORM golang:1.26-bookworm AS builder

ARG TARGETOS TARGETARCH

WORKDIR /src

# Cache module downloads in a separate layer
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build all Hermes binaries with symbol stripping (multi-arch)
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix ./cmd/reasonix && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix-mcpbridge ./cmd/reasonix-mcpbridge && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix-memoryserver ./cmd/reasonix-memoryserver && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix-hooks ./cmd/reasonix-hooks && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix-bot ./bot && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix-pr-review ./cmd/reasonix-pr-review && \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -ldflags="-s -w" -o /out/reasonix-e2ebench ./cmd/e2ebench

# --- Runtime image ---
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

# Copy binaries
COPY --from=builder /out/reasonix /usr/local/bin/reasonix
COPY --from=builder /out/reasonix-mcpbridge /usr/local/bin/reasonix-mcpbridge
COPY --from=builder /out/reasonix-memoryserver /usr/local/bin/reasonix-memoryserver
COPY --from=builder /out/reasonix-hooks /usr/local/bin/reasonix-hooks
COPY --from=builder /out/reasonix-bot /usr/local/bin/reasonix-bot
COPY --from=builder /out/reasonix-pr-review /usr/local/bin/reasonix-pr-review
COPY --from=builder /out/reasonix-e2ebench /usr/local/bin/reasonix-e2ebench

# Create workspace mount point
RUN mkdir -p /workspace && chown 65532:65532 /workspace
WORKDIR /workspace

# Health check — reasonix exits 0 on healthy, non-zero on failure.
# Note: distroless static has no shell; binary must be invoked directly.
HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD ["/usr/local/bin/reasonix", "healthcheck"]

# Default entrypoint: CLI. Override with --entrypoint for other binaries.
ENTRYPOINT ["/usr/local/bin/reasonix"]
CMD ["chat"]

# For read-only root filesystem: mount /tmp and /workspace as tmpfs.
# docker run --read-only --tmpfs /tmp --tmpfs /workspace ...

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
