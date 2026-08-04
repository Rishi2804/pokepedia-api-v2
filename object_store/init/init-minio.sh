#!/bin/sh
set -e

echo "Waiting for MinIO to be ready..."

until mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD"
do
  echo "MinIO not ready yet, sleeping 2s..."
  sleep 2
done

echo "MinIO is ready"

BUCKET_NAME="pokemon-images"

# If bucket already exists → assume everything is initialized
if mc ls local/$BUCKET_NAME >/dev/null 2>&1; then
  echo "Bucket already exists. Skipping initialization."
  exit 0
fi

echo "Creating bucket: $BUCKET_NAME"
mc mb local/$BUCKET_NAME

echo "Uploading seed images..."
mc cp --recursive --quiet /seed/* local/$BUCKET_NAME

echo "Setting public read access..."
mc anonymous set download local/$BUCKET_NAME

echo "MinIO initialization complete"