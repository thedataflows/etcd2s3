# Build stage
FROM golang:1.27.3 AS builder
WORKDIR /app
COPY . .
ARG VERSION=v0.0.0-snapshot
RUN CGO_ENABLED=0 go build -trimpath -buildvcs=false \
    -ldflags="-s -w -X=main.version=${VERSION}" \
    -o /out/etcd2s3 .

# Final stage
FROM alpine:3.24
RUN apk add --no-cache ca-certificates && \
    addgroup -g 1000 etcd2s3 && \
    adduser -D -u 1000 -G etcd2s3 etcd2s3 && \
    mkdir -p /data/snapshots && \
    chown -R etcd2s3:etcd2s3 /data
COPY --from=builder /out/etcd2s3 /usr/local/bin/etcd2s3
USER etcd2s3
WORKDIR /data
ENTRYPOINT ["etcd2s3"]
CMD ["--help"]
