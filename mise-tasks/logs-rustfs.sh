#!/bin/bash
if pgrep -af 'rustfs server' >/dev/null; then
#MISE description="Show rustfs status"
  pgrep -af 'rustfs server'
  echo "Data directory:"
  ls -la ./testdata/rustfs/
else
  echo "rustfs is not running"
fi
