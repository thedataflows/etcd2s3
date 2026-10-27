#!/bin/bash
# Complete restore testing workflow
#MISE description="Complete restore testing workflow"
set -euo pipefail
curl -sf http://localhost:2379/health >/dev/null || { echo "Starting etcd..."; mise run etcd-start & sleep 5; }
curl -sf http://localhost:9000/health >/dev/null || { echo "Starting rustfs..."; mise run rustfs-start & sleep 5; }
mise run test-setup
mise run test-create-restore-snapshot
mise run test-restore local
mise run test-restore s3
mise run dev-stop
echo "Restore testing complete"
