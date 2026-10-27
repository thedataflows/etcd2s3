#!/bin/bash
# Start local etcd for testing (data in ./testdata/etcd)
#MISE description="Start local etcd for testing"
set -euo pipefail
mkdir -p ./testdata/etcd
exec etcd \
  --name etcd-dev \
  --data-dir ./testdata/etcd \
  --listen-client-urls http://localhost:2379 \
  --advertise-client-urls http://localhost:2379 \
  --listen-peer-urls http://localhost:2380 \
  --initial-advertise-peer-urls http://localhost:2380 \
  --initial-cluster "etcd-dev=http://localhost:2380" \
  --initial-cluster-token etcd-cluster-dev \
  --initial-cluster-state new \
  --log-level info
