#!/bin/bash
# Add some test data to etcd
#MISE description="Add some test data to etcd"
set -euo pipefail
curl -sf http://localhost:2379/health >/dev/null || { echo "etcd is not running. Start it with: mise run etcd-start"; exit 1; }
etcdctl put /test/key1 value1
etcdctl put /test/key2 value2
etcdctl put /app/config/database "postgresql://localhost:5432/mydb"
etcdctl put /app/config/redis "redis://localhost:6379"
echo "Test data in etcd:"
etcdctl get --prefix /
