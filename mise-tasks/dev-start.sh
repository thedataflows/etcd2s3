#!/bin/bash
#MISE depends=["etcd-start", "rustfs-start"]
#MISE description="Start both etcd and rustfs for development"
echo "etcd and rustfs are running (stop with: mise run dev-stop)"
