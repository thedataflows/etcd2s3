# etcd2s3

A CLI tool for managing etcd snapshots to and from S3.

## Features

- **High-Performance S3 Operations**: Uses s5cmd library internally for efficient S3 uploads/downloads
- **Automatic Snapshot Management**: Create, upload, and manage etcd snapshots
- **Configurable Timeouts**: Set custom timeout values for etcd snapshot operations to prevent hanging
- **Retention Policies**: Configurable retention for both local and S3 stored snapshots
- **Environment Variable Support**: Full configuration via environment variables and CLI flags
- **CLI Interface**: Modern CLI with subcommands using Kong framework

## Usage

### Configuration

All configuration is done via CLI flags, environment variables, and optional `.env` files.

#### Configuration Priority (highest to lowest)

1. CLI flags
2. Environment variables
3. If present in the working directory, `.env, .env.local, .env.development` file values (order from left to right, first found is used)
4. Default values

#### Environment Variables

- Environment variables are automatically loaded and have priority over CLI defaults
- All CLI flags can be set via environment variables using uppercase names with underscores
- Key environment variables:
  - ETCD
    - `ETCD_ENDPOINTS` - etcd endpoints (default: <http://localhost:2379>)
    - `ETCD_SNAPSHOT_DIR` - local snapshot directory (default: /var/lib/etcd/snapshots)
    - `ETCD_SNAPSHOT_TIMEOUT` - timeout for snapshot operations (default: 1m0s)
    - `ETCD_USERNAME` - etcd username for authentication
    - `ETCD_PASSWORD` - etcd password for authentication
    - `ETCD_CERT_FILE` - etcd client certificate file
    - `ETCD_KEY_FILE` - etcd client key file
    - `ETCD_CA_FILE` - etcd CA certificate file
  - S3
    - `AWS_ACCESS_KEY_ID` - S3 access key
    - `AWS_SECRET_ACCESS_KEY` - S3 secret key
    - `AWS_SESSION_TOKEN` - S3 session token (optional)
    - `AWS_REGION` - S3 region (default: us-west-2)
    - `AWS_BUCKET` - S3 bucket name
    - `AWS_ENDPOINT_URL` - Custom S3 endpoint URL
    - `AWS_PREFIX` - S3 key prefix for snapshots (optional)
  - Retention Policy
    - `POLICY_KEEP_LAST` - keep last N snapshots (default: 5)
    - `POLICY_KEEP_LAST_DAYS` - keep snapshots for the last N days (default: 7)
    - `POLICY_KEEP_LAST_HOURS` - keep snapshots for the last N hours (default: 24)
    - `POLICY_KEEP_LAST_WEEKS` - keep snapshots for the last N weeks (default: 4)
    - `POLICY_KEEP_LAST_MONTHS` - keep snapshots for the last N months (default: 3)
    - `POLICY_KEEP_LAST_YEARS` - keep snapshots for the last N years (default: 1)
    - `POLICY_REMOVE_LOCAL` - remove local snapshots after upload to S3
    - `POLICY_TIMEOUT` - timeout for retention operations (default: 5m)

### CLI Commands

**Show help:**

```bash
./etcd2s3 --help
```

### Command Reference

#### Global Flags

- `--log-level` - Log level (trace,debug,info,warn,error), default: 'info'
- `--log-format` - Log format (console,json), default: 'console'

#### etcd Configuration Flags

- `--etcd-endpoints` - etcd endpoints, default: '<http://localhost:2379>'
- `--etcd-snapshot-dir` - Directory to store local snapshots, default: '/var/lib/etcd/snapshots'
- `--etcd-snapshot-timeout` - Timeout for snapshot operations, default: '1m0s'
- `--etcd-username` - etcd username for authentication
- `--etcd-password` - etcd password for authentication
- `--etcd-cert-file` - etcd client certificate file
- `--etcd-key-file` - etcd client key file
- `--etcd-ca-file` - etcd CA certificate file

#### S3 Configuration Flags

- `--aws-region` - S3 region, default: 'us-west-2'
- `--aws-access-key-id` - S3 access key ID
- `--aws-secret-access-key` - S3 secret access key
- `--aws-session-token` - S3 session token
- `--aws-prefix` - S3 key prefix for snapshots
- `--aws-bucket` - S3 bucket name
- `--aws-endpoint-url` - Custom S3 endpoint URL

#### Retention Policy Flags

- `--policy-keep-last` - Keep last N snapshots, default: 5
- `--policy-keep-last-days` - Keep snapshots for the last N days, default: 7
- `--policy-keep-last-hours` - Keep snapshots for the last N hours, default: 24
- `--policy-keep-last-weeks` - Keep snapshots for the last N weeks, default: 4
- `--policy-keep-last-months` - Keep snapshots for the last N months, default: 3
- `--policy-keep-last-years` - Keep snapshots for the last N years, default: 1
- `--policy-remove-local` - Remove local snapshots after upload to S3
- `--policy-timeout` - Timeout for retention operations, default: '5m'

**Take a snapshot:**

```bash
./etcd2s3 snapshot \
  --etcd-endpoints http://localhost:2379 \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots
```

**Take a snapshot with custom options:**

```bash
./etcd2s3 snapshot \
  --etcd-endpoints http://localhost:2379 \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots \
  --name "custom-snapshot-name" \
  --compression zstd \
  --upload-to-s3 \
  --remove-local \
  --apply-retention
```

**Take a snapshot and upload to S3:**

```bash
./etcd2s3 snapshot \
  --etcd-endpoints http://localhost:2379 \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots \
  --aws-region us-west-2 \
  --upload-to-s3 \
  --remove-local
```

**List snapshots:**

```bash
# List all snapshots (local and S3)
./etcd2s3 list \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots

# List only local snapshots
./etcd2s3 list --local \
  --etcd-snapshot-dir /var/lib/etcd/snapshots

# List only S3 snapshots
./etcd2s3 list --remote \
  --aws-bucket my-etcd-snapshots

# Output as JSON
./etcd2s3 list --format=json \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots
```

**Restore from snapshot:**

```bash
# Restore from local snapshot
./etcd2s3 restore /path/to/snapshot.db \
  --data-dir /var/lib/etcd \
  --name default \
  --initial-cluster "default=http://localhost:2380" \
  --initial-advertise-peer-urls "http://localhost:2380"

# Restore from S3 snapshot using s3:// URL
./etcd2s3 restore s3://my-bucket/snapshot.db \
  --data-dir /var/lib/etcd \
  --aws-bucket my-etcd-snapshots \
  --name default \
  --initial-cluster "default=http://localhost:2380" \
  --initial-advertise-peer-urls "http://localhost:2380"

# Restore from S3 snapshot using key name (will auto-download)
./etcd2s3 restore snapshot-20240101-120000.db \
  --data-dir /var/lib/etcd \
  --aws-bucket my-etcd-snapshots \
  --name default \
  --initial-cluster "default=http://localhost:2380" \
  --initial-advertise-peer-urls "http://localhost:2380" \
  --skip-hash-check
```

**Cleanup old snapshots:**

```bash
# Dry run (show what would be deleted)
./etcd2s3 cleanup --dry-run \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots

# Clean local snapshots only
./etcd2s3 cleanup --local \
  --etcd-snapshot-dir /var/lib/etcd/snapshots

# Clean S3 snapshots only
./etcd2s3 cleanup --remote \
  --aws-bucket my-etcd-snapshots

# Use unified retention evaluation (default)
./etcd2s3 cleanup --unified \
  --etcd-snapshot-dir /var/lib/etcd/snapshots \
  --aws-bucket my-etcd-snapshots
```

**Show version:**

```bash
./etcd2s3 version
```

### Command-Specific Flags

#### snapshot command

- `--name` - Custom snapshot name (default: auto-generated with timestamp)
- `--upload-to-s3` - Upload snapshot to S3 (default: true)
- `--remove-local` - Remove local snapshot after S3 upload
- `--apply-retention` - Apply retention policies after snapshot (default: true)
- `--unified` - Use unified retention evaluation across local and S3 (default: true)
- `--compression` - Compression algorithm for snapshot (default: 'zstd', options: none,bzip2,gzip,lz4,zstd)

#### list command

- `--local` - List local snapshots only
- `--remote` - List S3 snapshots only
- `--format` - Output format (table,json,yaml) (default: 'table')
- `--unified` - Use unified retention evaluation across local and S3 (default: true)

#### restore command

- `--data-dir` - etcd data directory for restore (default: '/var/lib/etcd')
- `--name` - etcd member name (default: 'default')
- `--initial-cluster` - Initial cluster configuration (default: 'default=<http://localhost:2380>')
- `--initial-advertise-peer-urls` - Initial advertise peer URLs (default: '<http://localhost:2380>')
- `--skip-hash-check` - Skip hash check during restore

#### cleanup command

- `--local` - Clean local snapshots only
- `--remote` - Clean S3 snapshots only
- `--dry-run` - Show what would be deleted without actually deleting
- `--unified` - Use unified retention evaluation across local and S3 (default: true)

### Authentication

**etcd Authentication:**

```bash
# Basic authentication
./etcd2s3 snapshot \
  --etcd-username "etcd-user" \
  --etcd-password "etcd-password" \
  --etcd-snapshot-timeout "1m0s"

# TLS certificates for secure connections
./etcd2s3 snapshot \
  --etcd-cert-file "/path/to/client.crt" \
  --etcd-key-file "/path/to/client.key" \
  --etcd-ca-file "/path/to/ca.crt"
```

**AWS/S3 Authentication:**

- Use AWS credentials chain (recommended via environment variables)
- Or configure explicitly via CLI flags:

```bash
./etcd2s3 snapshot \
  --aws-access-key-id "YOUR_ACCESS_KEY" \
  --aws-secret-access-key "YOUR_SECRET_KEY" \
  --aws-session-token "OPTIONAL_SESSION_TOKEN"
```

### TLS Configuration

The tool supports secure TLS connections to etcd with the following configurations:

- **CA-only verification**: Provide only `--etcd-ca-file` to verify the etcd server's certificate
- **Mutual TLS**: Provide `--etcd-cert-file`, `--etcd-key-file`, and optionally `--etcd-ca-file` for client certificate authentication
- **Insecure TLS**: Provide `--etcd-cert-file` and `--etcd-key-file` without `--etcd-ca-file` to skip certificate verification (not recommended for production)

```bash
# TLS with CA verification only
./etcd2s3 snapshot \
  --etcd-endpoints "https://etcd.example.com:2379" \
  --etcd-ca-file "/path/to/ca.crt"

# Mutual TLS with client certificates
./etcd2s3 snapshot \
  --etcd-endpoints "https://etcd.example.com:2379" \
  --etcd-cert-file "/path/to/client.crt" \
  --etcd-key-file "/path/to/client.key" \
  --etcd-ca-file "/path/to/ca.crt"
```

## Development

**Build from source:**

```bash
git clone https://github.com/thedataflows/etcd2s3.git
cd etcd2s3
go build .
```

**Development workflow:**

This repo uses [mise-en-place](https://github.com/jdx/mise)

```bash
# Set environment using mise
mise up

# Run in development mode
LOG_LEVEL=debug go run . snapshot \
  --etcd-endpoints http://localhost:2379 \
  --etcd-snapshot-dir ./testdata/snapshots \
  --aws-bucket etcd-snapshots-test

# Build binary
go build -o etcd2s3 .

# Run tests
go test ./...
```

**Dependencies:**

- Go 1.24+
- Uses s5cmd library internally (no external installation required)
- Compatible with etcd v3.6+

## Docker

**Build Docker image:**

```bash
docker build -t etcd2s3 .
```

**Run with Docker:**

```bash
# Run with environment variables
docker run --rm \
  -e ETCD_ENDPOINTS=http://etcd:2379 \
  -e ETCD_SNAPSHOT_DIR=/data/snapshots \
  -e S3_BUCKET=my-etcd-snapshots \
  etcd2s3 snapshot

# Run interactively
docker run --rm -it etcd2s3 --help
```

**Use pre-built image from GitHub Container Registry:**

```bash
docker pull ghcr.io/thedataflows/etcd2s3:latest
docker run --rm ghcr.io/thedataflows/etcd2s3:latest version
```

**Development with Docker Compose:**

```bash
# Start etcd, MinIO, and etcd2s3 services
docker compose up -d

# Wait for services to be healthy, then run etcd2s3 commands
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 snapshot \
  --etcd-endpoints http://etcd:2379 \
  --etcd-snapshot-dir /data/snapshots \
  --aws-bucket etcd-snapshots \
  --aws-endpoint-url http://minio:9000 \
  --aws-access-key-id minioadmin \
  --aws-secret-access-key minioadmin

# List snapshots
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 list \
  --etcd-snapshot-dir /data/snapshots \
  --aws-bucket etcd-snapshots \
  --aws-endpoint-url http://minio:9000 \
  --aws-access-key-id minioadmin \
  --aws-secret-access-key minioadmin

# Restore from snapshot (requires stopping etcd first)
docker compose stop etcd
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 restore <snapshot-name> \
  --data-dir /var/lib/etcd \
  --name etcd-dev \
  --initial-cluster "etcd-dev=http://etcd:2380" \
  --initial-advertise-peer-urls "http://etcd:2380"
# Note: Restore process requires manual data copying to etcd volume
```

**Complete End-to-End Example:**

```bash
# 1. Start all services
docker compose up -d

# 2. Add some test data to etcd
docker exec etcd-dev etcdctl put /test/key1 "value1"
docker exec etcd-dev etcdctl put /test/key2 "value2"

# 3. Create and upload snapshot
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 snapshot \
  --etcd-endpoints http://etcd:2379 \
  --etcd-snapshot-dir /data/snapshots \
  --aws-bucket etcd-snapshots \
  --aws-endpoint-url http://minio:9000 \
  --aws-access-key-id minioadmin \
  --aws-secret-access-key minioadmin

# 4. Verify snapshot was uploaded to MinIO S3
docker exec minio-dev mc ls local/etcd-snapshots/

# 5. Add more data and create another snapshot
docker exec etcd-dev etcdctl put /test/key3 "value3"
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 snapshot \
  --etcd-endpoints http://etcd:2379 \
  --etcd-snapshot-dir /data/snapshots \
  --aws-bucket etcd-snapshots \
  --aws-endpoint-url http://minio:9000 \
  --aws-access-key-id minioadmin \
  --aws-secret-access-key minioadmin

# 6. List all available snapshots
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 list \
  --etcd-snapshot-dir /data/snapshots \
  --aws-bucket etcd-snapshots \
  --aws-endpoint-url http://minio:9000 \
  --aws-access-key-id minioadmin \
  --aws-secret-access-key minioadmin

# 7. Simulate data loss and restore
docker exec etcd-dev etcdctl del /test/key3
docker compose stop etcd

# 8. Restore from the second snapshot (contains key3)
# Note: In production, you would use proper etcd cluster restore procedures
docker exec etcd2s3-dev /usr/local/bin/etcd2s3 restore etcd-snapshot-YYYYMMDD-HHMMSS.db \
  --data-dir /var/lib/etcd \
  --name etcd-dev \
  --initial-cluster "etcd-dev=http://etcd:2380" \
  --initial-advertise-peer-urls "http://etcd:2380" \
  --aws-bucket etcd-snapshots \
  --aws-endpoint-url http://minio:9000 \
  --aws-access-key-id minioadmin \
  --aws-secret-access-key minioadmin

# 9. Copy restored data to etcd volume (development setup)
docker cp etcd2s3-dev:/var/lib/etcd ./restored-etcd-data
docker run --rm -v etcd2s3_etcd-data:/etcd-data --mount type=bind,source=$(pwd)/restored-etcd-data,target=/restore-data --entrypoint /bin/sh alpine:latest -c "rm -rf /etcd-data/* && cp -r /restore-data/* /etcd-data/"
rm -rf restored-etcd-data

# 10. Start etcd with restored data
docker compose start etcd

# 11. Verify data was restored
docker exec etcd-dev etcdctl get /test/ --prefix
```

## Testing

See [testdata/README.md](testdata/README.md) for example test workflows and commands.

## License

[MIT License](LICENSE)
