#!/bin/bash
if pgrep -af 'etcd --name etcd-dev' >/dev/null; then
#MISE description="Show etcd status"
  pgrep -af 'etcd --name etcd-dev'
  echo "Data directory:"
  ls -la ./testdata/etcd/
else
  echo "etcd is not running"
fi
