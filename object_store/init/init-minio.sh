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

if mc ls "local/$BUCKET_NAME" >/dev/null 2>&1; then
  echo "Bucket $BUCKET_NAME already exists."
else
  echo "Creating bucket: $BUCKET_NAME"
  mc mb "local/$BUCKET_NAME"
fi

echo "Ensuring public read access..."
mc anonymous set download "local/$BUCKET_NAME"

# mc ls on a missing prefix exits 0 with empty output (unlike a missing
# bucket, which errors), so presence has to be checked by output, not exit code.
for dir in /seed/*/; do
  name=$(basename "$dir")
  if [ -n "$(mc ls "local/$BUCKET_NAME/$name/" 2>/dev/null)" ]; then
    echo "Prefix '$name/' already present. Skipping."
  else
    echo "Uploading seed prefix: $name"
    mc cp --recursive --quiet "${dir%/}" "local/$BUCKET_NAME/"
  fi
done

echo "MinIO initialization complete"
