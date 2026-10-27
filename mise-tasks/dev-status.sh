#!/bin/bash
echo "=== Development Environment Status ==="
#MISE description="Show status of all development services"
echo -n "etcd: "
curl -sf http://localhost:2379/health >/dev/null && echo "up (http://localhost:2379)" || echo "down"
echo -n "rustfs: "
curl -sf http://localhost:9000/health >/dev/null && echo "up (http://localhost:9000)" || echo "down"
echo -n "rustfs console: "
curl -s -o /dev/null http://localhost:9001/ && echo "up (http://localhost:9001)" || echo "down"
echo
echo "Local snapshots: $(ls -1 ./testdata/snapshots 2>/dev/null | wc -l)"
echo "S3 objects: $(s5cmd --endpoint-url "${AWS_ENDPOINT_URL:-http://localhost:9000}" ls "s3://${AWS_BUCKET:-etcd-snapshots}/" 2>/dev/null | wc -l)"
echo
echo "Data sizes:"
du -sh ./testdata/etcd ./testdata/rustfs ./testdata/snapshots 2>/dev/null || true
