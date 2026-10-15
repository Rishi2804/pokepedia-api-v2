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
#
# items/ is the exception: fetch-item-images.sh upgrades images in place (a
# 30px sprite replaced by a larger Bulbapedia one), and without --overwrite
# mc refuses to replace the object with a different size, errors out, and
# under `set -e` aborts before the prefixes after it are synced. --overwrite
# only re-uploads objects that differ, so unchanged files still cost nothing.
for dir in /seed/*/; do
  name=$(basename "$dir")
  echo "Syncing seed prefix: $name"
  if [ "$name" = "items" ]; then
    mc mirror --quiet --overwrite "${dir%/}" "local/$BUCKET_NAME/$name"
  else
    mc mirror --quiet "${dir%/}" "local/$BUCKET_NAME/$name"
  fi
done

echo "MinIO initialization complete"
