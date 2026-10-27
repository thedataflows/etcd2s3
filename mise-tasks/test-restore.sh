#!/bin/bash
# Restore test — usage: mise run test-restore [local|s3]
#MISE description="Restore test — usage: mise run test-restore [local|s3]"
set -euo pipefail
source_name="${1:-local}"
case "$source_name" in
  local) src="$ETCD_SNAPSHOT_DIR/test-restore-snapshot.db" ;;
  s3) src="s3://$AWS_BUCKET/test-restore-snapshot.db" ;;
  *) echo "usage: mise run test-restore [local|s3]"; exit 1 ;;
esac

if [ "$source_name" = s3 ]; then
  curl -sf http://localhost:9000/health >/dev/null || { echo "rustfs is not running. Start it with: mise run rustfs-start"; exit 1; }
  s5cmd --endpoint-url "$AWS_ENDPOINT_URL" ls "s3://$AWS_BUCKET/" | grep -q test-restore-snapshot || {
    echo "Snapshot not in S3 yet — run: mise run test-create-restore-snapshot && mise run test-snapshot"
    exit 1
  }
elif [ ! -e "$ETCD_SNAPSHOT_DIR/test-restore-snapshot.db" ] && [ ! -e "$ETCD_SNAPSHOT_DIR/test-restore-snapshot.db.zst" ]; then
  echo "Fixture snapshot missing — creating it first"
  mise run test-create-restore-snapshot
fi

pkill -f 'etcd --name etcd-dev' 2>/dev/null || true
# Wait for etcd to fully exit so the ports are free again
for _ in $(seq 1 20); do
  pgrep -f 'etcd --name etcd-dev' >/dev/null || break
  sleep 0.5
done
pkill -9 -f 'etcd --name etcd-dev' 2>/dev/null || true
rm -rf ./testdata/etcd/*
restore_dir="./testdata/etcd-restore-$source_name"

echo "Restoring from $src"
go run . restore "$src" --data-dir "$restore_dir"

etcd \
  --name etcd-dev \
  --data-dir "$restore_dir" \
  --listen-client-urls http://localhost:2379 \
  --advertise-client-urls http://localhost:2379 \
  --listen-peer-urls http://localhost:2380 \
  --initial-advertise-peer-urls http://localhost:2380 \
  --initial-cluster "etcd-dev=http://localhost:2380" \
  --initial-cluster-token etcd-cluster-dev \
  --initial-cluster-state new \
  --log-level warn &
for _ in $(seq 1 60); do curl -sf http://localhost:2379/health >/dev/null && break; sleep 1; done

echo "Restored etcd data (post-snapshot key must be gone):"
etcdctl get --prefix /
pkill -f 'etcd --name etcd-dev' 2>/dev/null || true
rm -rf "$restore_dir"
echo "Restore test ($source_name) complete"
