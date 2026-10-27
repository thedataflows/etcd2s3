#!/bin/bash
# Create a snapshot with known data, then change data to prove a restore works
#MISE description="Create a test snapshot with known data for restore testing"
set -euo pipefail
curl -sf http://localhost:2379/health >/dev/null || { echo "etcd is not running. Start it with: mise run etcd-start"; exit 1; }
etcdctl del --prefix / >/dev/null
etcdctl put /restore-test/timestamp "$(date)"
etcdctl put /restore-test/data1 original-value-1
etcdctl put /restore-test/data2 original-value-2
etcdctl put /app/restore-config restore-test-config
go run . snapshot --name test-restore-snapshot
etcdctl put /post-snapshot/key this-should-disappear-after-restore
etcdctl put /restore-test/data1 modified-value-1
echo "Post-snapshot etcd data:"
etcdctl get --prefix /
