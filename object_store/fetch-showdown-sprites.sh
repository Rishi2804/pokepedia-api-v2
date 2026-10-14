#!/usr/bin/env bash
# Downloads the Gen 1-4 Pokemon Showdown battle sprites into the MinIO seed
# folder. Gens 5+ already ship in the seed (gen5*, ani*); @pkmn/img asks for
# gen1..gen4 folders (front, -back, -shiny, -back-shiny) when a battle is
# played under those generations, and without them every sprite 404s.
#
# Idempotent: files already on disk are skipped, so it is safe to re-run.
# After running, `docker compose up minio-init` uploads whatever is new.
set -euo pipefail

BASE_URL="https://play.pokemonshowdown.com/sprites"
DEST="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/seed/showdown/sprites"
PARALLEL="${PARALLEL:-8}"

DIRS=(
  gen1 gen1-back
  gen2 gen2-back gen2-shiny gen2-back-shiny
  gen3 gen3-back gen3-shiny gen3-back-shiny
  gen4 gen4-back gen4-shiny gen4-back-shiny
)

download() {
  local dir="$1" file="$2" out="$DEST/$1/$2"
  [ -s "$out" ] && return 0
  curl -sf --retry 3 -o "$out.part" "$BASE_URL/$dir/$file" && mv "$out.part" "$out" \
    || { rm -f "$out.part"; echo "failed: $dir/$file" >&2; }
}
export -f download
export BASE_URL DEST

for dir in "${DIRS[@]}"; do
  mkdir -p "$DEST/$dir"
  # The directory index is an HTML page of <a href="./name.png"> links.
  files=$(curl -sf "$BASE_URL/$dir/" | grep -o 'href="\./[^"/]*\.png"' | sed 's|href="\./||; s/"$//')
  if [ -z "$files" ]; then
    echo "no files listed for $dir" >&2
    exit 1
  fi
  echo "$dir: $(echo "$files" | wc -l | tr -d ' ') files"
  echo "$files" | xargs -P "$PARALLEL" -I{} bash -c 'download "$0" "$1"' "$dir" {}
done

echo "Done. Run 'docker compose up minio-init' to upload them."
