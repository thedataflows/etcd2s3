#!/bin/bash
set -e

echo "Initializing MinIO bucket..."

# Wait for MinIO to be ready
until mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" > /dev/null 2>&1; do
    echo "Waiting for MinIO to be ready..."
    sleep 2
done

echo "MinIO is ready, setting up bucket..."

# Create bucket if it doesn't exist
if ! mc ls local/etcd-snapshots > /dev/null 2>&1; then
    echo "Creating etcd-snapshots bucket..."
    mc mb local/etcd-snapshots
    echo "Bucket created successfully!"
else
    echo "Bucket etcd-snapshots already exists"
fi

# Set bucket policy (optional - for development)
mc anonymous set download local/etcd-snapshots

echo "MinIO initialization complete!"
