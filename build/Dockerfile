# =============================================================================
# Builder stage: compile the Go command specified by the CMD build arg.
# Uses the official Go Alpine image for a minimal build environment.
# =============================================================================
ARG GO_VERSION=1.24.0

FROM --platform=$BUILDPLATFORM golang:${GO_VERSION}-alpine AS builder
# CMD determines which CLI entrypoint under cmd/ to compile (e.g. "logistics").
ARG CMD
WORKDIR /src

# Install git (required by some Go modules) and CA certificates so the
# resulting binary can make HTTPS requests at runtime.
RUN apk add --no-cache git ca-certificates && update-ca-certificates

# Download module dependencies first to take advantage of Docker layer caching.
COPY go.mod go.sum ./
RUN go mod download

# Copy the full source tree and build a statically-linked Linux binary for the
# selected command.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/app ./cmd/${CMD}

# =============================================================================
# Runtime stage: minimal scratch image with no OS — only the compiled binary
# and CA certificates are included, keeping the final image as small as possible.
# =============================================================================
FROM scratch
# Bring in trusted root CA certificates so outbound TLS connections work.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
# Copy the statically-linked binary produced by the builder stage.
COPY --from=builder /out/app /app

# Run the binary as the container entrypoint.
ENTRYPOINT ["/app"]
