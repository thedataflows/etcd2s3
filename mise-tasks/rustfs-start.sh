#!/bin/bash
# Start local rustfs (S3-compatible) for testing (data in ./testdata/rustfs)
#MISE description="Start local rustfs (S3) for testing"
set -euo pipefail
mkdir -p ./testdata/rustfs
exec rustfs server \
  --address :9000 \
  --console-enable \
  --console-address :9001 \
  --access-key rustfsadmin \
  --secret-key rustfsadmin \
  ./testdata/rustfs
