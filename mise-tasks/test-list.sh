#!/bin/bash
# List local and S3 snapshots
#MISE description="List local and S3 snapshots"
set -euo pipefail
curl -sf http://localhost:2379/health >/dev/null || { echo "etcd is not running. Start it with: mise run etcd-start"; exit 1; }
curl -sf http://localhost:9000/health >/dev/null || { echo "rustfs is not running. Start it with: mise run rustfs-start"; exit 1; }
go run . list
