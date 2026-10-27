#!/bin/bash
# Create a snapshot and upload it to S3
#MISE description="Create a snapshot and upload it to S3"
set -euo pipefail
curl -sf http://localhost:2379/health >/dev/null || { echo "etcd is not running. Start it with: mise run etcd-start"; exit 1; }
curl -sf http://localhost:9000/health >/dev/null || { echo "rustfs is not running. Start it with: mise run rustfs-start"; exit 1; }
go run . snapshot
