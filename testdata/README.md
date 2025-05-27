# Test Data Directory

This directory contains local test data for etcd2s3 development.

Make sure [mise-en-place](https://github.com/jdx/mise) is available in PATH.

## Directory Structure

- `etcd/` - Local etcd data directory
- `minio/` - Local MinIO data directory
- `snapshots/` - Local snapshot storage directory

## Configuration

The test environment is configured via CLI flags and environment variables:

- Connect to local etcd at `localhost:2379`
- Store snapshots in `./testdata/snapshots`
- Upload to local MinIO bucket `etcd-snapshots-test`
- Use minimal retention policy for testing

## Service Endpoints

- **etcd**: <http://localhost:2379>
- **MinIO API**: <http://localhost:9000>
- **MinIO Console**: <http://localhost:9001>
  - Username: `minioadmin`
  - Password: `minioadmin`

## Quick Start

```bash
# Start both etcd and MinIO:
mise run dev-start

# Setup MinIO bucket:
mise run test-setup

# Create a test snapshot:
mise run test-snapshot

# Stop services:
mise run dev-stop
```

## All Available Tasks

### Core Services

```bash
# Start etcd only
mise run etcd-start

# Start MinIO only
mise run minio-start

# Start both services
mise run dev-start

# Stop both services
mise run dev-stop
```

### Testing Tasks

```bash
# Setup MinIO bucket
mise run test-setup

# Populate etcd with test data
mise run test-populate-etcd

# Create a snapshot
mise run test-snapshot

# List all snapshots
mise run test-list

# Run complete test workflow
mise run test-full-workflow
```

### Monitoring & Maintenance

```bash
# Check status of all services
mise run dev-status

# Show etcd logs
mise run logs-etcd

# Show MinIO logs
mise run logs-minio

# Clean all test data
mise run dev-clean
```

## Example Workflow

```bash
# 1. Start services
mise run dev-start

# 2. Setup MinIO bucket
mise run test-setup

# 3. Add test data to etcd
mise run test-populate-etcd

# 4. Create snapshots
mise run test-snapshot

# 5. List snapshots (local and S3)
mise run test-list

# 6. Check service status
mise run dev-status

# 7. Clean up when done
mise run dev-stop
mise run dev-clean
```

## Troubleshooting

If services aren't working:

```bash
# Check if etcd is running
curl http://localhost:2379/health

# Check if MinIO is running
curl http://localhost:9000/minio/health/ready

# View logs for debugging
mise run logs-etcd
mise run logs-minio

# Restart services
mise run dev-stop
mise run dev-start

# Complete reset
mise run dev-clean
mise run dev-start
mise run test-setup
```

## Notes

- All data is stored in `./testdata/` directory
- MinIO console is available at <http://localhost:9001>
- etcd data persists between restarts
- Use `dev-clean` to reset all test data
- Snapshots are created with minimal retention for testing
