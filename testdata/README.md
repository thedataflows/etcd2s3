# Test Data Directory

This directory contains local test data for etcd2s3 development.

Make sure [mise-en-place](https://github.com/jdx/mise) is available in PATH.

## Directory Structure

- `etcd/` - Local etcd data directory
- `rustfs/` - Local rustfs (S3) data directory
- `snapshots/` - Local snapshot storage directory

## Configuration

Shared config lives in `[env]` in `mise.toml`. The app reads the same
variables through its own flags, so `go run .` tasks need no flags:

- Connect to local etcd at `http://localhost:2379`
- Store snapshots in `./testdata/snapshots`
- Upload to local rustfs bucket `etcd-snapshots`
- S3 endpoint `http://localhost:9000`, credentials `rustfsadmin`/`rustfsadmin`

## Service Endpoints

- **etcd**: <http://localhost:2379>
- **rustfs S3 API**: <http://localhost:9000> (health: `/health`)
- **rustfs Console**: <http://localhost:9001>
  - Username: `rustfsadmin`
  - Password: `rustfsadmin`

## Quick Start

```bash
# Start both etcd and rustfs:
mise run dev-start

# Setup the rustfs bucket:
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

# Start rustfs only
mise run rustfs-start

# Start both services
mise run dev-start

# Stop both services
mise run dev-stop
```

### Testing Tasks

```bash
# Setup rustfs bucket
mise run test-setup

# Populate etcd with test data
mise run test-populate-etcd

# Create a snapshot
mise run test-snapshot

# List all snapshots
mise run test-list

# Run complete test workflow
mise run test-full-workflow

# Restore tests (local file or from S3)
mise run test-restore local
mise run test-restore s3
mise run test-restore-workflow
```

### Monitoring & Maintenance

```bash
# Check status of all services
mise run dev-status

# Show etcd status
mise run logs-etcd

# Show rustfs status
mise run logs-rustfs

# Clean all test data
mise run dev-clean
```

## Example Workflow

```bash
# 1. Start services
mise run dev-start

# 2. Setup rustfs bucket
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

# Check if rustfs is running
curl http://localhost:9000/health

# View status for debugging
mise run logs-etcd
mise run logs-rustfs

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
- rustfs console is available at <http://localhost:9001>
- etcd data persists between restarts
- Use `dev-clean` to reset all test data
- Snapshots are created with minimal retention for testing
