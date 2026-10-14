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

# mirror (without --overwrite) uploads only objects that are missing, so a
# volume that was seeded earlier still picks up newly added files (e.g. extra
# sprite folders) instead of the whole prefix being skipped.
for dir in /seed/*/; do
  name=$(basename "$dir")
  echo "Syncing seed prefix: $name"
  mc mirror --quiet "${dir%/}" "local/$BUCKET_NAME/$name"
done

echo "MinIO initialization complete"
