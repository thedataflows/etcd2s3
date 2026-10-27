#!/bin/bash
pkill -f 'etcd --name etcd-dev' || echo "etcd not running"
#MISE description="Stop etcd process"
