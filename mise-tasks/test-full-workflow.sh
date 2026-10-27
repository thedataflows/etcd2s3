#!/bin/bash
# Populate etcd, snapshot, upload, and list everything
#MISE description="Populate etcd, snapshot, upload, and list everything"
set -euo pipefail
curl -sf http://localhost:2379/health >/dev/null || { echo "etcd is not running. Start it with: mise run etcd-start"; exit 1; }
curl -sf http://localhost:9000/health >/dev/null || { echo "rustfs is not running. Start it with: mise run rustfs-start"; exit 1; }

echo "1. Populating etcd"
mise run test-populate-etcd
echo "2. Snapshot and upload"
go run . snapshot
echo "3. List all"
go run . list
echo "4. List local only"
go run . list --local
echo "5. List S3 only"
go run . list --remote
echo "Full workflow complete"
