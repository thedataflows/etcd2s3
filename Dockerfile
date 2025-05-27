# Build stage
FROM goreleaser/goreleaser:v2.9.0 AS builder

# Set working directory
WORKDIR /app

# Copy source code and goreleaser config
COPY . .

# Build the application using goreleaser
# Use --snapshot for local builds without git tags
# Filter for linux/amd64 only to speed up Docker build
ARG VERSION=v0.0.0-snapshot
ENV GORELEASER_CURRENT_TAG=$VERSION
RUN goreleaser build --snapshot --clean --single-target

# Final stage
FROM alpine:3.21

# Install ca-certificates for HTTPS requests
RUN apk add --no-cache ca-certificates

# Create non-root user
RUN addgroup -g 1000 etcd2s3 && \
    adduser -D -s /bin/sh -u 1000 -G etcd2s3 etcd2s3

# Create directories for data and snapshots
RUN mkdir -p /data /data/snapshots && \
    chown -R etcd2s3:etcd2s3 /data

# Copy binary from builder stage
COPY --from=builder /app/dist/default_linux_amd64_v1/etcd2s3 /usr/local/bin/etcd2s3

# Set user
USER etcd2s3

# Set working directory
WORKDIR /data

# Default command
ENTRYPOINT ["etcd2s3"]
CMD ["--help"]
