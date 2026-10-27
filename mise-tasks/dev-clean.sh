#!/bin/bash
#MISE depends=["dev-stop"]
#MISE description="Clean all test data"
set -euo pipefail
rm -rf ./testdata/etcd* ./testdata/rustfs ./testdata/snapshots
echo "Test data cleaned"
