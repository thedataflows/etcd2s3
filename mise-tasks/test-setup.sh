#!/bin/bash
# Create the test bucket on the local rustfs (waits for it to come up)
#MISE description="Create the test bucket on local rustfs"
set -euo pipefail
export AWS_EC2_METADATA_DISABLED=true
until curl -sf http://localhost:9000/health >/dev/null; do
  echo "Waiting for rustfs on :9000..."
  sleep 1
done
s5cmd --endpoint-url "$AWS_ENDPOINT_URL" ls "s3://$AWS_BUCKET/" >/dev/null 2>&1 ||
  s5cmd --endpoint-url "$AWS_ENDPOINT_URL" mb "s3://$AWS_BUCKET"
echo "rustfs ready at $AWS_ENDPOINT_URL — bucket: $AWS_BUCKET"
